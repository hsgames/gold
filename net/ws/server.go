package ws

import (
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/hsgames/gold/log"
	goldnet "github.com/hsgames/gold/net"
	"github.com/hsgames/gold/net/http"
	"github.com/hsgames/gold/net/tcp"
	"github.com/hsgames/gold/safe"
	"github.com/pkg/errors"
	"sync"
	"sync/atomic"
)

type Server struct {
	*http.Server
	opts       serverOptions
	newHandler goldnet.NewHandlerFunc
	upgrader   *websocket.Upgrader
	connsWg    sync.WaitGroup
	connsMu    sync.Mutex
	conns      map[*Conn]struct{}
	connId     uint64
	logger     log.Logger
}

func NewServer(name, network, addr string, newHandler goldnet.NewHandlerFunc,
	logger log.Logger, opt ...ServerOption) *Server {
	opts := defaultServerOptions()
	for _, o := range opt {
		o(&opts)
	}
	opts.ensure()
	s := &Server{
		opts:       opts,
		newHandler: newHandler,
		upgrader: &websocket.Upgrader{
			HandshakeTimeout: opts.handshakeTimeout,
			CheckOrigin:      func(_ *http.Request) bool { return opts.checkOrigin },
		},
		conns:  make(map[*Conn]struct{}),
		logger: logger,
	}
	s.Server = http.NewServer(name, network, addr,
		http.Handlers{opts.pattern: s.serve}, s.logger,
		http.ServerReadTimeout(opts.readTimeout),
		http.ServerWriteTimeout(opts.writeTimeout))
	return s
}

func (s *Server) ConnNum() int {
	s.connsMu.Lock()
	defer s.connsMu.Unlock()
	return len(s.conns)
}

func (s *Server) Shutdown() {
	s.Server.Shutdown()
	s.connsMu.Lock()
	for c := range s.conns {
		c.Shutdown()
	}
	s.conns = make(map[*Conn]struct{})
	s.connsMu.Unlock()
	s.connsWg.Wait()
}

func (s *Server) serve(w http.ResponseWriter, r *http.Request) {
	defer safe.Recover(s.logger)
	if r.Method != "GET" {
		s.logger.Error("ws: server %s method %s not allowed", s, r.Method)
		http.Error(w, "Method not allowed", 405)
		return
	}
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Error("ws: server %s upgrade failed err: %+v", s, errors.WithStack(err))
		return
	}
	s.connsMu.Lock()
	if s.opts.maxConnNum > 0 && len(s.conns) >= s.opts.maxConnNum {
		s.connsMu.Unlock()
		s.logger.Warn("ws: server %s accept too many conns", s)
		err = conn.Close()
		if err != nil {
			s.logger.Error("ws: server %s close overflow conn err: %+v",
				s, errors.Wrapf(err, "ws: conn %+v", conn))
		}
		return
	}
	if err = tcp.SetConnOptions(conn.UnderlyingConn(), s.opts.keepAlivePeriod); err != nil {
		s.connsMu.Unlock()
		s.logger.Error("ws: server %s set conn options err: %+v", s, err)
		if err := conn.Close(); err != nil {
			s.logger.Error("ws: server %s close set options conn err: %+v",
				s, errors.Wrapf(err, "ws: conn %+v", conn))
		}
		return
	}
	conn.SetReadLimit(int64(s.opts.maxReadMsgSize))
	connId := atomic.AddUint64(&s.connId, 1)
	name := fmt.Sprintf("%s_%d", s.Name(), connId)
	c := newConn(name, conn, s, s.newHandler, s.opts.connOptions, s.logger)
	s.conns[c] = struct{}{}
	s.connsMu.Unlock()
	s.connsWg.Add(1)
	defer s.connsWg.Done()
	c.serve()
	s.connsMu.Lock()
	delete(s.conns, c)
	s.connsMu.Unlock()
}
