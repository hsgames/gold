package tcp

import (
	"fmt"
	"math"
	"time"
)

type connOptions struct {
	maxWriteQueueSize    int
	writeQueueShrinkSize int
	maxReadMsgSize       int
	maxWriteMsgSize      int
	readBufSize          int
	writeBufSize         int
	keepAlivePeriod      time.Duration
	readDeadlinePeriod   time.Duration
	shutdownReadPeriod   time.Duration
	shutdownWritePeriod  time.Duration
}

func defaultConnOptions() connOptions {
	opts := connOptions{
		maxWriteQueueSize:    0,
		writeQueueShrinkSize: 1024 * 1024,
		maxReadMsgSize:       math.MaxUint16,
		maxWriteMsgSize:      math.MaxUint16,
		readBufSize:          math.MaxUint16,
		writeBufSize:         math.MaxUint16,
		keepAlivePeriod:      3 * time.Minute,
		shutdownReadPeriod:   5 * time.Second,
		shutdownWritePeriod:  30 * time.Second,
	}
	return opts
}

func (opts *connOptions) ensure() {
	if opts.maxReadMsgSize <= 0 {
		panic(fmt.Sprintf("tcp: connOptions maxReadMsgSize:%d <= 0", opts.maxReadMsgSize))
	}
	if opts.maxWriteMsgSize <= 0 {
		panic(fmt.Sprintf("tcp: connOptions maxWriteMsgSize:%d <= 0", opts.maxWriteMsgSize))
	}
	if opts.readBufSize < opts.maxReadMsgSize {
		panic(fmt.Sprintf("tcp: connOptions readBufSize:%d < maxReadMsgSize:%d",
			opts.readBufSize, opts.maxReadMsgSize))
	}
	if opts.writeBufSize < opts.maxWriteMsgSize {
		panic(fmt.Sprintf("tcp: connOptions writeBufSize:%d < maxWriteMsgSize:%d",
			opts.writeBufSize, opts.maxWriteMsgSize))
	}
	if opts.shutdownReadPeriod <= 0 {
		panic(fmt.Sprintf("tcp: connOptions shutdownReadPeriod:%d <= 0",
			opts.shutdownReadPeriod))
	}
	if opts.shutdownWritePeriod <= 0 {
		panic(fmt.Sprintf("tcp: connOptions shutdownWritePeriod:%d <= 0",
			opts.shutdownWritePeriod))
	}
}

type serverOptions struct {
	connOptions
	maxConnNum int
}

func defaultServerOptions() serverOptions {
	return serverOptions{
		connOptions: defaultConnOptions(),
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

func ServerReadBufSize(readBufSize int) ServerOption {
	return func(o *serverOptions) {
		o.readBufSize = readBufSize
	}
}

func ServerWriteBufSize(writeBufSize int) ServerOption {
	return func(o *serverOptions) {
		o.writeBufSize = writeBufSize
	}
}

func ServerKeepAlivePeriod(keepAlivePeriod time.Duration) ServerOption {
	return func(o *serverOptions) {
		o.keepAlivePeriod = keepAlivePeriod
	}
}

func ServerReadDeadlinePeriod(readDeadlinePeriod time.Duration) ServerOption {
	return func(o *serverOptions) {
		o.readDeadlinePeriod = readDeadlinePeriod
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

func ServerMaxConnNum(maxConnNum int) ServerOption {
	return func(o *serverOptions) {
		o.maxConnNum = maxConnNum
	}
}

type clientOptions struct {
	connOptions
	autoRedial     bool
	redialInterval time.Duration
	dialTimeout    time.Duration
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
		panic("tcp: clientOptions redialInterval <= 0")
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

func ClientReadBufSize(readBufSize int) ClientOption {
	return func(o *clientOptions) {
		o.readBufSize = readBufSize
	}
}

func ClientWriteBufSize(writeBufSize int) ClientOption {
	return func(o *clientOptions) {
		o.writeBufSize = writeBufSize
	}
}

func ClientKeepAlivePeriod(keepAlivePeriod time.Duration) ClientOption {
	return func(o *clientOptions) {
		o.keepAlivePeriod = keepAlivePeriod
	}
}

func ClientReadDeadlinePeriod(readDeadlinePeriod time.Duration) ClientOption {
	return func(o *clientOptions) {
		o.readDeadlinePeriod = readDeadlinePeriod
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
