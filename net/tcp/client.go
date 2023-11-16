package tcp

import (
	"context"
	"errors"
	"fmt"
	"net"

	gnet "github.com/hsgames/gold/net"
	"github.com/hsgames/gold/safe"
)

type Client struct {
	opts       options
	name       string
	network    string
	addr       string
	newHandler func() gnet.Handler
	dialer     *net.Dialer
}

func NewClient(name, network, addr string,
	newHandler func() gnet.Handler, opt ...Option) (c *Client, err error) {

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
		network:    network,
		addr:       addr,
		newHandler: newHandler,
		dialer:     &net.Dialer{},
	}

	return
}

func (c *Client) String() string {
	return fmt.Sprintf("[name:%s][dail_addr:%s]", c.Name(), c.Addr())
}

func (c *Client) Name() string {
	return c.name
}

func (c *Client) Addr() string {
	return c.addr
}

func (c *Client) Dial(ctx context.Context) (gnet.Conn, error) {
	conn, err := c.dialer.DialContext(ctx, c.network, c.addr)
	if err != nil {
		return nil, fmt.Errorf("tcp: client [%s] dial err [%w]", c, err)
	}

	if err = SetConnOptions(conn, c.opts.keepAlivePeriod); err != nil {
		if e := conn.Close(); e != nil {
			err = errors.Join(err, e)
		}

		return nil, fmt.Errorf("tcp: client [%s] set conn options err [%w]", c, err)
	}

	tc := newConn(c.name, conn, c.newHandler(), c.opts.connOptions)

	safe.Go(tc.serve)

	return tc, nil
}
