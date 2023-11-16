package ws

import (
	"fmt"
	"math"
	"time"

	"github.com/gorilla/websocket"
)

type connOptions struct {
	writeChanSize   int
	maxReadMsgSize  int
	maxWriteMsgSize int
	msgType         int
	keepAlivePeriod time.Duration
}

type options struct {
	connOptions

	maxConnNum       int
	pattern          string
	checkOrigin      bool
	handshakeTimeout time.Duration
	readTimeout      time.Duration
	writeTimeout     time.Duration
}

func defaultOptions() options {
	return options{
		connOptions: connOptions{writeChanSize: 200,
			maxReadMsgSize:  math.MaxUint16,
			maxWriteMsgSize: math.MaxUint16,
			msgType:         websocket.BinaryMessage,
			keepAlivePeriod: 3 * time.Minute,
		},
		pattern:     "/",
		checkOrigin: true,
	}
}

func (o *options) check() error {
	if o.maxReadMsgSize <= 0 {
		return fmt.Errorf("ws: options maxReadMsgSize:[%d] <= 0", o.maxReadMsgSize)
	}

	if o.maxWriteMsgSize <= 0 {
		return fmt.Errorf("ws: options maxWriteMsgSize:[%d] <= 0", o.maxWriteMsgSize)
	}

	return nil
}

type Option func(o *options)

func WithWriteChanSize(writeChanSize int) Option {
	return func(o *options) {
		o.writeChanSize = writeChanSize
	}
}

func WithMaxReadMsgSize(maxReadMsgSize int) Option {
	return func(o *options) {
		o.maxReadMsgSize = maxReadMsgSize
	}
}

func WithMaxWriteMsgSize(maxWriteMsgSize int) Option {
	return func(o *options) {
		o.maxWriteMsgSize = maxWriteMsgSize
	}
}

func WithBinaryMessage() Option {
	return func(o *options) {
		o.msgType = websocket.BinaryMessage
	}
}

func WithTextMessage() Option {
	return func(o *options) {
		o.msgType = websocket.TextMessage
	}
}

func WithKeepAlivePeriod(keepAlivePeriod time.Duration) Option {
	return func(o *options) {
		o.keepAlivePeriod = keepAlivePeriod
	}
}

func WithPattern(pattern string) Option {
	return func(o *options) {
		o.pattern = pattern
	}
}

func WithMaxConnNum(maxConnNum int) Option {
	return func(o *options) {
		o.maxConnNum = maxConnNum
	}
}

func WithCheckOrigin(checkOrigin bool) Option {
	return func(o *options) {
		o.checkOrigin = checkOrigin
	}
}

func WithHandshakeTimeout(handshakeTimeout time.Duration) Option {
	return func(o *options) {
		o.handshakeTimeout = handshakeTimeout
	}
}

func WithReadTimeout(readTimeout time.Duration) Option {
	return func(o *options) {
		o.readTimeout = readTimeout
	}
}

func WithWriteTimeout(writeTimeout time.Duration) Option {
	return func(o *options) {
		o.writeTimeout = writeTimeout
	}
}
