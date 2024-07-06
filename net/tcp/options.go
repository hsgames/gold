package tcp

import (
	"errors"
	"fmt"
	"time"
)

type connOptions struct {
	writeChanSize   int
	readLimit       int
	keepAlivePeriod time.Duration
	newReader       func() Reader
	newWriter       func() Writer
	withReadPool    bool
}

type options struct {
	connOptions

	maxConnNum int
}

func defaultOptions() options {
	return options{
		connOptions: connOptions{
			writeChanSize:   128,
			keepAlivePeriod: 3 * time.Minute,
			newReader:       defaultReader,
			newWriter:       defaultWriter,
			withReadPool:    false,
		},
	}
}

func (o *options) check() error {
	if o.readLimit < 0 {
		return fmt.Errorf("tcp: options readLimit [%d] < 0", o.readLimit)
	}

	if o.newReader == nil {
		return errors.New("tcp: options newReader is nil")
	}

	if o.newWriter == nil {
		return errors.New("tcp: options newWriter is nil")
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

func WithReadPool() Option {
	return func(o *options) {
		o.withReadPool = true
	}
}

func WithMaxConnNum(maxConnNum int) Option {
	return func(o *options) {
		o.maxConnNum = maxConnNum
	}
}
