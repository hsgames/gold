package http

import (
	"context"
	"fmt"
	"github.com/hsgames/gold/log"
	"github.com/pkg/errors"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
)

type ResponseWriter = http.ResponseWriter
type Request = http.Request
type Handler = func(ResponseWriter, *Request)
type Handlers = map[string]Handler

type Server struct {
	opts     serverOptions
	name     string
	network  string
	addr     string
	serveWg  sync.WaitGroup
	mu       sync.Mutex
	lis      net.Listener
	lisAddr  atomic.Value
	server   *http.Server
	served   bool
	shutdown bool
	logger   log.Logger
}

func NewServer(name, network, addr string, handlers Handlers, logger log.Logger, opt ...ServerOption) *Server {
	if handlers == nil {
		panic("http: NewServer handlers is nil")
	}
	opts := defaultServerOptions(logger)
	for _, o := range opt {
		o(&opts)
	}
	opts.ensure()
	mux := http.NewServeMux()
	for k, v := range handlers {
		v := v
		mux.HandleFunc(k, func(w ResponseWriter, r *Request) {
			if opts.allowAllOrigins {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			}
			handler := v
			if len(opts.chainUnaryInts) == 1 {
				handler = opts.chainUnaryInts[0](handler)
			} else if len(opts.chainUnaryInts) > 1 {
				handler = opts.chainUnaryInts[0](getChainUnaryHandler(opts.chainUnaryInts, 0, handler))
			}
			handler(w, r)
		})
	}
	return &Server{
		opts:    opts,
		name:    name,
		network: network,
		addr:    addr,
		server: &http.Server{
			Handler:      mux,
			ReadTimeout:  opts.readTimeout,
			WriteTimeout: opts.writeTimeout,
		},
		logger: logger,
	}
}

func (s *Server) String() string {
	return fmt.Sprintf("[name:%s][listen_addr:%s]", s.Name(), s.Addr())
}

func (s *Server) Name() string {
	return s.name
}

func (s *Server) Addr() string {
	if lisAddr := s.lisAddr.Load(); lisAddr != nil {
		return lisAddr.(string)
	}
	return s.addr
}

func (s *Server) ListenAndServe() error {
	if err := s.Listen(); err != nil {
		return err
	}
	return s.Serve()
}

func (s *Server) Listen() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.shutdown {
		return errors.Errorf("http: server %s already shutdown", s)
	}
	if s.lis != nil {
		return errors.Errorf("http: server %s already listened", s)
	}
	if s.addr == "" {
		s.addr = ":http"
	}
	lis, err := net.Listen(s.network, s.addr)
	if err != nil {
		return errors.Wrapf(err, "http: server %s listen", s)
	}
	s.lis = lis
	s.lisAddr.Store(lis.Addr().String())
	return nil
}

func (s *Server) Serve() error {
	s.mu.Lock()
	if s.shutdown {
		s.mu.Unlock()
		return errors.Errorf("http: server %s already shutdown", s)
	}
	if s.served {
		s.mu.Unlock()
		return errors.Errorf("http: server %s already served", s)
	}
	if s.lis == nil {
		s.mu.Unlock()
		return errors.Errorf("http: server %s no listener", s)
	}
	s.served = true
	s.serveWg.Add(1)
	defer s.serveWg.Done()
	s.mu.Unlock()
	err := s.server.Serve(s.lis)
	if err == http.ErrServerClosed {
		return nil
	}
	return errors.Wrapf(err, "http: server %s serve", s)
}

func (s *Server) Shutdown() {
	s.mu.Lock()
	if s.shutdown {
		s.mu.Unlock()
		return
	}
	s.shutdown = true
	s.mu.Unlock()
	err := s.server.Shutdown(context.TODO())
	if err != nil {
		s.logger.Error("http: server %s shutdown err:%+v",
			s, errors.WithStack(err))
	}
	s.serveWg.Wait()
}

func Error(w ResponseWriter, error string, code int) {
	http.Error(w, error, code)
}
