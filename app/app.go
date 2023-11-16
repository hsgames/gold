package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"
)

type Service interface {
	Name() string
	Init() error
	Start() error
	Stop() error
}

type App struct {
	signals   []os.Signal
	services  []Service
	preStart  []func() error
	postStart []func() error
	preStop   []func() error
	postStop  []func() error
}

func New(opt ...Option) *App {
	a := &App{
		signals: []os.Signal{
			syscall.SIGTERM,
			syscall.SIGINT,
			syscall.SIGQUIT,
			syscall.SIGKILL,
		},
	}

	for _, v := range opt {
		v(a)
	}

	return a
}

func (a *App) Run(ctx context.Context) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("gold: app run panic [%v] stack [%s]", r, string(debug.Stack()))
		}
	}()

	for _, f := range a.preStart {
		if err = f(); err != nil {
			return
		}
	}

	for _, s := range a.services {
		if err = s.Init(); err != nil {
			err = fmt.Errorf("gold: app service [%s] init err [%w]", s.Name(), err)
			return
		}
	}

	errs := make(chan error, len(a.services))
	for _, s := range a.services {
		s := s

		go func() {
			defer func() {
				if r := recover(); r != nil {
					errs <- fmt.Errorf("gold: app service [%s] panic [%v]", s.Name(), r)
				}
			}()

			if err := s.Start(); err != nil {
				errs <- fmt.Errorf("gold: app service [%s] start err [%w]", s.Name(), err)
			}
		}()

		slog.Info("gold: app service is running", slog.String("name", s.Name()))
	}

	for _, f := range a.postStart {
		if err = f(); err != nil {
			return
		}
	}

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, a.signals...)

	select {
	case err = <-errs:
	case sig := <-ch:
		slog.Info("gold: app quit", slog.String("signal", sig.String()))
	case <-ctx.Done():
	}

	for _, f := range a.preStop {
		if e := f(); e != nil {
			err = errors.Join(err, e)
		}
	}

	for _, s := range a.services {
		if e := s.Stop(); e != nil {
			err = errors.Join(err, fmt.Errorf("gold: app service [%s] stop err [%w]", s.Name(), e))
		} else {
			slog.Info("gold: app service is stopped", slog.String("name", s.Name()))
		}
	}

	for _, f := range a.postStop {
		if e := f(); e != nil {
			err = errors.Join(err, e)
		}
	}

	return
}
