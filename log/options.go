package log

import "time"

type fileOptions struct {
	maxSize       uint64
	alsoStderr    bool
	prefix        string
	flushInterval time.Duration
}

func defaultFileOptions() fileOptions {
	return fileOptions{
		maxSize:       64 * 1024 * 1024,
		flushInterval: 5 * time.Second,
	}
}

func (opts *fileOptions) ensure() {
	if opts.maxSize == 0 {
		panic("log: opts.maxSize == 0")
	}
	if opts.flushInterval <= 0 {
		panic("log: opts.flushInterval <= 0")
	}
}

type FileOption func(o *fileOptions)

func FileMaxSize(maxSize uint64) FileOption {
	return func(o *fileOptions) {
		o.maxSize = maxSize
	}
}

func FileAlsoStderr(alsoStderr bool) FileOption {
	return func(o *fileOptions) {
		o.alsoStderr = alsoStderr
	}
}

func FilePrefix(prefix string) FileOption {
	return func(o *fileOptions) {
		o.prefix = prefix
	}
}

func FileFlushInterval(flushInterval time.Duration) FileOption {
	return func(o *fileOptions) {
		o.flushInterval = flushInterval
	}
}
