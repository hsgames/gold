package grpc

import (
	"context"
	"github.com/hsgames/gold/log"
	"github.com/hsgames/gold/safe"
	"google.golang.org/grpc"
)

type Handler func(ctx context.Context, req interface{}) (resp interface{}, err error)

type Middleware func(Handler) Handler

func NewRecoverMiddleware(logger log.Logger) Middleware {
	return func(h Handler) Handler {
		return func(ctx context.Context, req interface{}) (resp interface{}, err error) {
			defer func() {
				if err != nil && logger != nil {
					logger.Error("grpc: recover middleware err:%+v", err)
				}
			}()
			defer safe.RecoverError(&err)
			resp, err = h(ctx, req)
			return
		}
	}
}

func NewContextMiddleware(logger log.Logger) Middleware {
	return func(h Handler) Handler {
		return func(ctx context.Context, req interface{}) (resp interface{}, err error) {
			defer func() {
				if err != nil && logger != nil {
					logger.Error("grpc: context middleware err:%+v", err)
				}
			}()
			select {
			case <-ctx.Done():
				resp, err = nil, ctx.Err()
			default:
				resp, err = h(ctx, req)
			}
			return
		}
	}
}

func UnaryServerInterceptor(m Middleware) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{},
		info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		h := func(ctx context.Context, req interface{}) (interface{}, error) {
			return handler(ctx, req)
		}
		if m != nil {
			h = m(h)
		}
		resp, err := h(ctx, req)
		if err != nil {
			return nil, err
		}
		return resp, nil
	}
}

func UnaryClientInterceptor(m Middleware) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{},
		cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		h := func(ctx context.Context, req interface{}) (interface{}, error) {
			return reply, invoker(ctx, method, req, reply, cc, opts...)
		}
		if m != nil {
			h = m(h)
		}
		_, err := h(ctx, req)
		return err
	}
}
