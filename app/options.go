package app

import (
	"os"
	"time"
)

// SignalHandler if return error, the app will be shutdown
type SignalHandler func(a *App, sig os.Signal) error

type options struct {
	sigs       []os.Signal
	sigHandler SignalHandler
}

func defaultOptions() options {
	return options{}
}

func (opts *options) ensure() {
}

type Option func(o *options)

func SetUserSignals(handler SignalHandler, sigs ...os.Signal) Option {
	return func(o *options) {
		o.sigs = o.sigs[0:0]
		o.sigs = append(o.sigs, sigs...)
		o.sigHandler = handler
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
