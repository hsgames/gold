package ws

import (
	"context"
	"fmt"
	"github.com/gorilla/websocket"
	gnet "github.com/hsgames/gold/net"
	"github.com/hsgames/gold/net/tcp"
	"github.com/hsgames/gold/safe"
	"log/slog"
	"net/http"
	"sync"
	"sync/atomic"
)

type Server struct {
	*http.Server

	opts       options
	name       string
	newHandler func() gnet.Handler
	upgrader   *websocket.Upgrader
	connsMu    sync.Mutex
	conns      map[*Conn]struct{}
	connId     uint64
}

func NewServer(name, addr string,
	newHandler func() gnet.Handler, opt ...Option) (s *Server, err error) {

	opts := defaultOptions()
	for _, o := range opt {
		o(&opts)
	}

	if err = opts.check(); err != nil {
		return
	}

	s = &Server{
		opts:       opts,
		name:       name,
		newHandler: newHandler,
		upgrader: &websocket.Upgrader{
			HandshakeTimeout: opts.handshakeTimeout,
			CheckOrigin:      func(_ *http.Request) bool { return opts.checkOrigin },
		},
		conns: make(map[*Conn]struct{}),
	}

	mux := http.NewServeMux()
	mux.HandleFunc(s.opts.pattern, s.serve)

	s.Server = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  s.opts.readTimeout,
		WriteTimeout: s.opts.writeTimeout,
	}

	return
}

func (s *Server) String() string {
	return fmt.Sprintf("[name:%s][listen_addr:%s]", s.Name(), s.Addr())
}

func (s *Server) Name() string {
	return s.name
}

func (s *Server) Addr() string {
	return s.Server.Addr
}

func (s *Server) ConnNum() int {
	s.connsMu.Lock()
	defer s.connsMu.Unlock()
	return len(s.conns)
}

func (s *Server) Shutdown(ctx context.Context) {
	if err := s.Server.Shutdown(ctx); err != nil {
		slog.Error("ws: server shutdown",
			slog.String("server", s.String()), slog.Any("error", err))
	}

	s.connsMu.Lock()
	defer s.connsMu.Unlock()
	defer clear(s.conns)

	for c := range s.conns {
		c.Shutdown()
	}

	for c := range s.conns {
		select {
		case <-c.Done():
		case <-ctx.Done():
			return
		}
	}
}

func (s *Server) serve(w http.ResponseWriter, r *http.Request) {
	defer safe.Recover()

	if r.Method != "GET" {
		slog.Error("ws: server method not allowed",
			slog.String("server", s.String()), slog.String("method", r.Method))

		http.Error(w, "Method not allowed", 405)

		return
	}

	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("ws: server %s upgrade",
			slog.String("server", s.String()), slog.Any("error", err))

		return
	}

	s.connsMu.Lock()
	if s.opts.maxConnNum > 0 && len(s.conns) >= s.opts.maxConnNum {
		s.connsMu.Unlock()

		slog.Info("ws: server accept too many conns",
			slog.String("server", s.String()))

		if err := conn.Close(); err != nil {
			slog.Error("ws: server close overflow conn",
				slog.String("server", s.String()), slog.Any("error", err))
		}

		return
	}

	if err = tcp.SetConnOptions(conn.NetConn(), s.opts.keepAlivePeriod); err != nil {
		s.connsMu.Unlock()

		slog.Error("ws: server set conn options",
			slog.String("server", s.String()), slog.Any("error", err))

		if err := conn.Close(); err != nil {
			slog.Error("ws: server close close set options conn",
				slog.String("server", s.String()), slog.Any("error", err))
		}

		return
	}

	conn.SetReadLimit(int64(s.opts.maxReadMsgSize))

	connId := atomic.AddUint64(&s.connId, 1)
	name := fmt.Sprintf("%s_%d", s.Name(), connId)
	c := newConn(name, conn, s.newHandler(), s.opts.connOptions)
	s.conns[c] = struct{}{}
	s.connsMu.Unlock()

	c.serve()

	s.connsMu.Lock()
	defer s.connsMu.Unlock()
	delete(s.conns, c)
}
