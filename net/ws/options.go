package ws

import (
	"fmt"
	"github.com/gorilla/websocket"
	"math"
	"time"
)

const (
	BinaryMessage = websocket.BinaryMessage
	TextMessage   = websocket.TextMessage
)

type connOptions struct {
	maxWriteQueueSize    int
	writeQueueShrinkSize int
	maxReadMsgSize       int
	maxWriteMsgSize      int
	msgType              int
	keepAlivePeriod      time.Duration
	shutdownReadPeriod   time.Duration
	shutdownWritePeriod  time.Duration
}

func defaultConnOptions() connOptions {
	return connOptions{
		maxWriteQueueSize:    0,
		writeQueueShrinkSize: 1024 * 1024,
		maxReadMsgSize:       math.MaxUint16,
		maxWriteMsgSize:      math.MaxUint16,
		msgType:              BinaryMessage,
		keepAlivePeriod:      3 * time.Minute,
		shutdownReadPeriod:   5 * time.Second,
		shutdownWritePeriod:  30 * time.Second,
	}
}

func (opts *connOptions) ensure() {
	if opts.maxReadMsgSize <= 0 {
		panic(fmt.Sprintf("ws: connOptions maxReadMsgSize:%d <= 0", opts.maxReadMsgSize))
	}
	if opts.maxWriteMsgSize <= 0 {
		panic(fmt.Sprintf("ws: connOptions maxWriteMsgSize:%d <= 0", opts.maxWriteMsgSize))
	}
	if opts.shutdownReadPeriod <= 0 {
		panic(fmt.Sprintf("ws: connOptions shutdownReadPeriod:%d <= 0",
			opts.shutdownReadPeriod))
	}
	if opts.shutdownWritePeriod <= 0 {
		panic(fmt.Sprintf("ws: connOptions shutdownWritePeriod:%d <= 0",
			opts.shutdownWritePeriod))
	}
	switch opts.msgType {
	case BinaryMessage, TextMessage:
	default:
		panic("ws: connOptions msgType not in (BinaryMessage, TextMessage)")
	}
}

type serverOptions struct {
	connOptions
	pattern          string
	maxConnNum       int
	handshakeTimeout time.Duration
	readTimeout      time.Duration
	writeTimeout     time.Duration
	checkOrigin      bool
}

func defaultServerOptions() serverOptions {
	return serverOptions{
		connOptions: defaultConnOptions(),
		pattern:     "/",
		checkOrigin: true,
	}
}

func (opts *serverOptions) ensure() {
	opts.connOptions.ensure()
}

type ServerOption func(o *serverOptions)

func ServerMaxWriteQueueSize(maxWriteQueueSize int) ServerOption {
	return func(o *serverOptions) {
		o.maxWriteQueueSize = maxWriteQueueSize
	}
}

func ServerWriteQueueShrinkSize(writeQueueShrinkSize int) ServerOption {
	return func(o *serverOptions) {
		o.writeQueueShrinkSize = writeQueueShrinkSize
	}
}

func ServerMaxReadMsgSize(maxReadMsgSize int) ServerOption {
	return func(o *serverOptions) {
		o.maxReadMsgSize = maxReadMsgSize
	}
}

func ServerMaxWriteMsgSize(maxWriteMsgSize int) ServerOption {
	return func(o *serverOptions) {
		o.maxWriteMsgSize = maxWriteMsgSize
	}
}

func ServerMsgType(msgType int) ServerOption {
	return func(o *serverOptions) {
		o.msgType = msgType
	}
}

func ServerKeepAlivePeriod(keepAlivePeriod time.Duration) ServerOption {
	return func(o *serverOptions) {
		o.keepAlivePeriod = keepAlivePeriod
	}
}

func ServerShutdownReadPeriod(shutdownReadPeriod time.Duration) ServerOption {
	return func(o *serverOptions) {
		o.shutdownReadPeriod = shutdownReadPeriod
	}
}

