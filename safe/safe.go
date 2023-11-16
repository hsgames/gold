package safe

import (
	"log/slog"
	"runtime/debug"
)

func Recover() {
	if r := recover(); r != nil {
		slog.Error("safe: panic recover",
			slog.Any("value", r), slog.String("stack", string(debug.Stack())))
	}
}

func Go(f func()) {
	go func() {
		defer Recover()
		f()
	}()
}
