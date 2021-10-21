package app_test

import (
	"github.com/hsgames/gold/app"
	"github.com/hsgames/gold/log"
	"github.com/hsgames/gold/safe"
	"github.com/hsgames/gold/timer"
	"testing"
	"time"
)

type Handler struct {
	logger log.Logger
}

func NewHandler(logger log.Logger) app.Handler {
	return &Handler{logger: logger}
}

func (h *Handler) OnInit(tm *timer.Manager) error {
	h.logger.Info("handler init")
	tm.AddTicker(1*time.Second, func() {
		h.logger.Info("ticker run")
	})
	return nil
}

func (h *Handler) OnDestroy() {
	h.logger.Info("handler destroy")
}

func (h *Handler) OnMessage(m interface{}) {
	h.logger.Info("handler on message %#v", m)
}

func TestWorker(t *testing.T) {
	logger := log.NewFileLogger(log.InfoLevel, "./log", log.FileAlsoStderr(true))
	defer logger.Shutdown()
	worker := app.NewWorker("test", NewHandler(logger), logger, app.WorkerMsgChanSize(0))
	safe.Go(logger, func() {
		err := worker.Run()
		if err != nil {
			logger.Error("%+v", err)
		}
	})
	worker.Push("test message")
	time.Sleep(10 * time.Second)
	worker.Shutdown()
}

func TestShutdownBeforeRun(t *testing.T) {
	logger := log.NewFileLogger(log.InfoLevel, "./log", log.FileAlsoStderr(true))
	defer logger.Shutdown()
	worker := app.NewWorker("test", NewHandler(logger), logger)
	worker.Shutdown()
	err := worker.Run()
	if err != nil {
		logger.Error("%+v", err)
	}
}
