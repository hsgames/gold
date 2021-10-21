package main

import (
	"context"
	"flag"
	"github.com/hsgames/gold/examples/net/grpc/helloworld"
	"github.com/hsgames/gold/log"
	"github.com/hsgames/gold/net/grpc"
	"github.com/hsgames/gold/safe"
	"time"
)

func NewTest1Middleware(logger log.Logger) grpc.Middleware {
	return func(handler grpc.Handler) grpc.Handler {
		logger.Info("Test1Middleware")
		return handler
	}
}

func main() {
	flag.Parse()
	logger := log.NewFileLogger(log.InfoLevel, "./log", log.FileAlsoStderr(true))
	defer logger.Shutdown()
	defer safe.Recover(logger)
	conn, err := grpc.NewClient("localhost:20000", grpc.ClientInsecure(),
		grpc.ClientChainUnaryInterceptor(NewTest1Middleware(logger)),
	)
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	c := helloworld.NewGreeterClient(conn)
	for {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		r, err := c.SayHello(ctx, &helloworld.HelloRequest{Name: "abc"})
		if err != nil {
			cancel()
			logger.Info("rpc error %+v", err)
			time.Sleep(1 * time.Second)
			continue
		}
		cancel()
		logger.Info("rpc response %s", r.Message)
		time.Sleep(1 * time.Second)
	}
}
