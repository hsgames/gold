package ws

import (
	"context"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/hsgames/gold/log"
	goldnet "github.com/hsgames/gold/net"
	"github.com/hsgames/gold/net/tcp"
	"github.com/hsgames/gold/safe"
	"github.com/pkg/errors"
	"sync"
	"time"
)

type Client struct {
	opts         clientOptions
	name         string
	addr         string
	newHandler   goldnet.NewHandlerFunc
	dialWg       sync.WaitGroup
	connWg       sync.WaitGroup
	mu           sync.Mutex
	connDoneChan chan struct{}
	conn         *Conn
	connId       uint64
	dialer       *websocket.Dialer
	dialed       bool
	shutdown     bool
	cancel       context.CancelFunc
	logger       log.Logger
}

func NewClient(name, addr string, newHandler goldnet.NewHandlerFunc,
	logger log.Logger, opt ...ClientOption) *Client {
	opts := defaultClientOptions()
	for _, o := range opt {
		o(&opts)
	}
	opts.ensure()
	return &Client{
		name:         name,
		addr:         addr,
		newHandler:   newHandler,
		opts:         opts,
		connDoneChan: make(chan struct{}, 1),
		dialer:       &websocket.Dialer{HandshakeTimeout: opts.handshakeTimeout},
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
		return errors.Errorf("ws: client %s already shutdown", c)
	}
	if c.dialed {
		c.mu.Unlock()
		return errors.Errorf("ws: client %s already dialed", c)
	}
	c.dialed = true
	c.dialWg.Add(1)
	defer c.dialWg.Done()
	var ctx context.Context
	ctx, c.cancel = context.WithCancel(context.Background())
	c.mu.Unlock()
	var (
		subCtx context.Context
		cancel context.CancelFunc
	)
	for {
		if c.opts.dialTimeout > 0 {
			subCtx, cancel = context.WithTimeout(ctx, c.opts.dialTimeout)
		} else {
			subCtx, cancel = context.WithCancel(ctx)
		}
		conn, _, err := c.dialer.DialContext(subCtx, c.addr, nil)
		cancel()
		if err != nil {
			select {
			case <-ctx.Done():
				if err = ctx.Err(); err != nil && err != context.Canceled {
					return errors.Wrapf(err, "ws: client %s dial", c)
				}
				return nil
			default:
			}
			if c.opts.autoRedial {
				c.logger.Error("ws: client %s dial retry err: %+v", c, errors.WithStack(err))
				goto sleep
			}
			return errors.Wrapf(err, "ws: client %s dial", c)
		}
		err = tcp.SetConnOptions(conn.UnderlyingConn(), c.opts.keepAlivePeriod)
		if err != nil {
			if c.opts.autoRedial {
				c.logger.Error("ws: client %s set conn options err: %+v", c, err)
			}
			if err = conn.Close(); err != nil {
				c.logger.Error("ws: client %s close set options conn err: %+v", c, err)
			}
			if c.opts.autoRedial {
				goto sleep
			}
			return errors.Wrap(err, "ws: client set conn options")
		}
		c.handleConn(conn)
		select {
		case <-ctx.Done():
			if err = ctx.Err(); err != nil && err != context.Canceled {
				return errors.Wrapf(err, "ws: client %s dial", c)
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
				return errors.Wrapf(err, "ws: client %s dial", c)
			}
			return nil
		case <-timer.C:
		}
	}
}

func (c *Client) handleConn(conn *websocket.Conn) {
	c.connId++
	name := fmt.Sprintf("%s_%d", c.name, c.connId)
	wsConn := newConn(name, conn, c, c.newHandler, c.opts.connOptions, c.logger)
	c.conn = wsConn
	c.connWg.Add(1)
	safe.Go(c.logger, func() {
		defer c.connWg.Done()
		defer func() { c.connDoneChan <- struct{}{} }()
		wsConn.serve()
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
