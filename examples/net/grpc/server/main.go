package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/hsgames/gold/app"
	"github.com/hsgames/gold/examples/net/grpc/helloworld"
	"github.com/hsgames/gold/log"
	"github.com/hsgames/gold/net/grpc"
	"github.com/hsgames/gold/safe"
)

func NewTest1Middleware(logger log.Logger) grpc.Middleware {
	return func(h grpc.Handler) grpc.Handler {
		logger.Info("main: Test1Middleware run")
		return h
	}
}

func NewTest2Middleware(logger log.Logger) grpc.Middleware {
	return func(h grpc.Handler) grpc.Handler {
		logger.Info("main: Test2Middleware run")
		return h
	}
}

type Server struct {
	helloworld.UnimplementedGreeterServer
	logger log.Logger
}

func NewServer(logger log.Logger) *Server {
	return &Server{logger: logger}
}

func (s *Server) SayHello(ctx context.Context, in *helloworld.HelloRequest) (*helloworld.HelloReply, error) {
	s.logger.Info("SayHello:%+v", in)
	return &helloworld.HelloReply{Message: fmt.Sprintf("Hello %+v", in)}, nil
}

func main() {
	flag.Parse()
	logger := log.NewFileLogger(log.InfoLevel, "./log", log.FileAlsoStderr(true))
	defer logger.Shutdown()
	defer safe.Recover(logger)
	s := grpc.NewServer("grpc_server", "tcp", ":20000", logger,
		grpc.ServerChainUnaryInterceptor(
			NewTest1Middleware(logger),
			NewTest2Middleware(logger),
		))
	helloworld.RegisterGreeterServer(s, NewServer(logger))
	a := app.New(logger)
	a.AddPProf("pprof", "tcp", ":8888")
	a.AddService(
		func() error {
			err := s.Listen()
			if err != nil {
				return err
			}
			logger.Info("main: grpc server %s listen", s)
			err = s.Serve()
			if err != nil {
				return err
			}
			logger.Info("main: grpc server %s shutdown", s)
			return nil
		},
		func() error {
			s.Shutdown()
			return nil
		},
	)
	if err := a.Run(); err != nil {
		logger.Error("main: app run err: %+v", err)
	}
	logger.Info("main: grpc server exit")
}
