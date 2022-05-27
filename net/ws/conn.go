package ws

import (
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/hsgames/gold/container/queue"
	"github.com/hsgames/gold/log"
	goldnet "github.com/hsgames/gold/net"
	"github.com/hsgames/gold/net/internal"
	"github.com/hsgames/gold/safe"
	"github.com/pkg/errors"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

type Conn struct {
	opts           connOptions
	name           string
	conn           *websocket.Conn
	ep             goldnet.EndPoint
	wg             sync.WaitGroup
	handler        goldnet.Handler
	defender       *internal.Defender
	writeQueue     *queue.MPSCQueue
	readBytes      uint64
	writeBytes     uint64
	closed         int32
	readDeadline   atomic.Value
	shutdownOnce   sync.Once
	closeOnce      sync.Once
	shutdownChan   chan struct{}
	closeWriteChan chan struct{}
	closeChan      chan bool
	doneChan       chan struct{}
	userData       interface{}
	logger         log.Logger
}

func newConn(name string, conn *websocket.Conn, ep goldnet.EndPoint,
	newHandler goldnet.NewHandlerFunc, opts connOptions, logger log.Logger) *Conn {
	return &Conn{
		opts:           opts,
		name:           name,
		conn:           conn,
		ep:             ep,
		handler:        newHandler(logger),
		defender:       internal.NewDefender(opts.maxDefenderMsgNum),
		writeQueue:     queue.NewMPSCQueue(opts.maxWriteQueueSize, opts.writeQueueShrinkSize),
		shutdownChan:   make(chan struct{}, 1),
		closeWriteChan: make(chan struct{}, 1),
		closeChan:      make(chan bool, 1),
		doneChan:       make(chan struct{}),
		logger:         logger,
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

func (c *Conn) EndPoint() goldnet.EndPoint {
	return c.ep
}

func (c *Conn) UserData() interface{} {
	return c.userData
}

func (c *Conn) SetUserData(data interface{}) {
	c.userData = data
}

func (c *Conn) ReadBytes() uint64 {
	return atomic.LoadUint64(&c.readBytes)
}

func (c *Conn) WriteBytes() uint64 {
	return atomic.LoadUint64(&c.writeBytes)
}

func (c *Conn) IsClosed() bool {
	return atomic.LoadInt32(&c.closed) == 1
}

func (c *Conn) Shutdown() {
	c.shutdownOnce.Do(func() {
		atomic.StoreInt32(&c.closed, 1)
		c.shutdownChan <- struct{}{}
	})
}

func (c *Conn) doShutdown() {
	c.writeQueue.Push(nil)
	err := c.conn.SetWriteDeadline(time.Now().Add(c.opts.shutdownWritePeriod))
	if err != nil {
		c.Close()
		c.logger.Error("ws: conn %s shutdown set write deadline err: %+v",
			c, errors.WithStack(err))
	}
}

func (c *Conn) Close() {
	c.close(true)
}

func (c *Conn) close(noLinger bool) {
	c.closeOnce.Do(func() {
		atomic.StoreInt32(&c.closed, 1)
		c.closeChan <- noLinger
	})
}

func (c *Conn) doClose(noLinger bool) {
	var err error
	close(c.doneChan)
	c.writeQueue.Push(nil)
	if noLinger {
		err = c.conn.UnderlyingConn().(*net.TCPConn).SetLinger(0)
		if err != nil {
			c.logger.Error("ws: conn %s close set linger err: %+v",
				c, errors.WithStack(err))
		}
	}
	err = c.conn.Close()
	if err != nil {
		c.logger.Error("ws: conn %s close err: %+v",
			c, errors.WithStack(err))
	}
}

func (c *Conn) closeWrite() {
	c.closeWriteChan <- struct{}{}
}

func (c *Conn) doCloseWrite() {
	err := c.conn.UnderlyingConn().(*net.TCPConn).CloseWrite()
	if err != nil {
		c.Close()
		c.logger.Error("ws: conn %s close write err: %+v",
			c, errors.WithStack(err))
		return
	}
	readDeadline := time.Now().Add(c.opts.shutdownReadPeriod)
	c.readDeadline.Store(readDeadline)
	err = c.conn.SetReadDeadline(readDeadline)
	if err != nil {
		c.Close()
		c.logger.Error("ws: conn %s close write set read deadline err: %+v",
			c, errors.WithStack(err))
	}
}

func (c *Conn) isDone() bool {
	select {
	case <-c.doneChan:
		return true
	default:
	}
	return false
}

func (c *Conn) Write(data []byte) {
	if c.IsClosed() || len(data) == 0 {
		return
	}
	c.writeQueue.Push(data)
}

func (c *Conn) serve() {
	c.wg.Add(2)
	defer c.wg.Wait()
	safe.Go(c.logger, c.read)
	safe.Go(c.logger, c.write)
	for {
		select {
		case <-c.shutdownChan:
			c.doShutdown()
		case <-c.closeWriteChan:
			c.doCloseWrite()
		case noLinger := <-c.closeChan:
			c.doClose(noLinger)
			return
		}
	}
}

func (c *Conn) read() {
	defer c.wg.Done()
	defer func() {
		if err := c.handler.OnClose(c); err != nil {
			c.logger.Error("ws: conn %s on close err: %+v", c, err)
		}
	}()
	defer c.close(false)
	if err := c.handler.OnOpen(c); err != nil {
		c.logger.Error("ws: conn %s on open err: %+v", c, err)
		return
	}
	for {
		data, err := c.readMessage()
		if err != nil {
			if !c.isDone() {
				c.logger.Info("ws: conn %s read message err: %v", c, err)
			}
			return
		}
		atomic.AddUint64(&c.readBytes, uint64(len(data)))
		if !c.defender.CheckPacketSpeed() {
			c.logger.Error("ws: conn %s defender check packet speed", c)
			return
		}
		err = c.handler.OnMessage(c, data)
		if err != nil {
			c.logger.Error("ws: conn %s handler on message err: %+v", c, err)
			return
		}
	}
}

func (c *Conn) write() {
	defer c.wg.Done()
	defer c.closeWrite()
	for {
		datas := c.writeQueue.Pop()
		for _, data := range *datas {
			if data == nil {
				return
			}
			n, err := c.writeMessage(data.([]byte))
			if err != nil {
				if !c.isDone() {
					c.logger.Error("ws: conn %s write err: %+v", c, err)
				}
				return
			}
			atomic.AddUint64(&c.writeBytes, uint64(n))
		}
	}
}

func (c *Conn) readMessage() ([]byte, error) {
	if c.opts.readDeadlinePeriod > 0 {
		readDeadline, ok := c.readDeadline.Load().(time.Time)
		if !ok {
			readDeadline = time.Now().Add(c.opts.readDeadlinePeriod)
		}
		err := c.conn.SetReadDeadline(readDeadline)
		if err != nil {
			return nil, errors.WithStack(err)
		}
	}
	msgType, data, err := c.conn.ReadMessage()
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if msgType != c.opts.msgType {
		return nil, errors.Errorf("ws: conn read msg type %d != %d",
			msgType, c.opts.msgType)
	}
	msgSize := len(data)
	if msgSize <= 0 {
		return nil, errors.Errorf("ws: conn read msg size %d <= 0", msgSize)
	}
	if msgSize > c.opts.maxReadMsgSize {
		return nil, errors.Errorf("ws: conn read msg size %d > %d",
			msgSize, c.opts.maxReadMsgSize)
	}
	return data, nil
}

func (c *Conn) writeMessage(data []byte) (int, error) {
	msgSize := len(data)
	if msgSize <= 0 {
		return 0, errors.Errorf("ws: conn write msg size %d <= 0", msgSize)
	}
	if msgSize > c.opts.maxWriteMsgSize {
		return 0, errors.Errorf("ws: conn write msg size %d > %d",
			msgSize, c.opts.maxWriteMsgSize)
	}
	err := c.conn.WriteMessage(c.opts.msgType, data)
	if err != nil {
		return 0, errors.WithStack(err)
	}
	return len(data), nil
}
