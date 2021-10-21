package log

import (
	"bufio"
	"github.com/hsgames/gold/container/queue"
	"os"
	"sync"
)

type StdLogger struct {
	*logger
	bw    *bufio.Writer
	queue *queue.MPSCQueue
	wg    sync.WaitGroup
}

func NewStdLogger(level uint64) *StdLogger {
	l := &StdLogger{
		bw:    bufio.NewWriterSize(os.Stderr, logWriterSize),
		queue: queue.NewMPSCQueue(0, logQueueShrinkSize),
	}
	l.logger = newLogger(l, level)
	l.wg.Add(1)
	go l.serve()
	return l
}

func (l *StdLogger) handle(info logInfo) {
	l.queue.Push(info)
}

func (l *StdLogger) Shutdown() {
	l.queue.Push(nil)
	l.wg.Wait()
}

func (l *StdLogger) serve() {
	var buf []byte
	defer l.wg.Done()
	defer l.bw.Flush()
	for {
		datas := l.queue.Pop()
		for _, data := range *datas {
			if data == nil {
				return
			}
			if cap(buf) >= logBufferSize {
				buf = []byte{}
			} else {
				buf = buf[:0]
			}
			l.formatLogInfo(&buf, data.(logInfo))
			l.bw.Write(buf)
		}
		l.bw.Flush()
	}
}
