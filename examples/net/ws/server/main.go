package main

import (
	"flag"
	"fmt"
	"github.com/hsgames/gold/app"
	"github.com/hsgames/gold/log"
	"github.com/hsgames/gold/net"
	"github.com/hsgames/gold/net/ws"
	"github.com/hsgames/gold/safe"
	"time"
)

type Handler struct {
	logger log.Logger
}

func NewHandler(logger log.Logger) net.Handler {
	return &Handler{logger: logger}
}

func (h *Handler) OnOpen(conn net.Conn) error {
	h.logger.Info("main: conn %s open", conn)
	return nil
}

func (h *Handler) OnClose(conn net.Conn) error {
	h.logger.Info("main: conn %s close", conn)
	return nil
}

func (h *Handler) OnMessage(conn net.Conn, data []byte) error {
	conn.Write(data)
	return nil
}

func showServerConnNum(s *ws.Server) {
	ticker := time.NewTicker(3 * time.Second)
	for {
		select {
		case <-ticker.C:
			fmt.Println(s.ConnNum())
		}
	}
}

func main() {
	flag.Parse()
	logger := log.NewFileLogger(log.InfoLevel, "./log", log.FileAlsoStderr(true))
	defer logger.Shutdown()
	defer safe.Recover(logger)
	s := ws.NewServer("ws_server", "tcp", ":8080",
		NewHandler, logger,
		ws.ServerMsgType(ws.TextMessage),
	)
	safe.Go(logger, func() { showServerConnNum(s) })
	a := app.New(logger)
	a.AddPProf("pprof", "tcp", ":8888")
	a.AddService(
		func() error {
			err := s.Listen()
			if err != nil {
				return err
			}
			logger.Info("main: ws server %s listen", s)
			err = s.Serve()
			if err != nil {
				return err
			}
			logger.Info("main: ws server %s shutdown", s)
			return nil
		},
		func() error {
			s.Shutdown()
			return nil
		},
	)
	if err := a.Run(); err != nil {
		logger.Error("main: app run err: %+v", err)
	}
	logger.Info("main: ws server exit")
}
