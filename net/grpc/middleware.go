package grpc

import (
	"context"
	"github.com/hsgames/gold/log"
	"github.com/hsgames/gold/safe"
	"google.golang.org/grpc"
)

type Handler func(ctx context.Context, req any) (resp any, err error)

type Middleware func(Handler) Handler

func NewRecoverMiddleware(logger log.Logger) Middleware {
	return func(h Handler) Handler {
		return func(ctx context.Context, req any) (resp any, err error) {
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
		return func(ctx context.Context, req any) (resp any, err error) {
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
	return func(ctx context.Context, req any,
		info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		h := func(ctx context.Context, req any) (any, error) {
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
	return func(ctx context.Context, method string, req, reply any,
		cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		h := func(ctx context.Context, req any) (any, error) {
			return reply, invoker(ctx, method, req, reply, cc, opts...)
		}
		if m != nil {
			h = m(h)
		}
		_, err := h(ctx, req)
		return err
	}
}