func ServerShutdownWritePeriod(shutdownWritePeriod time.Duration) ServerOption {
	return func(o *serverOptions) {
		o.shutdownWritePeriod = shutdownWritePeriod
	}
}

func ServerPattern(pattern string) ServerOption {
	return func(o *serverOptions) {
		o.pattern = pattern
	}
}

func ServerMaxConnNum(maxConnNum int) ServerOption {
	return func(o *serverOptions) {
		o.maxConnNum = maxConnNum
	}
}

func ServerHandshakeTimeout(handshakeTimeout time.Duration) ServerOption {
	return func(o *serverOptions) {
		o.handshakeTimeout = handshakeTimeout
	}
}

func ServerReadTimeout(readTimeout time.Duration) ServerOption {
	return func(o *serverOptions) {
		o.readTimeout = readTimeout
	}
}

func ServerWriteTimeout(writeTimeout time.Duration) ServerOption {
	return func(o *serverOptions) {
		o.writeTimeout = writeTimeout
	}
}

func ServerCheckOrigin(checkOrigin bool) ServerOption {
	return func(o *serverOptions) {
		o.checkOrigin = checkOrigin
	}
}

type clientOptions struct {
	connOptions
	autoRedial       bool
	redialInterval   time.Duration
	dialTimeout      time.Duration
	handshakeTimeout time.Duration
}

func defaultClientOptions() clientOptions {
	return clientOptions{
		connOptions:    defaultConnOptions(),
		autoRedial:     true,
		redialInterval: 3 * time.Second,
		dialTimeout:    3 * time.Second,
	}
}

func (opts *clientOptions) ensure() {
	opts.connOptions.ensure()
	if opts.autoRedial && opts.redialInterval <= 0 {
		panic("ws: clientOptions redialInterval <= 0")
	}
}

type ClientOption func(o *clientOptions)

func ClientMaxWriteQueueSize(maxWriteQueueSize int) ServerOption {
	return func(o *serverOptions) {
		o.maxWriteQueueSize = maxWriteQueueSize
	}
}

func ClientWriteQueueShrinkSize(writeQueueShrinkSize int) ServerOption {
	return func(o *serverOptions) {
		o.writeQueueShrinkSize = writeQueueShrinkSize
	}
}

func ClientMaxReadMsgSize(maxReadMsgSize int) ClientOption {
	return func(o *clientOptions) {
		o.maxReadMsgSize = maxReadMsgSize
	}
}

func ClientMaxWriteMsgSize(maxWriteMsgSize int) ClientOption {
	return func(o *clientOptions) {
		o.maxWriteMsgSize = maxWriteMsgSize
	}
}

func ClientMsgType(msgType int) ClientOption {
	return func(o *clientOptions) {
		o.msgType = msgType
	}
}

func ClientKeepAlivePeriod(keepAlivePeriod time.Duration) ClientOption {
	return func(o *clientOptions) {
		o.keepAlivePeriod = keepAlivePeriod
	}
}

func ClientShutdownReadPeriod(shutdownReadPeriod time.Duration) ClientOption {
	return func(o *clientOptions) {
		o.shutdownReadPeriod = shutdownReadPeriod
	}
}

func ClientShutdownWritePeriod(shutdownWritePeriod time.Duration) ClientOption {
	return func(o *clientOptions) {
		o.shutdownWritePeriod = shutdownWritePeriod
	}
}

func ClientAutoRedial(autoRedial bool) ClientOption {
	return func(o *clientOptions) {
		o.autoRedial = autoRedial
	}
}

func ClientRedialInterval(redialInterval time.Duration) ClientOption {
	return func(o *clientOptions) {
		o.redialInterval = redialInterval
	}
}

func ClientDialTimeout(dialTimeout time.Duration) ClientOption {
	return func(o *clientOptions) {
		o.dialTimeout = dialTimeout
	}
}

func ClientHandshakeTimeout(handshakeTimeout time.Duration) ClientOption {
	return func(o *clientOptions) {
		o.handshakeTimeout = handshakeTimeout
	}
}
