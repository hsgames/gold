package main

import (
	"flag"
	"github.com/hsgames/gold/app"
	"github.com/hsgames/gold/log"
	"github.com/hsgames/gold/net"
	"github.com/hsgames/gold/net/tcp"
	"github.com/hsgames/gold/safe"
)

type Handler struct {
	logger log.Logger
}

func NewHandler(logger log.Logger) net.Handler {
	return &Handler{logger: logger}
}

func (h *Handler) OnOpen(conn net.Conn) error {
	h.logger.Info("main: conn %s open", conn)
	data := make([]byte, 4)
	for i := 0; i < 10; i++ {
		conn.Write(data)
	}
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

func main() {
	flag.Parse()
	logger := log.NewFileLogger(log.InfoLevel, "./log", log.FileAlsoStderr(true))
	defer logger.Shutdown()
	defer safe.Recover(logger)
	clients := make([]*tcp.Client, 1)
	for i := range clients {
		clients[i] = tcp.NewClient("tcp_client", "tcp", "127.0.0.1:18888",
			tcp.DefaultParser, NewHandler, logger,
		)
	}
	a := app.New(logger)
	a.AddPProf("pprof", "tcp", ":9999")
	for _, v := range clients {
		c := v
		a.AddService(
			func() error {
				err := c.Dial()
				if err != nil {
					return err
				}
				logger.Info("main: tcp client %s shutdown", c)
				return nil
			},
			func() error {
				c.Shutdown()
				return nil
			},
		)
	}
	if err := a.Run(); err != nil {
		logger.Error("main: app run err: %+v", err)
	}
	logger.Info("main: tcp client exit")
}
