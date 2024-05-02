package main

import (
	"github.com/MaximMNsk/go-url-shortener/internal/util/logger"
	confModule "github.com/MaximMNsk/go-url-shortener/server/config"
	"github.com/MaximMNsk/go-url-shortener/server/server"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	var conf confModule.OuterConfig
	err := conf.InitConfig(false)
	if err != nil {
		logger.PrintLog(logger.INFO, "Config error")
	}

	var serv server.Server
	serv.Config = conf

	exit := make(chan os.Signal, 1)
	signal.Notify(exit, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGABRT, syscall.SIGINT, syscall.SIGSEGV)
	go func() {
		for {
			select {
			case <-exit:
				logger.PrintLog(logger.INFO, "Stopping server")
				stopped := serv.Stop()
				if stopped != nil {
					logger.PrintLog(logger.INFO, "Do not stop server")
				}
				return
			case <-time.After(1 * time.Second):
				continue
			}
		}
	}()

	logger.PrintLog(logger.INFO, `Start server`)
	err = serv.Start()
	if err != nil {
		logger.PrintLog(logger.ERROR, err.Error())
	}
}
