package app

import "os"

type Option func(*App)

func WithSignals(signals ...os.Signal) Option {
	return func(a *App) {
		clear(a.signals)
		a.signals = append(a.signals, signals...)
	}
}

func WithServices(services ...Service) Option {
	return func(a *App) {
		a.services = append(a.services, services...)
	}
}

func PreStart(f func() error) Option {
	return func(a *App) {
		a.preStart = append(a.preStart, f)
	}
}

func PostStart(f func() error) Option {
	return func(a *App) {
		a.postStart = append(a.postStart, f)
	}
}

func PreStop(f func() error) Option {
	return func(a *App) {
		a.preStop = append(a.preStop, f)
	}
}

func PostStop(f func() error) Option {
	return func(a *App) {
		a.postStop = append(a.postStop, f)
	}
}
