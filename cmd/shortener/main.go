package main

import (
	"context"
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
		logger.PrintLog(logger.INFO, "Config error", true)
		return
	}

	ctx := context.Background()

	var serv server.Server
	err = serv.Init(conf, true)
	if err != nil {
		logger.PrintLog(logger.INFO, "Server init error", true)
		return
	}

	exit := make(chan os.Signal, 1)
	signal.Notify(exit, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGABRT, syscall.SIGINT, syscall.SIGSEGV)
	go func() {
		for {
			select {
			case <-exit:
				logger.PrintLog(logger.INFO, "Stopping server", serv.LogEnabled)
				stopped := serv.Stop(ctx)
				if stopped != nil {
					logger.PrintLog(logger.INFO, "Do not stop server", serv.LogEnabled)
				}
				return
			case <-time.After(100 * time.Millisecond):
				continue
			}
		}
	}()

	logger.PrintLog(logger.INFO, `Start server`, serv.LogEnabled)
	err = serv.Start(ctx)
	if err != nil {
		logger.PrintLog(logger.ERROR, err.Error(), serv.LogEnabled)
	}
}
