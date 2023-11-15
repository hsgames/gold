package tcp

import (
	"fmt"
	gnet "github.com/hsgames/gold/net"
	"github.com/hsgames/gold/safe"
	"log/slog"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

type Conn struct {
	opts         connOptions
	name         string
	conn         net.Conn
	wg           sync.WaitGroup
	handler      gnet.Handler
	reader       Reader
	writer       Writer
	writeChan    chan []byte
	closed       int32
	shutdownOnce sync.Once
	closeOnce    sync.Once
	shutdownChan chan struct{}
	closeChan    chan bool
	wakeupChan   chan struct{}
	doneChan     chan struct{}
	userData     any
}

func newConn(name string, conn net.Conn, handler gnet.Handler, opts connOptions) *Conn {
	return &Conn{
		opts:         opts,
		name:         name,
		conn:         conn,
		handler:      handler,
		reader:       opts.newReader(),
		writer:       opts.newWriter(),
		writeChan:    make(chan []byte, opts.writeChanSize),
		shutdownChan: make(chan struct{}, 1),
		closeChan:    make(chan bool, 1),
		wakeupChan:   make(chan struct{}),
		doneChan:     make(chan struct{}),
	}
}

func (c *Conn) String() string {
	return fmt.Sprintf("[name:%s][local_addr:%s][remote_addr:%s]",
		c.Name(), c.LocalAddr(), c.RemoteAddr())
}

func (c *Conn) Name() string {
	return c.name
}

func (c *Conn) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

func (c *Conn) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

func (c *Conn) UserData() any {
	return c.userData
}

func (c *Conn) SetUserData(data any) {
	c.userData = data
}

func (c *Conn) IsClosed() bool {
	return atomic.LoadInt32(&c.closed) == 1
}

func (c *Conn) Done() chan struct{} {
	return c.doneChan
}

func (c *Conn) Shutdown() {
	c.shutdownOnce.Do(func() {
		atomic.StoreInt32(&c.closed, 1)
		c.shutdownChan <- struct{}{}
	})
}

func (c *Conn) doShutdown() {
	select {
	case c.writeChan <- nil:
		deadLine := time.Now().Add(5 * time.Second)
		if err := c.conn.SetWriteDeadline(deadLine); err != nil {
			slog.Error("tcp: conn shutdown set write deadline",
				slog.String("conn", c.String()), slog.Any("error", err))

			c.Close()
		}
	default:
		slog.Info("tcp: conn shutdown write channel is full",
			slog.String("conn", c.String()))

		c.Close()
	}
}

func (c *Conn) Close() {
	c.close(true)
}

func (c *Conn) close(force bool) {
	c.closeOnce.Do(func() {
		atomic.StoreInt32(&c.closed, 1)
		c.closeChan <- force
	})
}

func (c *Conn) doClose(force bool) {
	defer close(c.wakeupChan)

	if force {
		if err := c.conn.(*net.TCPConn).SetLinger(0); err != nil {
			slog.Error("tcp: conn close set linger",
				slog.String("conn", c.String()), slog.Any("error", err))
		}
	}

	if err := c.conn.Close(); err != nil {
		slog.Error("tcp: conn close",
			slog.String("conn", c.String()), slog.Any("error", err))
	}
}

func (c *Conn) Write(data []byte) {
	if c.IsClosed() || len(data) == 0 {
		return
	}

	select {
	case c.writeChan <- data:
	default:
		slog.Info("tcp: conn write channel is full",
			slog.String("conn", c.String()))

		c.Close()
	}
}

func (c *Conn) serve() {
	defer close(c.doneChan)

	c.wg.Add(2)
	defer c.wg.Wait()

	safe.Go(c.read)
	safe.Go(c.write)

	for {
		select {
		case <-c.shutdownChan:
			c.doShutdown()
		case force := <-c.closeChan:
			c.doClose(force)
			return
		}
	}
}

func (c *Conn) read() {
	defer c.wg.Done()
	defer c.close(false)

	defer func() {
		if err := c.handler.OnClose(c); err != nil {
			slog.Error("tcp: conn on close",
				slog.String("conn", c.String()), slog.Any("error", err))
		}
	}()

	if err := c.handler.OnOpen(c); err != nil {
		slog.Error("tcp: conn on open",
			slog.String("conn", c.String()), slog.Any("error", err))
		return
	}

	for {
		data, err := c.readMessage()
		if err != nil {
			slog.Info("tcp: conn read message",
				slog.String("conn", c.String()), slog.Any("error", err))
			return
		}

		err = c.handler.OnMessage(c, data)
		if err != nil {
			slog.Error("tcp: conn on message",
				slog.String("conn", c.String()), slog.Any("error", err))
			return
		}
	}
}

func (c *Conn) write() {
	defer c.wg.Done()
	defer c.close(false)

	for {
		select {
		case <-c.wakeupChan:
			return
		case data := <-c.writeChan:
			if data == nil {
				return
			}

			if _, err := c.writeMessage(data); err != nil {
				slog.Info("tcp: conn write message",
					slog.String("conn", c.String()), slog.Any("error", err))
				return
			}
		}
	}
}

func (c *Conn) readMessage() ([]byte, error) {
	return c.reader.Read(c.conn, c.opts.maxReadMsgSize)
}

func (c *Conn) writeMessage(data []byte) (int, error) {
	return c.writer.Write(c.conn, data, c.opts.maxWriteMsgSize)
}

func SetConnOptions(conn net.Conn, keepAlivePeriod time.Duration) error {
	if keepAlivePeriod > 0 {
		if err := conn.(*net.TCPConn).SetKeepAlive(true); err != nil {
			return fmt.Errorf("tcp: set conn [%s] keep alive err [%w]", conn, err)
		}

		if err := conn.(*net.TCPConn).SetKeepAlivePeriod(keepAlivePeriod); err != nil {
			return fmt.Errorf("tcp: set conn [%s] keep alive period err [%w]", conn, err)
		}
	}

	return nil
}
