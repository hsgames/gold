package safe

import (
	"log/slog"
	"runtime"
)

func Stack() string {
	buf := make([]byte, 2<<20)
	n := runtime.Stack(buf, false)
	return string(buf[:n])
}

func Recover() {
	if r := recover(); r != nil {
		slog.Error("panic recover",
			slog.Any("value", r), slog.String("stack", Stack()))
	}
}

func Go(f func()) {
	go func() {
		defer Recover()
		f()
	}()
}
