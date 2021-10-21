package app

import (
	"os"
	"syscall"
	"time"
)

type SignalHandler func(a *App, sig os.Signal) (done bool)

type options struct {
	sigs       []os.Signal
	sigHandler SignalHandler
}

func defaultOptions() options {
	return options{
		sigs: []os.Signal{
			syscall.SIGTERM,
			syscall.SIGQUIT,
			syscall.SIGINT,
		},
		sigHandler: func(a *App, sig os.Signal) bool {
			switch sig {
			case syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGINT:
				a.logger.Info("app: handle shutdown signal: %s", sig)
				return true
			default:
				a.logger.Info("app: unhandled signal: %s", sig)
				return false
			}
		},
	}
}

func (opts *options) ensure() {
	if opts.sigHandler == nil {
		panic("app: options sigHandler == nil")
	}
}

type Option func(o *options)

func AddSignals(sigs ...os.Signal) Option {
	return func(o *options) {
		m := make(map[os.Signal]struct{})
		for _, v := range o.sigs {
			m[v] = struct{}{}
		}
		for _, v := range sigs {
			if _, ok := m[v]; !ok {
				m[v] = struct{}{}
				o.sigs = append(o.sigs, v)
			}
		}
	}
}

func SetSignalHandler(sigHandler SignalHandler) Option {
	return func(o *options) {
		o.sigHandler = sigHandler
	}
}

type workerOptions struct {
	msgChanSize    int
	tickerDuration time.Duration
	tickerLimit    int
}

func defaultWorkerOptions() workerOptions {
	ops := workerOptions{
		msgChanSize:    2000,
		tickerDuration: time.Second,
		tickerLimit:    0,
	}
	return ops
}

func (opts *workerOptions) ensure() {
	if opts.tickerDuration <= 10*time.Millisecond {
		panic("app: workerOptions tickerDuration <= 10 ms")
	}
}

type WorkerOption func(o *workerOptions)

func WorkerMsgChanSize(msgChanSize int) WorkerOption {
	return func(o *workerOptions) {
		o.msgChanSize = msgChanSize
	}
}

func WorkerTickerDuration(tickerDuration time.Duration) WorkerOption {
	return func(o *workerOptions) {
		o.tickerDuration = tickerDuration
	}
}

func WorkerTickerLimit(tickerLimit int) WorkerOption {
	return func(o *workerOptions) {
		o.tickerLimit = tickerLimit
	}
}
