package http

import (
	"github.com/hsgames/gold/log"
	"time"
)

type serverOptions struct {
	chainUnaryInts  []Middleware
	readTimeout     time.Duration
	writeTimeout    time.Duration
	allowAllOrigins bool
}

func defaultServerOptions(logger log.Logger) serverOptions {
	ops := serverOptions{
		allowAllOrigins: true,
	}
	ops.chainUnaryInts = append(ops.chainUnaryInts, NewRecoverMiddleware(logger))
	return ops
}

func (opts *serverOptions) ensure() {
}

type ServerOption func(o *serverOptions)

func ServerClearChainUnaryInterceptor() ServerOption {
	return func(o *serverOptions) {
		o.chainUnaryInts = []Middleware{}
	}
}

func ServerChainUnaryInterceptor(ms ...Middleware) ServerOption {
	return func(o *serverOptions) {
		o.chainUnaryInts = append(o.chainUnaryInts, ms...)
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

func ServerAllowAllOrigins(allowAllOrigins bool) ServerOption {
	return func(o *serverOptions) {
		o.allowAllOrigins = allowAllOrigins
	}
}

type clientOptions struct {
	timeout time.Duration
}

func defaultClientOptions() clientOptions {
	return clientOptions{}
}

func (opts *clientOptions) ensure() {
}

type ClientOption func(o *clientOptions)

func ClientTimeout(timeout time.Duration) ClientOption {
	return func(o *clientOptions) {
		o.timeout = timeout
	}
}
