package ws

import (
	"context"
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	gnet "github.com/hsgames/gold/net"
	"github.com/hsgames/gold/net/tcp"
	"github.com/hsgames/gold/safe"
)

type Client struct {
	opts       options
	name       string
	addr       string
	newHandler func() gnet.Handler
	dialer     *websocket.Dialer
}

func NewClient(name, addr string, newHandler func() gnet.Handler,
	opt ...Option) (c *Client, err error) {

	opts := defaultOptions()
	for _, o := range opt {
		o(&opts)
	}

	if err = opts.check(); err != nil {
		return
	}

	c = &Client{
		opts:       opts,
		name:       name,
		addr:       addr,
		newHandler: newHandler,
		dialer:     &websocket.Dialer{},
	}

	return
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

func (c *Client) Dial(ctx context.Context) (gnet.Conn, error) {
	conn, _, err := c.dialer.DialContext(ctx, c.addr, nil)
	if err != nil {
		return nil, fmt.Errorf("ws: client [%s] dial err [%w]", c, err)
	}

	if err = tcp.SetConnOptions(conn.NetConn(), c.opts.keepAlivePeriod); err != nil {
		if e := conn.Close(); e != nil {
			err = errors.Join(err, e)
		}

		return nil, fmt.Errorf("ws: client [%s] set conn options err [%w]", c, err)
	}

	wc := newConn(c.name, conn, c.newHandler(), c.opts.connOptions)

	safe.Go(wc.serve)

	return wc, nil
}
