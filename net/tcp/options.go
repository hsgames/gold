package tcp

import (
	"fmt"
	"math"
	"time"
)

type connOptions struct {
	writeChanSize   int
	maxReadMsgSize  int
	maxWriteMsgSize int
	keepAlivePeriod time.Duration
	newReader       func() Reader
	newWriter       func() Writer
}

type options struct {
	connOptions

	maxConnNum int
}

func defaultOptions() options {
	return options{
		connOptions: connOptions{
			writeChanSize:   200,
			maxReadMsgSize:  math.MaxUint16,
			maxWriteMsgSize: math.MaxUint16,
			keepAlivePeriod: 3 * time.Minute,
			newReader:       defaultReader,
			newWriter:       defaultWriter,
		},
	}
}

func (o *options) check() error {
	if o.maxReadMsgSize <= 0 {
		return fmt.Errorf("tcp: options maxReadMsgSize [%d] <= 0", o.maxReadMsgSize)
	}

	if o.maxWriteMsgSize <= 0 {
		return fmt.Errorf("tcp: options maxWriteMsgSize [%d] <= 0", o.maxWriteMsgSize)
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

func WithKeepAlivePeriod(keepAlivePeriod time.Duration) Option {
	return func(o *options) {
		o.keepAlivePeriod = keepAlivePeriod
	}
}

func WithReader(newReader func() Reader) Option {
	return func(o *options) {
		o.newReader = newReader
	}
}

func WithWriter(newWriter func() Writer) Option {
	return func(o *options) {
		o.newWriter = newWriter
	}
}

func WithMaxConnNum(maxConnNum int) Option {
	return func(o *options) {
		o.maxConnNum = maxConnNum
	}
}
