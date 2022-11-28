package service

import (
	"encoding/json"
	"io"
	"net/url"

	"github.com/gorilla/websocket"
	webrtc "github.com/pion/webrtc/v3"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type OfferMessage = webrtc.SessionDescription
type AnswerMessage = webrtc.SessionDescription
type CandidateMessage struct {
	Type      string                  `json:"type"`
	Candidate webrtc.ICECandidateInit `json:"candidate"`
}

type WebRTCService struct {
	api           *webrtc.API
	peer          *webrtc.PeerConnection
	engine        *webrtc.MediaEngine
	messagingAddr string
	mediaAddr     string
	done          chan struct{}
	stopped       bool
	logger        zerolog.Logger
}

func NewWebRTCService(
	messagingAddr string,
	mediaAddr string,
	logger zerolog.Logger,
) *WebRTCService {
	return &WebRTCService{
		messagingAddr: messagingAddr,
		mediaAddr:     mediaAddr,
		done:          make(chan struct{}),
		logger:        logger,
	}
}

func (s *WebRTCService) initPeerConnection(ws *websocket.Conn) error {
	s.logger.Info().Msg("Initialising peer connection")
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
		SDPSemantics: webrtc.SDPSemanticsUnifiedPlanWithFallback,
	}
	s.engine = &webrtc.MediaEngine{}
	if err := s.engine.RegisterCodec(
		webrtc.RTPCodecParameters{
			RTPCodecCapability: webrtc.RTPCodecCapability{
				MimeType:     webrtc.MimeTypeH264,
				ClockRate:    90000,
				Channels:     0,
				SDPFmtpLine:  "level-asymmetry-allowed=1;packetization-mode=1;profile-level-id=42e034",
				RTCPFeedback: nil,
			},
			PayloadType: 96,
		},
		webrtc.RTPCodecTypeVideo,
	); err != nil {
		return err
	}
	if err := s.engine.RegisterCodec(
		webrtc.RTPCodecParameters{
			RTPCodecCapability: webrtc.RTPCodecCapability{
				MimeType:     webrtc.MimeTypeOpus,
				ClockRate:    48000,
				Channels:     2,
				SDPFmtpLine:  "minptime=10;useinbandfec=1",
				RTCPFeedback: nil,
			},
			PayloadType: 111,
		},
		webrtc.RTPCodecTypeAudio,
	); err != nil {
		return err
	}
	s.api = webrtc.NewAPI(webrtc.WithMediaEngine(s.engine))
	var err error
	s.peer, err = s.api.NewPeerConnection(config)
	if err != nil {
		return err
	}

	if _, err = s.peer.AddTransceiverFromKind(webrtc.RTPCodecTypeAudio); err != nil {
		return err
	}
	if _, err = s.peer.AddTransceiverFromKind(webrtc.RTPCodecTypeVideo); err != nil {
		return err
	}
	// PSP data channel
	if _, err = s.peer.CreateDataChannel("cirrus", nil); err != nil {
		return err
	}

	s.peer.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		// TODO: consume media
	})

	// On ICECandidate event
	s.peer.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		s.logger.Info().Msg("OnICECandidate")
		if candidate != nil {
			body, err := json.Marshal(candidate.ToJSON())
			if err != nil {
				s.logger.Error().
					Str("err", err.Error()).
					Msg("Failed to encode candidate payload")
				return
			}
			s.logger.Debug().Msg("Sending ICE candidate message")
			ws.WriteMessage(websocket.TextMessage, body)
		}
	})

	s.peer.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		s.logger.Debug().
			Str("state", connectionState.String()).
			Msg("Connection state has changed")
	})

	return nil
}

func (s *WebRTCService) Start() error {
	u := url.URL{
		Scheme: "ws",
		Host:   s.messagingAddr,
		Path:   "/",
	}
	s.logger.Info().Str("ws", u.String()).Msg("Connecting to messaging websocket")
	ws, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		s.logger.Error().
			Str("addr", s.messagingAddr).
			Str("err", err.Error()).
			Msg("Failed to connect to messaging websocket server")
		return err
	}
	defer ws.Close()

	go s.processMessages(ws)

	// wait for message ws disconnect or service stopped
	<-s.done

	return nil
}

func (s *WebRTCService) processMessages(ws *websocket.Conn) {
	defer func() {
		if s.stopped {
			return
		}
		s.stopped = true
		close(s.done)
	}()
	for {
		_, msg, err := ws.ReadMessage()
		if err != nil || err == io.EOF {
			s.logger.Error().
				Str("err", err.Error()).
				Msg("Failed to reading messages")
			break
		}
		log.Debug().Str("msg", string(msg)).Msg("Received message")
		raw := map[string]interface{}{}
		if err := json.Unmarshal(msg, &raw); err != nil {
			s.logger.Error().
				Str("err", err.Error()).
				Msg("Failed to decode message")
			return
		}
		switch raw["type"] {
		case "ping":
			// TODO
		case "offer":
			err = s.handleOffer(msg, ws)
		case "icecandidate":
			err = s.handleCandidate(msg)
		}
		if err != nil {
			s.logger.Error().
				Str("msg", string(msg)).
				Str("err", err.Error()).
				Msg("Failed to process message")
			return
		}
	}
}

func (s *WebRTCService) handleOffer(msg []byte, ws *websocket.Conn) error {

	offer := OfferMessage{}
	if err := json.Unmarshal(msg, &offer); err != nil {
		return err
	}
	// init peer connection
	if err := s.initPeerConnection(ws); err != nil {
		s.logger.Error().
			Str("err", err.Error()).
			Msg("Failed to initialise peer connection")
		return err
	}
	if err := s.peer.SetRemoteDescription(offer); err != nil {
		return err
	}

	// send answer
	answer, err := s.peer.CreateAnswer(nil)
	if err != nil {
		return err
	}
	body, err := json.Marshal(answer)
	if err != nil {
		return err
	}
	s.logger.Debug().Msg("Sending answer message")
	ws.WriteMessage(websocket.TextMessage, body)
	return nil
}

func (s *WebRTCService) handleCandidate(msg []byte) error {
	ice := CandidateMessage{}
	if err := json.Unmarshal(msg, &ice); err != nil {
		return err
	}
	return s.peer.AddICECandidate(ice.Candidate)
}

func (s *WebRTCService) Close() error {
	if s.stopped {
		return nil
	}
	close(s.done)
	return nil
}

func (s *WebRTCService) Name() string {
	return "pion webrtc service"
}
