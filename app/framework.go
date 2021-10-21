package app

import (
	"flag"
	"github.com/hsgames/gold/log"
	"github.com/hsgames/gold/safe"
)

type Framework interface {
	Init() error
	Run() error
	Destroy() error
}

func DefaultFileLogger() log.Logger {
	var (
		logLevel      uint64
		logDir        string
		logAlsoStderr bool
	)
	flag.Uint64Var(&logLevel, "log_level", log.InfoLevel, "log level")
	flag.StringVar(&logDir, "log_dir", "./log", "log dir")
	flag.BoolVar(&logAlsoStderr, "log_also_stderr", true, "log also stderr")
	flag.Parse()
	return log.NewFileLogger(logLevel, logDir, log.FileAlsoStderr(logAlsoStderr))
}

func DefaultStdLogger() log.Logger {
	var logLevel uint64
	flag.Uint64Var(&logLevel, "log_level", log.InfoLevel, "log level")
	return log.NewStdLogger(logLevel)
}

func RunFramework(newFunc func(logger log.Logger) Framework, logger log.Logger) {
	defer logger.Shutdown()
	defer safe.Recover(logger)
	logger.Info("app: run framework start")
	f := newFunc(logger)
	err := f.Init()
	if err != nil {
		logger.Error("app: init err: %+v", err)
		return
	}
	err = f.Run()
	if err != nil {
		logger.Error("app: run err: %+v", err)
		return
	}
	err = f.Destroy()
	if err != nil {
		logger.Error("app: destroy err: %+v", err)
		return
	}
	logger.Info("app: run framework stop")
}
