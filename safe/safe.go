package safe

import (
	"github.com/hsgames/gold/log"
	"github.com/pkg/errors"
	"runtime"
)

func Stacktrace() string {
	buf := make([]byte, 2<<20)
	n := runtime.Stack(buf, false)
	return string(buf[:n])
}

func Recover(logger log.Logger) {
	if r := recover(); r != nil {
		logger.Error("panic recover, %+v\n%s", r, Stacktrace())
	}
}

func RecoverError(err *error) {
	if r := recover(); r != nil {
		switch r.(type) {
		case error:
			*err = errors.WithStack(r.(error))
		default:
			*err = errors.Errorf("panic %v", r)
		}
	}
}

func Go(logger log.Logger, f func()) {
	go func() {
		defer Recover(logger)
		f()
	}()
}
