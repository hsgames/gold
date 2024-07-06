package ws

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

type connOptions struct {
	writeChanSize   int
	readLimit       int
	dataType        int
	keepAlivePeriod time.Duration
}

type options struct {
	connOptions

	maxConnNum       int
	pattern          string
	checkOrigin      func(_ *http.Request) bool
	handshakeTimeout time.Duration
	readTimeout      time.Duration
	writeTimeout     time.Duration
}

func defaultOptions() options {
	return options{
		connOptions: connOptions{writeChanSize: 128,
			dataType:        websocket.BinaryMessage,
			keepAlivePeriod: 3 * time.Minute,
		},
		pattern:     "/",
		checkOrigin: func(_ *http.Request) bool { return true },
	}
}

func (o *options) check() error {
	if o.readLimit < 0 {
		return fmt.Errorf("ws: options readLimit:[%d] < 0", o.readLimit)
	}

	return nil
}

type Option func(o *options)

func WithWriteChanSize(writeChanSize int) Option {
	return func(o *options) {
		o.writeChanSize = writeChanSize
	}
}

func WithReadLimit(readLimit int) Option {
	return func(o *options) {
		o.readLimit = readLimit
	}
}

func WithBinary() Option {
	return func(o *options) {
		o.dataType = websocket.BinaryMessage
	}
}

func WithText() Option {
	return func(o *options) {
		o.dataType = websocket.TextMessage
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

func WithCheckOrigin(checkOrigin func(_ *http.Request) bool) Option {
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
