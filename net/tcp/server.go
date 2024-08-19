package tcp

import (
	"context"
	"errors"
	"fmt"
	"github.com/hsgames/gold/net/internal"
	"log/slog"
	"net"
	"sync"
	"time"

	gnet "github.com/hsgames/gold/net"
	"github.com/hsgames/gold/safe"
)

type Server struct {
	opts       options
	name       string
	network    string
	addr       string
	newHandler func() gnet.Handler
	serveWg    sync.WaitGroup
	mu         sync.Mutex
	ln         net.Listener
	connsMu    sync.Mutex
	conns      map[*Conn]struct{}
	served     bool
	shutdown   bool
	closeOnce  sync.Once
	doneChan   chan struct{}
}

func NewServer(name, network, addr string,
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
		network:    network,
		addr:       addr,
		newHandler: newHandler,
		conns:      make(map[*Conn]struct{}),
		doneChan:   make(chan struct{}),
	}

	return
}

func (s *Server) String() string {
	return fmt.Sprintf("[name:%s][listen_addr:%s]", s.Name(), s.addr)
}

func (s *Server) Name() string {
	return s.name
}

func (s *Server) Addr() string {
	return s.addr
}

func (s *Server) ConnNum() int {
	s.connsMu.Lock()
	defer s.connsMu.Unlock()
	return len(s.conns)
}

func (s *Server) ListenAndServe() (err error) {
	s.mu.Lock()

	if s.shutdown {
		s.mu.Unlock()
		err = fmt.Errorf("tcp: server [%s] is already shutdown", s)
		return
	}

	if s.served {
		s.mu.Unlock()
		err = fmt.Errorf("tcp: server [%s] is already served", s)
		return
	}

	s.served = true

	s.serveWg.Add(1)
	defer s.serveWg.Done()

	s.mu.Unlock()

	if s.ln, err = net.Listen(s.network, s.addr); err != nil {
		err = fmt.Errorf("tcp: server [%s] listen error [%w]", s.addr, err)
		return
	}

	defer func() {
		if err = s.ln.Close(); err != nil {
			slog.Error("tcp: server close",
				slog.String("server", s.String()), slog.Any("error", err))
		}
	}()

	var (
		tempDelay time.Duration
		conn      net.Conn
	)

	for {
		if conn, err = s.ln.Accept(); err != nil {
			select {
			case <-s.doneChan:
				return nil
			default:
			}

			var ne net.Error
			if errors.As(err, &ne) && ne.Timeout() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}

				if maxDelay := 1 * time.Second; tempDelay > maxDelay {
					tempDelay = maxDelay
				}

				slog.Info("tcp: server accept retry",
					slog.String("server", s.String()), slog.Any("error", err))

				timer := time.NewTimer(tempDelay)
				select {
				case <-timer.C:
				case <-s.doneChan:
					timer.Stop()
					return nil
				}

				continue
			}

			return fmt.Errorf("tcp: server [%s] accept error [%w]", s, err)
		}
		tempDelay = 0

		s.serveWg.Add(1)
		safe.Go(func() {
			defer s.serveWg.Done()
			s.handleConn(conn)
		})
	}
}

func (s *Server) handleConn(conn net.Conn) {
	s.connsMu.Lock()

	if s.opts.maxConnNum > 0 && len(s.conns) >= s.opts.maxConnNum {
		s.connsMu.Unlock()

		slog.Info("tcp: server accept too many conns",
			slog.String("server", s.String()))

		if err := conn.Close(); err != nil {
			slog.Error("tcp: server close overflow conn",
				slog.String("server", s.String()), slog.Any("error", err))
		}

		return
	}

	if err := SetConnOptions(conn, s.opts.keepAlivePeriod); err != nil {
		s.connsMu.Unlock()

		slog.Error("tcp: server set conn options",
			slog.String("server", s.String()), slog.Any("error", err))

		if err := conn.Close(); err != nil {
			slog.Error("tcp: server close close set options conn",
				slog.String("server", s.String()), slog.Any("error", err))
		}

		return
	}

	c := newConn(internal.NextId(), s.name, conn, s.newHandler(), s.opts.connOptions)
	s.conns[c] = struct{}{}

	s.connsMu.Unlock()

	safe.Go(func() {
		c.serve()

		s.connsMu.Lock()
		defer s.connsMu.Unlock()
		delete(s.conns, c)
	})
}

func (s *Server) Shutdown(ctx context.Context) {
	s.mu.Lock()
	if s.shutdown {
		s.mu.Unlock()
		return
	}
	s.shutdown = true
	s.mu.Unlock()

	close(s.doneChan)

	if s.ln != nil {
		if err := s.ln.Close(); err != nil {
			slog.Error("tcp: server close",
				slog.String("server", s.String()), slog.Any("error", err))
		}
	}

	s.serveWg.Wait()

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
