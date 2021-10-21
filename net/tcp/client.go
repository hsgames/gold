package tcp

import (
	"bufio"
	"context"
	"fmt"
	"github.com/hsgames/gold/log"
	goldnet "github.com/hsgames/gold/net"
	"github.com/hsgames/gold/safe"
	"github.com/pkg/errors"
	"net"
	"sync"
	"time"
)

type Client struct {
	opts         clientOptions
	name         string
	network      string
	addr         string
	newParser    NewParserFunc
	newHandler   goldnet.NewHandlerFunc
	brPool       *sync.Pool
	bwPool       *sync.Pool
	dialWg       sync.WaitGroup
	connWg       sync.WaitGroup
	mu           sync.Mutex
	connDoneChan chan struct{}
	conn         *Conn
	connId       uint64
	dialer       *net.Dialer
	dialed       bool
	shutdown     bool
	cancel       context.CancelFunc
	logger       log.Logger
}

func NewClient(name, network, addr string, newParser NewParserFunc,
	newHandler goldnet.NewHandlerFunc, logger log.Logger, opt ...ClientOption) *Client {
	opts := defaultClientOptions()
	for _, o := range opt {
		o(&opts)
	}
	opts.ensure()
	return &Client{
		name:       name,
		network:    network,
		addr:       addr,
		newParser:  newParser,
		newHandler: newHandler,
		brPool: &sync.Pool{
			New: func() interface{} {
				return bufio.NewReaderSize(nil, opts.readBufSize)
			},
		},
		bwPool: &sync.Pool{
			New: func() interface{} {
				return bufio.NewWriterSize(nil, opts.writeBufSize)
			},
		},
		opts:         opts,
		connDoneChan: make(chan struct{}, 1),
		dialer:       &net.Dialer{Timeout: opts.dialTimeout},
		logger:       logger,
	}
}

func (c *Client) String() string {
	return fmt.Sprintf("[name:%s][connect_addr:%s]", c.Name(), c.Addr())
}

func (c *Client) Name() string {
	return c.name
}

func (c *Client) Addr() string {
	return c.addr
}

func (c *Client) Dial() error {
	c.mu.Lock()
	if c.shutdown {
		c.mu.Unlock()
		return errors.Errorf("tcp: client %s already shutdown", c)
	}
	if c.dialed {
		c.mu.Unlock()
		return errors.Errorf("tcp: client %s already dialed", c)
	}
	c.dialed = true
	c.dialWg.Add(1)
	defer c.dialWg.Done()
	var ctx context.Context
	ctx, c.cancel = context.WithCancel(context.Background())
	c.mu.Unlock()
	for {
		conn, err := c.dialer.DialContext(ctx, c.network, c.addr)
		if err != nil {
			select {
			case <-ctx.Done():
				if err = ctx.Err(); err != nil && err != context.Canceled {
					return errors.Wrapf(err, "tcp: client %s dial", c)
				}
				return nil
			default:
			}
			if c.opts.autoRedial {
				c.logger.Error("tcp: client %s dial retry err: %+v", c, errors.WithStack(err))
				goto sleep
			}
			return errors.Wrapf(err, "tcp: client %s dial", c)
		}
		err = SetConnOptions(conn, c.opts.keepAlivePeriod)
		if err != nil {
			if c.opts.autoRedial {
				c.logger.Error("tcp: client %s set conn options err: %+v", c, err)
			}
			if err = conn.Close(); err != nil {
				c.logger.Error("tcp: client %s close set options conn err: %+v",
					c, errors.Wrapf(err, "tcp: conn %+v", conn))
			}
			if c.opts.autoRedial {
				goto sleep
			}
			return errors.Wrapf(err, "tcp: client %s set conn options", c)
		}
		c.handleConn(conn)
		select {
		case <-ctx.Done():
			if err = ctx.Err(); err != nil && err != context.Canceled {
				return errors.Wrapf(err, "tcp: client %s dial addr", c)
			}
			return nil
		case <-c.connDoneChan:
			c.conn = nil
		}

	sleep:
		if !c.opts.autoRedial {
			return nil
		}
		timer := time.NewTimer(c.opts.redialInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			if err = ctx.Err(); err != nil && err != context.Canceled {
				return errors.Wrapf(err, "tcp: client %s dial addr", c)
			}
			return nil
		case <-timer.C:
		}
	}
}

func (c *Client) handleConn(conn net.Conn) {
	c.connId++
	name := fmt.Sprintf("%s_%d", c.name, c.connId)
	tcpConn := newConn(name, conn, c, c.newParser, c.newHandler,
		c.opts.connOptions, c.brPool, c.bwPool, c.logger)
	c.conn = tcpConn
	c.connWg.Add(1)
	safe.Go(c.logger, func() {
		defer c.connWg.Done()
		defer func() { c.connDoneChan <- struct{}{} }()
		tcpConn.serve()
	})
}

func (c *Client) Shutdown() {
	c.mu.Lock()
	if c.shutdown {
		c.mu.Unlock()
		return
	}
	c.shutdown = true
	c.mu.Unlock()
	if c.cancel != nil {
		c.cancel()
	}
	c.dialWg.Wait()
	if c.conn != nil {
		c.conn.Shutdown()
		c.conn = nil
	}
	c.connWg.Wait()
}
