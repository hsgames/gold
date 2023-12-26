package ws

import (
	"errors"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	gnet "github.com/hsgames/gold/net"
	"github.com/hsgames/gold/pool/bytespool"
	"github.com/hsgames/gold/safe"
)

var (
	ErrConnClosed        = errors.New("ws: conn is closed")
	ErrConnWriteChanFull = errors.New("ws: conn write channel is full")
	ErrConnWriteDataNil  = errors.New("ws: conn write data is nil")
)

type Conn struct {
	opts         connOptions
	name         string
	conn         *websocket.Conn
	wg           sync.WaitGroup
	handler      gnet.Handler
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

func newConn(name string, conn *websocket.Conn, handler gnet.Handler, opts connOptions) *Conn {
	return &Conn{
		opts:         opts,
		name:         name,
		conn:         conn,
		handler:      handler,
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
			slog.Error("ws: conn shutdown set write deadline",
				slog.String("conn", c.String()), slog.Any("error", err))

			c.Close()
		}
	default:
		slog.Info("ws: conn shutdown write channel is full",
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
		if err := c.conn.NetConn().(*net.TCPConn).SetLinger(0); err != nil {
			slog.Error("ws: conn close set linger",
				slog.String("conn", c.String()), slog.Any("error", err))
		}
	}

	if err := c.conn.Close(); err != nil {
		slog.Error("ws: conn close",
			slog.String("conn", c.String()), slog.Any("error", err))
	}
}

func (c *Conn) Write(data []byte) error {
	if data == nil {
		return ErrConnWriteDataNil
	}

	if c.IsClosed() {
		return ErrConnClosed
	}

	b := bytespool.Get(len(data))
	copy(b, data)

	select {
	case c.writeChan <- b:
		return nil
	default:
		bytespool.Put(b)
		return ErrConnWriteChanFull
	}
}

func (c *Conn) serve() {
	defer close(c.doneChan)

	defer c.clear()

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

func (c *Conn) clear() {
	for {
		select {
		case data := <-c.writeChan:
			if data != nil {
				bytespool.Put(data)
			}
		default:
			return
		}
	}
}

func (c *Conn) read() {
	defer c.wg.Done()
	defer c.close(false)

	defer c.handler.OnClose(c)

	if err := c.handler.OnOpen(c); err != nil {
		return
	}

	for {
		data, err := c.readData()
		if err != nil {
			slog.Debug("ws: conn read data",
				slog.String("conn", c.String()), slog.Any("error", err))
			return
		}

		if err = c.handler.OnRead(c, data); err != nil {
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

			if _, err := c.writeData(data); err != nil {
				slog.Debug("ws: conn write data",
					slog.String("conn", c.String()), slog.Any("error", err))
				bytespool.Put(data)
				return
			}

			bytespool.Put(data)
		}
	}
}

func (c *Conn) readData() (data []byte, err error) {
	var dataType int

	if dataType, data, err = c.conn.ReadMessage(); err != nil {
		err = fmt.Errorf("ws: conn [%s] read message error [%w]", c, err)
		return
	}

	if dataType != c.opts.dataType {
		err = fmt.Errorf("ws: conn [%s] read data type [%d] != [%d]",
			c, dataType, c.opts.dataType)
		return
	}

	dataSize := len(data)

	if dataSize > c.opts.maxReadDataSize {
		err = fmt.Errorf("ws: conn [%s] read data size [%d] > [%d]",
			c, dataSize, c.opts.maxReadDataSize)
		return
	}

	return
}

func (c *Conn) writeData(data []byte) (n int, err error) {
	dataSize := len(data)

	if dataSize > c.opts.maxWriteDataSize {
		err = fmt.Errorf("ws: conn [%s] write data size [%d] > [%d]",
			c, dataSize, c.opts.maxWriteDataSize)
		return
	}

	if err = c.conn.WriteMessage(c.opts.dataType, data); err != nil {
		err = fmt.Errorf("ws: conn [%s] write message error [%w]", c, err)
		return
	}

	n = len(data)

	return
}
