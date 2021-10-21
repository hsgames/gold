package app

import (
	"github.com/hsgames/gold/log"
	"github.com/hsgames/gold/safe"
	"github.com/hsgames/gold/timer"
	"github.com/pkg/errors"
	"sync"
	"time"
)

type Handler interface {
	OnInit(*timer.Manager) error
	OnDestroy()
	OnMessage(interface{})
}

type Worker struct {
	opts     workerOptions
	tm       *timer.Manager
	name     string
	handler  Handler
	msgChan  chan interface{}
	doneChan chan struct{}
	doneOnce sync.Once
	shutdown bool
	run      bool
	mu       sync.Mutex
	wg       sync.WaitGroup
	logger   log.Logger
}

func NewWorker(name string, handler Handler, logger log.Logger, opt ...WorkerOption) *Worker {
	if handler == nil {
		panic("app: NewWorker handler is nil")
	}
	opts := defaultWorkerOptions()
	for _, o := range opt {
		o(&opts)
	}
	opts.ensure()
	return &Worker{
		opts:     opts,
		tm:       timer.NewManager(logger),
		name:     name,
		handler:  handler,
		msgChan:  make(chan interface{}, opts.msgChanSize),
		doneChan: make(chan struct{}),
		logger:   logger,
	}
}

func (w *Worker) String() string {
	return w.name
}

func (w *Worker) done() {
	w.doneOnce.Do(func() { close(w.doneChan) })
}

func (w *Worker) Run() (err error) {
	defer safe.RecoverError(&err)
	w.mu.Lock()
	if w.shutdown {
		w.mu.Unlock()
		return errors.Errorf("app: worker %s already shutdown", w)
	}
	if w.run {
		w.mu.Unlock()
		return errors.Errorf("app: worker %s already run", w)
	}
	w.run = true
	w.wg.Add(1)
	defer w.wg.Done()
	w.mu.Unlock()
	defer w.done()
	defer w.handler.OnDestroy()
	err = w.handler.OnInit(w.tm)
	if err != nil {
		return
	}
	ticker := time.NewTicker(w.opts.tickerDuration)
	defer ticker.Stop()
	for {
		select {
		case m := <-w.msgChan:
			if m == nil {
				return
			}
			func() {
				defer safe.Recover(w.logger)
				w.handler.OnMessage(m)
			}()
		case <-ticker.C:
			w.tm.Run(w.opts.tickerLimit)
		}
	}
}

func (w *Worker) Push(m interface{}) {
	select {
	case w.msgChan <- m:
	case <-w.doneChan:
	}
}

func (w *Worker) Shutdown() {
	w.mu.Lock()
	if w.shutdown {
		w.mu.Unlock()
		return
	}
	w.shutdown = true
	if !w.run {
		w.done()
		w.mu.Unlock()
		return
	}
	w.mu.Unlock()
	w.Push(nil)
	w.wg.Wait()
}
