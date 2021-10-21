package app_test

import (
	"github.com/hsgames/gold/app"
	"github.com/hsgames/gold/log"
	"testing"
)

type Server struct {
	logger log.Logger
}

func NewServer(logger log.Logger) app.Framework {
	return &Server{
		logger: logger,
	}
}

func (s *Server) Init() error {
	s.logger.Info("app_test: server init")
	return nil
}

func (s *Server) Run() error {
	s.logger.Info("app_test: server run")
	return nil
}

func (s *Server) Destroy() error {
	s.logger.Info("app_test: server destroy")
	return nil
}

func TestRunFramework(t *testing.T) {
	app.RunFramework(NewServer, app.DefaultStdLogger())
}
