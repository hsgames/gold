package app

import (
	"github.com/hsgames/gold/log"
	"github.com/hsgames/gold/net/http"
	"github.com/hsgames/gold/safe"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

type service struct {
	start func() error
	stop  func() error
	wg    sync.WaitGroup
}

type App struct {
	opts     options
	services []*service
	logger   log.Logger
}

func New(logger log.Logger, opt ...Option) *App {
	opts := defaultOptions()
	for _, o := range opt {
		o(&opts)
	}
	opts.ensure()
	return &App{
		opts:   opts,
		logger: logger,
	}
}

func (a *App) AddPProf(name, network, addr string, opt ...http.ServerOption) {
	s := http.NewPProfServer(name, network, addr, a.logger, opt...)
	a.AddService(
		func() error {
			err := s.Listen()
			if err != nil {
				return err
			}
			a.logger.Info("app: pprof server %s listen", s)
			err = s.Serve()
			if err != nil {
				return err
			}
			a.logger.Info("app: pprof server %s shutdown", s)
			return nil
		},
		func() error {
			s.Shutdown()
			return nil
		},
	)
}

func (a *App) AddService(start, stop func() error) {
	if start == nil {
		panic("app: app add service start func is nil")
	}
	if stop == nil {
		panic("app: app add service stop func is nil")
	}
	w := &service{
		start: start,
		stop:  stop,
	}
	a.services = append(a.services, w)
}

func (a *App) Run() error {
	var errOnce sync.Once
	errChan := make(chan error, 1)
	for _, v := range a.services {
		s := v
		s.wg.Add(1)
		safe.Go(a.logger, func() {
			defer s.wg.Done()
			var err error
			defer func() {
				if err != nil {
					errOnce.Do(func() { errChan <- err })
				}
			}()
			defer safe.RecoverError(&err)
			err = s.start()
		})
	}
	var err error
	shutdownCh := make(chan os.Signal, 1)
	signal.Notify(shutdownCh, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGINT)
	userSigCh := make(chan os.Signal, 1)
	if len(a.opts.sigs) > 0 {
		signal.Notify(userSigCh, a.opts.sigs...)
	}
	for {
		select {
		case err = <-errChan:
			a.logger.Error("app: service run err: %+v", err)
			goto end
		case sig := <-shutdownCh:
			switch sig {
			case syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGINT:
				a.logger.Info("app: receive shutdown signal: %s", sig)
				goto end
			}
		case sig := <-userSigCh:
			if a.opts.sigHandler != nil {
				err = func() error {
					defer safe.Recover(a.logger)
					return a.opts.sigHandler(a, sig)
				}()
				if err != nil {
					goto end
				}
			}
		}
	}

end:
	for i := len(a.services) - 1; i >= 0; i-- {
		s := a.services[i]
		func() {
			defer s.wg.Wait()
			defer safe.Recover(a.logger)
			err = s.stop()
			if err != nil {
				a.logger.Error("app: service stop err: %+v", err)
			}
		}()
	}
	return err
}
