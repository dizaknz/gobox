package service

import (
	"errors"
	"os"
	"os/signal"
	"syscall"

	"github.com/oklog/oklog/pkg/group"
	"github.com/rs/zerolog"
)

var ErrShutdownSignal = errors.New("shutdown signal received")

type Service interface {
	Name() string
	Start() error
	Close() error
}

type ServiceGroup struct {
	*group.Group
	logger zerolog.Logger
}

func NewServiceGroup(logger zerolog.Logger) *ServiceGroup {
	return &ServiceGroup{
		Group:  &group.Group{},
		logger: logger,
	}
}

func (sg *ServiceGroup) Register(svc Service) {
	sg.Add(func() error {
		sg.logger.Info().Str("service", svc.Name()).Msg("Starting service")
		return svc.Start()
	}, func(err error) {
		if err != nil && err != ErrShutdownSignal {
			sg.logger.Error().
				Str("service", svc.Name()).
				Str("err", err.Error()).
				Msg("Failed to start service")
			return
		}
		sg.logger.Info().Str("service", svc.Name()).Msg("Shutting down service")
		svc.Close()
	})
}

func (sg *ServiceGroup) registerInterrupt() {
	cancelInterrupt := make(chan struct{})
	sg.Add(func() error {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		select {
		case sig := <-c:
			sg.logger.Info().Str("signal", sig.String()).Msg("Received signal")
			return ErrShutdownSignal
		case <-cancelInterrupt:
			return nil
		}
	}, func(error) {
		close(cancelInterrupt)
	})
}

func (sg *ServiceGroup) Run() error {
	sg.registerInterrupt()

	return sg.Group.Run()
}
