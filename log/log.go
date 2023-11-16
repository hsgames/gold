package log

import (
	"io"
	"log/slog"
	"os"
)

type options struct {
	w    io.Writer
	opts *slog.HandlerOptions
	args []any
}

type Option func(*options)

func WithWriter(w io.Writer) Option {
	return func(opts *options) {
		opts.w = w
	}
}

func WithHandlerOptions(ho *slog.HandlerOptions) Option {
	return func(opts *options) {
		opts.opts = ho
	}
}

func WithSource(add bool) Option {
	return func(opts *options) {
		opts.opts.AddSource = add
	}
}

func WithLevel(level slog.Level) Option {
	return func(opts *options) {
		opts.opts.Level = level
	}
}

func WithAttrs(args ...any) Option {
	return func(opts *options) {
		clear(opts.args)
		opts.args = append(opts.args, args...)
	}
}

func Init(opt ...Option) {
	opts := options{
		w:    os.Stdout,
		opts: &slog.HandlerOptions{AddSource: true},
	}

	for _, v := range opt {
		v(&opts)
	}

	logger := slog.New(slog.NewJSONHandler(opts.w, opts.opts)).With(opts.args...)

	slog.SetDefault(logger)
}
