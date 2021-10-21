package log

import (
	"testing"
)

func TestFileLogger(t *testing.T) {
	logger := NewFileLogger(InfoLevel, "./dir", FileAlsoStderr(true))
	defer logger.Shutdown()
	for i := 1; i < 100; i++ {
		logger.Info("%d %s %d", 1111, "你好你好", i)
	}
}

func BenchmarkFileLogger(b *testing.B) {
	logger := NewFileLogger(InfoLevel, "./dir")
	defer logger.Shutdown()
	for i := 0; i < b.N; i++ {
		logger.Info("%d %s %d", 1111, "你好你好", i)
	}
}

func TestStdLogger(t *testing.T) {
	logger := NewStdLogger(InfoLevel)
	defer logger.Shutdown()
	for i := 1; i < 100; i++ {
		logger.Info("%d %s %d", 1111, "你好你好", i)
	}
}
