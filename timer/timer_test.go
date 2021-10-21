package timer_test

import (
	"github.com/hsgames/gold/log"
	"github.com/hsgames/gold/timer"
	"testing"
	"time"
)

func TestManager_AddTimer(t *testing.T) {
	logger := log.NewFileLogger(log.InfoLevel, "./log", log.FileAlsoStderr(true))
	m := timer.NewManager(logger)
	m.AddTimer(2*time.Second, func() {
		logger.Info("timer run 2")
	})
	m.AddTimer(5*time.Second, func() {
		logger.Info("timer run")
	})
	ticker := time.NewTicker(10 * time.Millisecond)
	for {
		select {
		case <-ticker.C:
			m.Run(0)
		}
	}
}

func TestManager_AddTicker(t *testing.T) {
	logger := log.NewFileLogger(log.InfoLevel, "./log", log.FileAlsoStderr(true))
	m := timer.NewManager(logger)
	t1 := m.AddTicker(100*time.Millisecond, func() {
		logger.Info("ticker run 3")
	})
	t2 := m.AddTicker(2*time.Second, func() {
		logger.Info("ticker run 2")
	})
	t3 := m.AddTicker(5*time.Second, func() {
		logger.Info("ticker run 1")
	})
	ticker := time.NewTicker(10 * time.Millisecond)
	stop := time.NewTimer(5 * time.Second)
	for {
		select {
		case <-ticker.C:
			m.Run(0)
		case <-stop.C:
			m.Remove(t3)
			m.Remove(t1)
			m.Remove(t2)
		}
	}
}
