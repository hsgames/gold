package tcp

import (
	"bufio"
	"fmt"
	"github.com/hsgames/gold/log"
	goldnet "github.com/hsgames/gold/net"
	"github.com/hsgames/gold/safe"
	"github.com/pkg/errors"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

type Server struct {
	opts         serverOptions
	name         string
	network      string
	addr         string
	newParser    NewParserFunc
	newHandler   goldnet.NewHandlerFunc
	brPool       *sync.Pool
	bwPool       *sync.Pool
	serveWg      sync.WaitGroup
	connsWg      sync.WaitGroup
	mu           sync.Mutex
	closeLisOnce sync.Once
	lis          net.Listener
	lisAddr      atomic.Value
	connsMu      sync.Mutex
	conns        map[*Conn]struct{}
	connId       uint64
	served       bool
	shutdown     bool
	doneChan     chan struct{}
	logger       log.Logger
}

func NewServer(name, network, addr string, newParser NewParserFunc,
	newHandler goldnet.NewHandlerFunc, logger log.Logger, opt ...ServerOption) *Server {
	opts := defaultServerOptions()
	for _, o := range opt {
		o(&opts)
	}
	opts.ensure()
	return &Server{
		opts:       opts,
		name:       name,
		network:    network,
		addr:       addr,
		newParser:  newParser,
		newHandler: newHandler,
		brPool: &sync.Pool{
			New: func() any {
				return bufio.NewReaderSize(nil, opts.readBufSize)
			},
		},
		bwPool: &sync.Pool{
			New: func() any {
				return bufio.NewWriterSize(nil, opts.writeBufSize)
			},
		},
		conns:    make(map[*Conn]struct{}),
		doneChan: make(chan struct{}),
		logger:   logger,
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

func (s *Server) ConnNum() int {
	s.connsMu.Lock()
	defer s.connsMu.Unlock()
	return len(s.conns)
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
		return errors.Errorf("tcp: server %s already shutdown", s)
	}
	if s.lis != nil {
		return errors.Errorf("tcp: server %s already listened", s)
	}
	lis, err := net.Listen(s.network, s.addr)
	if err != nil {
		return errors.Wrapf(err, "tcp: server %s listen", s.addr)
	}
	s.lis = lis
	s.lisAddr.Store(lis.Addr().String())
	return nil
}

func (s *Server) Serve() error {
	s.mu.Lock()
	if s.shutdown {
		s.mu.Unlock()
		return errors.Errorf("tcp: server %s already shutdown", s)
	}
	if s.served {
		s.mu.Unlock()
		return errors.Errorf("tcp: server %s already served", s)
	}
	if s.lis == nil {
		s.mu.Unlock()
		return errors.Errorf("tcp: server %s no listener", s)
	}
	s.served = true
	s.serveWg.Add(1)
	defer s.serveWg.Done()
	s.mu.Unlock()
	defer func() {
		if err := s.closeListener(); err != nil {
			s.logger.Error("tcp: server %s close listener err: %+v", s, err)
		}
	}()
	var tempDelay time.Duration
	for {
		conn, err := s.lis.Accept()
		if err != nil {
			select {
			case <-s.doneChan:
				return nil
			default:
			}
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}
				if max := 1 * time.Second; tempDelay > max {
					tempDelay = max
				}
				s.logger.Error("tcp: server %s accept retry",
					s, errors.WithStack(err))
				timer := time.NewTimer(tempDelay)
				select {
				case <-timer.C:
				case <-s.doneChan:
					timer.Stop()
					return nil
				}
				continue
			}
			return errors.Wrapf(err, "tcp: server %s accept", s)
		}
		tempDelay = 0
		s.serveWg.Add(1)
		safe.Go(s.logger, func() {
			defer s.serveWg.Done()
			s.handleConn(conn)
		})
	}
}

func (s *Server) handleConn(conn net.Conn) {
	s.connsMu.Lock()
	if s.opts.maxConnNum > 0 && len(s.conns) >= s.opts.maxConnNum {
		s.connsMu.Unlock()
		s.logger.Warn("tcp: server %s accept too many conns", s)
		if err := conn.Close(); err != nil {
			s.logger.Error("tcp: server %s close overflow conn",
				s, errors.Wrapf(err, "tcp: conn %+v", conn))
		}
		return
	}
	if err := SetConnOptions(conn, s.opts.keepAlivePeriod); err != nil {
		s.connsMu.Unlock()
		s.logger.Error("tcp: server %s set conn options", s, err)
		if err := conn.Close(); err != nil {
			s.logger.Error("tcp: server %s close set options conn",
				s, errors.Wrapf(err, "tcp: conn %+v", conn))
		}
		return
	}
	s.connId++
	name := fmt.Sprintf("%s_%d", s.name, s.connId)
	c := newConn(name, conn, s, s.newParser, s.newHandler,
		s.opts.connOptions, s.brPool, s.bwPool, s.logger)
	s.conns[c] = struct{}{}
	s.connsMu.Unlock()
	s.connsWg.Add(1)
	safe.Go(s.logger, func() {
		defer s.connsWg.Done()
		c.serve()
		s.connsMu.Lock()
		delete(s.conns, c)
		s.connsMu.Unlock()
	})
}

func (s *Server) Shutdown() {
	s.mu.Lock()
	if s.shutdown {
		s.mu.Unlock()
		return
	}
	s.shutdown = true
	s.mu.Unlock()
	close(s.doneChan)
	if s.lis != nil {
		err := s.closeListener()
		if err != nil {
			s.logger.Error("tcp: server %s close listener err: %+v", s, err)
		}
	}
	s.serveWg.Wait()
	s.connsMu.Lock()
	for c := range s.conns {
		c.Shutdown()
	}
	s.conns = nil
	s.connsMu.Unlock()
	s.connsWg.Wait()
}

func (s *Server) closeListener() error {
	var err error
	s.closeLisOnce.Do(func() {
		err = errors.WithStack(s.lis.Close())
	})
	return err
}
