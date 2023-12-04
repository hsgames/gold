package tcp

import (
	"errors"
	"fmt"
	"math"
	"time"
)

type connOptions struct {
	writeChanSize    int
	maxReadDataSize  int
	maxWriteDataSize int
	keepAlivePeriod  time.Duration
	newReader        func() Reader
	newWriter        func() Writer
	getReadData      func(size int) []byte
	putWriteData     func(b []byte)
}

type options struct {
	connOptions

	maxConnNum int
}

func defaultOptions() options {
	return options{
		connOptions: connOptions{
			writeChanSize:    200,
			maxReadDataSize:  math.MaxUint16,
			maxWriteDataSize: math.MaxUint16,
			keepAlivePeriod:  3 * time.Minute,
			newReader:        defaultReader,
			newWriter:        defaultWriter,
			getReadData:      func(size int) []byte { return make([]byte, size) },
			putWriteData:     func(b []byte) {},
		},
	}
}

func (o *options) check() error {
	if o.maxReadDataSize <= 0 {
		return fmt.Errorf("tcp: options maxReadDataSize [%d] <= 0", o.maxReadDataSize)
	}

	if o.maxWriteDataSize <= 0 {
		return fmt.Errorf("tcp: options maxWriteDataSize [%d] <= 0", o.maxWriteDataSize)
	}

	if o.newReader == nil {
		return errors.New("tcp: options newReader is nil")
	}

	if o.newWriter == nil {
		return errors.New("tcp: options newWriter is nil")
	}

	if o.getReadData == nil {
		return errors.New("tcp: options getReadData is nil")
	}

	if o.putWriteData == nil {
		return errors.New("tcp: options putWriteData is nil")
	}

	return nil
}

type Option func(o *options)

func WithWriteChanSize(writeChanSize int) Option {
	return func(o *options) {
		o.writeChanSize = writeChanSize
	}
}

func WithMaxReadDataSize(maxReadDataSize int) Option {
	return func(o *options) {
		o.maxReadDataSize = maxReadDataSize
	}
}

func WithMaxWriteDataSize(maxWriteDataSize int) Option {
	return func(o *options) {
		o.maxWriteDataSize = maxWriteDataSize
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

func GetReadData(getReadData func(size int) []byte) Option {
	return func(o *options) {
		o.getReadData = getReadData
	}
}

func PutWriteData(putWriteData func(b []byte)) Option {
	return func(o *options) {
		o.putWriteData = putWriteData
	}
}

func WithMaxConnNum(maxConnNum int) Option {
	return func(o *options) {
		o.maxConnNum = maxConnNum
	}
}
