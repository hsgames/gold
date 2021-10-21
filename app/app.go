package app

import (
	"context"
	"github.com/hsgames/gold/log"
	"github.com/hsgames/gold/net/http"
	"github.com/hsgames/gold/safe"
	"os"
	"os/signal"
	"sync"
)

type service struct {
	start    func() error
	stop     func() error
	doneChan chan error
}

type App struct {
	opts     options
	services []*service
	cancel   context.CancelFunc
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
		start:    start,
		stop:     stop,
		doneChan: make(chan error, 2),
	}
	a.services = append(a.services, w)
}

func (a *App) Run() error {
	var errOnce sync.Once
	errChan := make(chan error, 1)
	for _, v := range a.services {
		s := v
		safe.Go(a.logger, func() {
			var err error
			defer func() {
				s.doneChan <- err
				if err != nil {
					errOnce.Do(func() { errChan <- err })
				}
			}()
			defer safe.RecoverError(&err)
			err = s.start()
		})
	}
	var (
		err  error
		done bool
	)
	c := make(chan os.Signal, 1)
	signal.Notify(c, a.opts.sigs...)
	for {
		select {
		case err = <-errChan:
			goto end
		case sig := <-c:
			done, err = func() (done bool, err error) {
				defer safe.RecoverError(&err)
				done = a.opts.sigHandler(a, sig)
				return
			}()
			if err != nil {
				a.logger.Error("app: app handle signal err:%+v", err)
				goto end
			}
			if done {
				goto end
			}
		}
	}

end:
	for i := len(a.services) - 1; i >= 0; i-- {
		s := a.services[i]
		func() {
			var err error
			defer func() {
				if err != nil {
					s.doneChan <- err
				}
			}()
			defer safe.RecoverError(&err)
			err = s.stop()
		}()
	}
	for i, v := range a.services {
		if err := <-v.doneChan; err != nil {
			a.logger.Error("app: app worker %d done err: %+v", i, err)
		}
	}
	return err
}
