package tcp

import (
	"bufio"
	"fmt"
	"github.com/hsgames/gold/container/queue"
	"github.com/hsgames/gold/log"
	goldnet "github.com/hsgames/gold/net"
	"github.com/hsgames/gold/net/internal"
	"github.com/hsgames/gold/safe"
	"github.com/pkg/errors"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

type Conn struct {
	opts           connOptions
	name           string
	conn           net.Conn
	ep             goldnet.EndPoint
	wg             sync.WaitGroup
	parser         Parser
	handler        goldnet.Handler
	defender       *internal.Defender
	writeQueue     *queue.MPSCQueue
	brPool         *sync.Pool
	bwPool         *sync.Pool
	br             *bufio.Reader
	bw             *bufio.Writer
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

func newConn(name string, conn net.Conn, ep goldnet.EndPoint,
	newParser NewParserFunc, newHandler goldnet.NewHandlerFunc, opts connOptions,
	brPool *sync.Pool, bwPool *sync.Pool, logger log.Logger) *Conn {
	br := brPool.Get().(*bufio.Reader)
	bw := bwPool.Get().(*bufio.Writer)
	br.Reset(conn)
	bw.Reset(conn)
	return &Conn{
		opts:           opts,
		name:           name,
		conn:           conn,
		ep:             ep,
		parser:         newParser(logger),
		handler:        newHandler(logger),
		defender:       internal.NewDefender(opts.maxDefenderMsgNum),
		writeQueue:     queue.NewMPSCQueue(opts.maxWriteQueueSize, opts.writeQueueShrinkSize),
		brPool:         brPool,
		bwPool:         bwPool,
		br:             br,
		bw:             bw,
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
		c.logger.Error("tcp: conn %s shutdown set write deadline err: %+v",
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
		err = c.conn.(*net.TCPConn).SetLinger(0)
		if err != nil {
			c.logger.Error("tcp: conn %s close set linger err: %+v",
				c, errors.WithStack(err))
		}
	}
	err = c.conn.Close()
	if err != nil {
		c.logger.Error("tcp: conn %s close err: %+v",
			c, errors.WithStack(err))
	}
}

func (c *Conn) closeWrite() {
	c.closeWriteChan <- struct{}{}
}

func (c *Conn) doCloseWrite() {
	err := c.conn.(*net.TCPConn).CloseWrite()
	if err != nil {
		c.Close()
		c.logger.Error("tcp: conn %s close write err: %+v",
			c, errors.WithStack(err))
		return
	}
	readDeadline := time.Now().Add(c.opts.shutdownReadPeriod)
	c.readDeadline.Store(readDeadline)
	err = c.conn.SetReadDeadline(readDeadline)
	if err != nil {
		c.Close()
		c.logger.Error("tcp: conn %s close write set read deadline err: %+v",
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

func (c *Conn) clear() {
	c.br.Reset(nil)
	c.bw.Reset(nil)
	c.brPool.Put(c.br)
	c.bwPool.Put(c.bw)
	c.br = nil
	c.bw = nil
}

func (c *Conn) serve() {
	defer c.clear()
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
			c.logger.Error("tcp: conn %s on close err: %+v", c, err)
		}
	}()
	defer c.close(false)
	if err := c.handler.OnOpen(c); err != nil {
		c.logger.Error("tcp: conn %s on open err: %+v", c, err)
		return
	}
	for {
		data, err := c.readMessage()
		if err != nil {
			if !c.isDone() {
				if errors.Is(err, io.EOF) {
					c.logger.Info("tcp: conn %s read message err: %v", c, err)
				} else {
					c.logger.Error("tcp: conn %s read message err: %+v", c, err)
				}
			}
			return
		}
		atomic.AddUint64(&c.readBytes, uint64(len(data)))
		if !c.defender.CheckPacketSpeed() {
			c.logger.Error("tcp: conn %s defender check packet speed", c)
			return
		}
		err = c.handler.OnMessage(c, data)
		if err != nil {
			c.logger.Error("tcp: conn %s handler on message err: %+v", c, err)
			return
		}
	}
}

func (c *Conn) write() {
	defer c.wg.Done()
	defer c.closeWrite()
	var (
		n    int
		err  error
		exit bool
	)
	for {
		datas := c.writeQueue.Pop()
		for _, data := range *datas {
			if data == nil {
				exit = true
				break
			}
			n, err = c.writeMessage(data.([]byte))
			if err != nil {
				if !c.isDone() {
					c.logger.Error("tcp: conn %s buffer write err: %+v", c, err)
				}
				return
			}
			atomic.AddUint64(&c.writeBytes, uint64(n))
		}
		err = c.bw.Flush()
		if err != nil {
			if !c.isDone() {
				c.logger.Error("tcp: conn %s buffer flush err: %+v", c, errors.WithStack(err))
			}
			return
		}
		if exit {
			return
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
	return c.parser.ReadMessage(c.br, c.opts.maxReadMsgSize)
}

func (c *Conn) writeMessage(data []byte) (int, error) {
	return c.parser.WriteMessage(c.bw, data, c.opts.maxWriteMsgSize)
}

func SetConnOptions(conn net.Conn, keepAlivePeriod time.Duration) error {
	if keepAlivePeriod > 0 {
		if err := conn.(*net.TCPConn).SetKeepAlive(true); err != nil {
			return errors.Wrapf(err, "tcp: set conn keep alive, conn %+v", conn)
		}
		if err := conn.(*net.TCPConn).SetKeepAlivePeriod(keepAlivePeriod); err != nil {
			return errors.Wrapf(err, "tcp: set conn keep alive period, conn %+v", conn)
		}
	}
	return nil
}
