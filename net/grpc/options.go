package grpc

import (
	"github.com/hsgames/gold/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
	"time"
)

type serverOptions struct {
	chainUnaryInts []grpc.UnaryServerInterceptor
	creds          credentials.TransportCredentials
	maxRecvMsgSize int
	maxSendMsgSize int
}

func defaultServerOptions(logger log.Logger) serverOptions {
	opts := serverOptions{}
	opts.chainUnaryInts = append(opts.chainUnaryInts,
		UnaryServerInterceptor(NewRecoverMiddleware(logger)),
		UnaryServerInterceptor(NewContextMiddleware(logger)),
	)
	return opts
}

func (opts *serverOptions) ensure() {
}

type ServerOption func(o *serverOptions)

func ServerClearChainUnaryInterceptor() ServerOption {
	return func(o *serverOptions) {
		o.chainUnaryInts = []grpc.UnaryServerInterceptor{}
	}
}

func ServerChainUnaryInterceptor(ms ...Middleware) ServerOption {
	return func(o *serverOptions) {
		for _, m := range ms {
			o.chainUnaryInts = append(o.chainUnaryInts, UnaryServerInterceptor(m))
		}
	}
}

func ServerCreds(creds credentials.TransportCredentials) ServerOption {
	return func(o *serverOptions) {
		o.creds = creds
	}
}

func ServerMaxRecvMsgSize(maxRecvMsgSize int) ServerOption {
	return func(o *serverOptions) {
		o.maxRecvMsgSize = maxRecvMsgSize
	}
}

func ServerMaxSendMsgSize(maxSendMsgSize int) ServerOption {
	return func(o *serverOptions) {
		o.maxSendMsgSize = maxSendMsgSize
	}
}

type KeepaliveClientParameters = keepalive.ClientParameters

type clientOptions struct {
	chainUnaryInts        []grpc.UnaryClientInterceptor
	timeout               time.Duration
	insecure              bool
	block                 bool
	transportCredentials  credentials.TransportCredentials
	KeepaliveParams       KeepaliveClientParameters
	maxReceiveMessageSize int
	maxSendMessageSize    int
}

func defaultClientOptions() clientOptions {
	opts := clientOptions{
		timeout:  3 * time.Second,
		insecure: false,
		KeepaliveParams: KeepaliveClientParameters{
			Time:                10 * time.Second,
			Timeout:             3 * time.Second,
			PermitWithoutStream: true,
		},
	}
	opts.chainUnaryInts = append(opts.chainUnaryInts,
		UnaryClientInterceptor(NewRecoverMiddleware(nil)),
		UnaryClientInterceptor(NewContextMiddleware(nil)),
	)
	return opts
}

func (opts *clientOptions) ensure() {
}

type ClientOption func(o *clientOptions)

func ClientClearChainUnaryInterceptor() ClientOption {
	return func(o *clientOptions) {
		o.chainUnaryInts = []grpc.UnaryClientInterceptor{}
	}
}

func ClientChainUnaryInterceptor(ms ...Middleware) ClientOption {
	return func(o *clientOptions) {
		for _, m := range ms {
			o.chainUnaryInts = append(o.chainUnaryInts, UnaryClientInterceptor(m))
		}
	}
}

func ClientTimeOut(timeout time.Duration) ClientOption {
	return func(o *clientOptions) {
		o.timeout = timeout
	}
}

func ClientInsecure() ClientOption {
	return func(o *clientOptions) {
		o.insecure = true
	}
}

func ClientBlock() ClientOption {
	return func(o *clientOptions) {
		o.block = true
	}
}

func ClientKeepaliveParams(kp KeepaliveClientParameters) ClientOption {
	return func(o *clientOptions) {
		o.KeepaliveParams = kp
	}
}

func ClientTransportCredentials(transportCredentials credentials.TransportCredentials) ClientOption {
	return func(o *clientOptions) {
		o.transportCredentials = transportCredentials
	}
}

func ClientMaxRecvMsgSize(bytes int) ClientOption {
	return func(o *clientOptions) {
		o.maxReceiveMessageSize = bytes
	}
}

func ClientMaxSendMsgSize(bytes int) ClientOption {
	return func(o *clientOptions) {
		o.maxSendMessageSize = bytes
	}
}
