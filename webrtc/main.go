package main

import (
	"flag"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"webrtc/service"
)

const app = "webrtc"

func main() {
	cli()

	setLogLevel(viper.GetString("log_level"))

	messagingAddr := viper.GetString("messaging_addr")
	mediaAddr := viper.GetString("media_addr")
	webrtc := service.NewWebRTCService(
		messagingAddr,
		mediaAddr,
		log.With().Logger(),
	)

	services := service.NewServiceGroup(log.With().Logger())
	services.Register(webrtc)

	log.Info().Str("app", app).Msg("Starting")
	if err := services.Run(); err != nil && err != service.ErrShutdownSignal {
		log.Error().Str("err", err.Error()).Msg("Failed to start services")
		os.Exit(1)
	}
	log.Info().Str("app", app).Msg("Shutdown")
}

func setLogLevel(level string) {
	switch level {
	case "info":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "warn":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	default:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}
}

func cli() {
	viper.SetEnvPrefix(app)
	fs := flag.NewFlagSet(app, flag.ExitOnError)

	fs.String("messaging_addr", "127.0.0.1:8091", "Websocket for consuming SDP messaging")
	fs.String("media_addr", "127.0.0.1:8090", "Websocket for consuming media")
	fs.String("log_level", "info", "Set the log level, eg. info, warn, debug or error")

	envs := []string{
		"messaging_addr",
		"media_addr",
		"log_level",
	}

	for _, env := range envs {
		viper.BindEnv(env)
	}

	pflag.CommandLine.AddGoFlagSet(fs)
	pflag.Parse()
	viper.BindPFlags(pflag.CommandLine)
}
