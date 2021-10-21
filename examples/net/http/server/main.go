package main

import (
	"flag"
	"github.com/hsgames/gold/app"
	"github.com/hsgames/gold/log"
	"github.com/hsgames/gold/net/http"
	"github.com/hsgames/gold/safe"
	"time"
)

func NewTest1Middleware(logger log.Logger) http.Middleware {
	return func(h http.Handler) http.Handler {
		logger.Info("Test1Middleware")
		return h
	}
}

func NewTest2Middleware(logger log.Logger) http.Middleware {
	return func(h http.Handler) http.Handler {
		logger.Info("NewTest2Middleware")
		return h
	}
}

func main() {
	flag.Parse()
	logger := log.NewFileLogger(log.InfoLevel, "./log", log.FileAlsoStderr(true))
	defer logger.Shutdown()
	defer safe.Recover(logger)
	s := http.NewServer("http_server", "tcp", "",
		http.Handlers{
			"/hello": func(w http.ResponseWriter, r *http.Request) {
				logger.Info("hello")
				w.Write([]byte("hello"))
			},
			"/world": func(w http.ResponseWriter, r *http.Request) {
				logger.Info("world")
				w.Write([]byte("world"))
			},
			"/app": func(w http.ResponseWriter, r *http.Request) {
				name := r.FormValue("name")
				logger.Info("app recv %s", name)
				w.Write([]byte("hello " + name))
			},
		},
		logger,
		http.ServerChainUnaryInterceptor(
			NewTest1Middleware(logger),
			NewTest2Middleware(logger)))
	app := app.New(logger)
	app.AddPProf("pprof", "tcp", ":8888")
	app.AddService(
		func() error {
			err := s.Listen()
			if err != nil {
				return err
			}
			logger.Info("main: http server %s listen", s)
			err = s.Serve()
			if err != nil {
				return err
			}
			logger.Info("main: http server %s shutdown", s)
			return nil
		},
		func() error {
			s.Shutdown()
			return nil
		},
	)
	if err := app.Run(); err != nil {
		logger.Error("main: app run err: %+v", err)
	}
	logger.Info("main: http server exit")
	time.Sleep(10 * time.Second)
}
