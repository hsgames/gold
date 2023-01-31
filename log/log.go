package log

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"time"
)

const (
	DebugLevel uint64 = iota
	InfoLevel
	WarnLevel
	ErrorLevel
)

const (
	logWriterSize      = 1024 * 1024
	logBufferSize      = 1024 * 16
	logQueueShrinkSize = 1024 * 16
)

var (
	pid       = os.Getpid()
	program   = filepath.Base(os.Args[0])
	host      = "unknownhost"
	userName  = "unknownuser"
	LevelName = []string{"D", "I", "W", "E"}
)

func init() {
	h, err := os.Hostname()
	if err == nil {
		host = h
	}
	current, err := user.Current()
	if err == nil {
		userName = current.Username
	}
}

type Logger interface {
	Shutdown()
	SetLevel(level uint64)
	Debug(format string, v ...any)
	Info(format string, v ...any)
	Warn(format string, v ...any)
	Error(format string, v ...any)
}

type handler interface {
	handle(info logInfo)
}

type logInfo struct {
	time  time.Time
	level uint64
	file  string
	line  int
	s     string
}

type logger struct {
	h     handler
	level uint64
}

func newLogger(h handler, level uint64) *logger {
	return &logger{h: h, level: level}
}

func (l *logger) SetLevel(level uint64) {
	atomic.StoreUint64(&l.level, level)
}

func fmtLog(format string, v ...any) string {
	if len(v) == 0 {
		return format
	}
	return fmt.Sprintf(format, v...)
}

func (l *logger) Debug(format string, v ...any) {
	l.output(DebugLevel, 2, fmtLog(format, v...))
}

func (l *logger) Info(format string, v ...any) {
	l.output(InfoLevel, 2, fmtLog(format, v...))
}

func (l *logger) Warn(format string, v ...any) {
	l.output(WarnLevel, 2, fmtLog(format, v...))
}

func (l *logger) Error(format string, v ...any) {
	l.output(ErrorLevel, 2, fmtLog(format, v...))
}

func (l *logger) output(level uint64, calldepth int, s string) {
	if level < atomic.LoadUint64(&l.level) {
		return
	}
	_, file, line, ok := runtime.Caller(calldepth)
	if !ok {
		file = "???"
		line = 0
	}
	l.h.handle(logInfo{
		time:  time.Now(),
		level: level,
		file:  file,
		line:  line,
		s:     s,
	})
}

func itoa(buf *[]byte, i int, wid int) {
	var b [20]byte
	bp := len(b) - 1
	for i >= 10 || wid > 1 {
		wid--
		q := i / 10
		b[bp] = byte('0' + i - q*10)
		bp--
		i = q
	}
	b[bp] = byte('0' + i)
	*buf = append(*buf, b[bp:]...)
}

func (l *logger) formatHeader(buf *[]byte, level uint64, t time.Time, file string, line int) {
	*buf = append(*buf, LevelName[level]...)
	year, month, day := t.Date()
	itoa(buf, year, 4)
	itoa(buf, int(month), 2)
	itoa(buf, day, 2)
	*buf = append(*buf, ' ')
	hour, min, sec := t.Clock()
	itoa(buf, hour, 2)
	*buf = append(*buf, ':')
	itoa(buf, min, 2)
	*buf = append(*buf, ':')
	itoa(buf, sec, 2)
	*buf = append(*buf, '.')
	itoa(buf, t.Nanosecond()/1e3, 6)
	*buf = append(*buf, ' ')
	short := file
	for i := len(file) - 1; i > 0; i-- {
		if file[i] == '/' {
			short = file[i+1:]
			break
		}
	}
	file = short
	*buf = append(*buf, file...)
	*buf = append(*buf, ':')
	itoa(buf, line, -1)
	*buf = append(*buf, "] "...)
}

func (l *logger) formatLogInfo(buf *[]byte, info logInfo) {
	l.formatHeader(buf, info.level, info.time, info.file, info.line)
	*buf = append(*buf, info.s...)
	if len(info.s) == 0 || info.s[len(info.s)-1] != '\n' {
		*buf = append(*buf, '\n')
	}
}
