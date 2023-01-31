package grpc

import (
	"fmt"
	"github.com/hsgames/gold/log"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"net"
	"sync"
	"sync/atomic"
)

type Server struct {
	opts     serverOptions
	name     string
	network  string
	addr     string
	serveWg  sync.WaitGroup
	mu       sync.Mutex
	lis      net.Listener
	lisAddr  atomic.Value
	server   *grpc.Server
	served   bool
	shutdown bool
	logger   log.Logger
}

func NewServer(name, network, addr string, logger log.Logger, opt ...ServerOption) *Server {
	opts := defaultServerOptions(logger)
	for _, o := range opt {
		o(&opts)
	}
	opts.ensure()
	var serverOpts []grpc.ServerOption
	if len(opts.chainUnaryInts) > 0 {
		serverOpts = append(serverOpts, grpc.ChainUnaryInterceptor(opts.chainUnaryInts...))
	}
	if opts.creds != nil {
		serverOpts = append(serverOpts, grpc.Creds(opts.creds))
	}
	if opts.maxRecvMsgSize > 0 {
		serverOpts = append(serverOpts, grpc.MaxRecvMsgSize(opts.maxRecvMsgSize))
	}
	if opts.maxSendMsgSize > 0 {
		serverOpts = append(serverOpts, grpc.MaxSendMsgSize(opts.maxSendMsgSize))
	}
	return &Server{
		opts:    opts,
		name:    name,
		network: network,
		addr:    addr,
		server:  grpc.NewServer(serverOpts...),
		logger:  logger,
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
		return errors.Errorf("grpc: server %s already shutdown", s)
	}
	if s.lis != nil {
		return errors.Errorf("grpc: server %s already listened", s)
	}
	if s.addr == "" {
		s.addr = ":http"
	}
	lis, err := net.Listen(s.network, s.addr)
	if err != nil {
		return errors.Wrapf(err, "grpc: server %s listen", s)
	}
	s.lis = lis
	s.lisAddr.Store(lis.Addr().String())
	return nil
}

func (s *Server) Serve() error {
	s.mu.Lock()
	if s.shutdown {
		s.mu.Unlock()
		return errors.Errorf("grpc: server %s already shutdown", s)
	}
	if s.served {
		s.mu.Unlock()
		return errors.Errorf("grpc: server %s already served", s)
	}
	if s.lis == nil {
		s.mu.Unlock()
		return errors.Errorf("grpc: server %s no listener", s)
	}
	s.served = true
	s.serveWg.Add(1)
	defer s.serveWg.Done()
	s.mu.Unlock()
	err := s.server.Serve(s.lis)
	if err != nil {
		return errors.Wrapf(err, "grpc: server %s serve", s)
	}
	return nil
}

func (s *Server) Shutdown() {
	s.mu.Lock()
	if s.shutdown {
		s.mu.Unlock()
		return
	}
	s.shutdown = true
	s.mu.Unlock()
	s.server.GracefulStop()
	s.serveWg.Wait()
}

func (s *Server) RegisterService(desc *grpc.ServiceDesc, impl any) {
	s.server.RegisterService(desc, impl)
}
