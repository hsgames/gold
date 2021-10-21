package grpc

import (
	"context"
	"google.golang.org/grpc"
)

func NewClient(target string, opt ...ClientOption) (*grpc.ClientConn, error) {
	opts := defaultClientOptions()
	for _, o := range opt {
		o(&opts)
	}
	opts.ensure()
	ctx := context.Background()
	var dialOpts []grpc.DialOption
	if len(opts.chainUnaryInts) > 0 {
		dialOpts = append(dialOpts, grpc.WithChainUnaryInterceptor(opts.chainUnaryInts...))
	}
	if opts.timeout > 0 {
		ctx, _ = context.WithTimeout(ctx, opts.timeout)
	}
	if opts.insecure {
		dialOpts = append(dialOpts, grpc.WithInsecure())
	}
	if opts.block {
		dialOpts = append(dialOpts, grpc.WithBlock())
	}
	if opts.transportCredentials != nil {
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(opts.transportCredentials))
	}
	dialOpts = append(dialOpts, grpc.WithKeepaliveParams(opts.KeepaliveParams))
	var callOptions []grpc.CallOption
	if opts.maxReceiveMessageSize > 0 {
		callOptions = append(callOptions, grpc.MaxCallRecvMsgSize(opts.maxReceiveMessageSize))
	}
	if opts.maxSendMessageSize > 0 {
		callOptions = append(callOptions, grpc.MaxCallSendMsgSize(opts.maxSendMessageSize))
	}
	if len(callOptions) > 0 {
		dialOpts = append(dialOpts, grpc.WithDefaultCallOptions(callOptions...))
	}
	return grpc.DialContext(ctx, target, dialOpts...)
}
