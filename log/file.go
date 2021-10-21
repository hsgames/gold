package log

import (
	"bufio"
	"context"
	"fmt"
	"github.com/hsgames/gold/container/queue"
	"github.com/pkg/errors"
	"os"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type FileLogger struct {
	*logger
	opts     fileOptions
	file     *os.File
	dir      string
	name     string
	bytes    uint64
	flushing int32
	cancel   context.CancelFunc
	bw       *bufio.Writer
	queue    *queue.MPSCQueue
	wg       sync.WaitGroup
}

func NewFileLogger(level uint64, dir string, opt ...FileOption) *FileLogger {
	opts := defaultFileOptions()
	for _, o := range opt {
		o(&opts)
	}
	opts.ensure()
	if !strings.HasSuffix(dir, "/") && !strings.HasSuffix(dir, "\\") {
		dir += "/"
	}
	l := &FileLogger{
		opts:  opts,
		dir:   dir,
		queue: queue.NewMPSCQueue(0, logQueueShrinkSize),
	}
	l.logger = newLogger(l, level)
	var ctx context.Context
	ctx, l.cancel = context.WithCancel(context.Background())
	l.wg.Add(2)
	go l.serve()
	go l.daemon(ctx)
	return l
}

func (l *FileLogger) Shutdown() {
	l.queue.Push(nil)
	l.cancel()
	l.wg.Wait()
}

func (l *FileLogger) handle(info logInfo) {
	l.queue.Push(info)
}

func (l *FileLogger) createDirs() error {
	_, err := os.Stat(l.dir)
	if err != nil {
		if os.IsNotExist(err) {
			err = os.MkdirAll(l.dir, 0775)
			if err != nil {
				return errors.Wrapf(err, "log: %s create dirs", l.dir)
			}
			return nil
		}
		return errors.Wrapf(err, "log: %s stat", l.dir)
	}
	return nil
}

func (l *FileLogger) fileName(t time.Time) string {
	name := fmt.Sprintf("%s.%04d%02d%02d-%02d%02d%02d.%d.log",
		program,
		t.Year(),
		t.Month(),
		t.Day(),
		t.Hour(),
		t.Minute(),
		t.Second(),
		pid)
	if len(l.opts.prefix) > 0 {
		name = l.opts.prefix + "." + name
	}
	return name
}

func (l *FileLogger) createFile() error {
	var (
		n    int
		err  error
		file *os.File
	)
	err = l.closeFile()
	l.name = ""
	l.bytes = 0
	l.file = nil
	l.bw = nil
	if err != nil {
		return err
	}
	now := time.Now()
	name := l.fileName(now)
	path := l.dir + name
	_, err = os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			file, err = os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0644)
		}
	} else {
		file, err = os.OpenFile(path, os.O_APPEND|os.O_RDWR, 0644)
	}
	if err != nil {
		return errors.Wrapf(err, "log: %s open file", name)
	}
	header := fmt.Sprintf("Log file created at: %s\n"+
		"Running on machine: %s\n"+
		"User name: %s\n"+
		"Process id: %d\n"+
		"Binary: Built with %s %s for %s/%s\n"+
		"Log line format: [DIWE]yyyymmdd hh:mm:ss.uuuuuu file:line] msg\n",
		now.Format("2006/01/02 15:04:05"),
		host, userName, pid,
		runtime.Compiler, runtime.Version(), runtime.GOOS, runtime.GOARCH)
	n, err = file.Write([]byte(header))
	if err != nil {
		return errors.Wrapf(err, "log: %s write header", name)
	}
	l.bytes += uint64(n)
	l.name = name
	l.file = file
	l.bw = bufio.NewWriterSize(l.file, logWriterSize)
	return nil
}

func (l *FileLogger) flushFile() error {
	if l.file != nil {
		var err error
		err = l.bw.Flush()
		if err != nil {
			return errors.Wrapf(err, "log: %s flush", l.name)
		}
		err = l.file.Sync()
		if err != nil {
			return errors.Wrapf(err, "log: %s sync", l.name)
		}
	}
	return nil
}

func (l *FileLogger) closeFile() error {
	if l.file != nil {
		var err error
		err = l.flushFile()
		if err != nil {
			return err
		}
		err = l.file.Close()
		if err != nil {
			return errors.Wrapf(err, "log: %s close", l.name)
		}
	}
	return nil
}

func (l *FileLogger) writeFile(p []byte) error {
	if l.file == nil {
		return errors.New("log: no file")
	}
	var (
		err error
		n   int
	)
	if l.bytes+uint64(len(p)) >= l.opts.maxSize {
		if err = l.createFile(); err != nil {
			return err
		}
	}
	n, err = l.bw.Write(p)
	if err != nil {
		return errors.Wrapf(err, "log: %s write", l.name)
	}
	l.bytes += uint64(n)
	if l.opts.alsoStderr {
		os.Stderr.Write(p)
	}
	return nil
}

func (l *FileLogger) serve() {
	var (
		err error
		buf []byte
	)
	defer l.wg.Done()
	defer func() {
		err = l.closeFile()
		if err != nil {
			l.printError(err)
		}
	}()
	err = l.createDirs()
	if err != nil {
		l.panic(err)
	}
	err = l.createFile()
	if err != nil {
		l.panic(err)
	}
	for {
		datas := l.queue.Pop()
		for _, data := range *datas {
			if data == nil {
				return
			}
			_, ok := data.(struct{})
			if ok {
				err = l.flushFile()
				if err != nil {
					l.printError(err)
				}
				atomic.StoreInt32(&l.flushing, 0)
				continue
			}
			if cap(buf) >= logBufferSize {
				buf = []byte{}
			} else {
				buf = buf[:0]
			}
			l.formatLogInfo(&buf, data.(logInfo))
			err = l.writeFile(buf)
			if err != nil {
				l.printError(err)
				os.Stderr.Write(buf)
			}
		}
	}
}

func (l *FileLogger) daemon(ctx context.Context) {
	defer l.wg.Done()
	ticker := time.NewTicker(l.opts.flushInterval)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if atomic.CompareAndSwapInt32(&l.flushing, 0, 1) {
				l.queue.Push(struct{}{})
			}
		}
	}
}

func (l *FileLogger) panic(err error) {
	panic(fmt.Sprintf("%+v", err))
}

func (l *FileLogger) printError(err error) {
	os.Stderr.WriteString(fmt.Sprintf("%+v", err))
}
