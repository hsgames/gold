package main

import (
	"flag"
	"github.com/hsgames/gold/log"
	"github.com/hsgames/gold/net/http"
	"github.com/hsgames/gold/safe"
	"time"
)

func main() {
	flag.Parse()
	logger := log.NewFileLogger(log.InfoLevel, "./log", log.FileAlsoStderr(true))
	defer logger.Shutdown()
	defer safe.Recover(logger)
	c := http.NewClient(logger, http.ClientTimeout(3*time.Second))
	data, err := c.PostForm("http://localhost/app", http.Form{"name": "abcd"})
	if err != nil {
		panic(err)
	}
	logger.Info("http: response %s", data)
	logger.Info("http: client exit")
}
