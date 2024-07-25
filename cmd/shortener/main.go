// Package main - основной пакет приложения.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/MaximMNsk/go-url-shortener/internal/util/logger"
	confModule "github.com/MaximMNsk/go-url-shortener/server/config"
	"github.com/MaximMNsk/go-url-shortener/server/server"
)

// go build -ldflags "-X main.buildVersion=1.21 -X 'main.buildDate=$(date +'%Y/%m/%d %H:%M:%S')' -X main.buildCommit=Iter21" main.go
var (
	buildVersion string
	buildDate    string
	buildCommit  string
)

func main() {
	if buildVersion == `` {
		buildVersion = `N/A`
	}
	if buildDate == `` {
		buildDate = `N/A`
	}
	if buildCommit == `` {
		buildCommit = `N/A`
	}
	fmt.Println(`Build version:`, buildVersion)
	fmt.Println(`Build date:`, buildDate)
	fmt.Println(`Build commit:`, buildCommit)

	var conf confModule.OuterConfig
	err := conf.InitConfig(false)
	if err != nil {
		logger.PrintLog(logger.INFO, "Config error", true)
		return
	}

	ctx := context.Background()

	var serv server.Server
	err = serv.Init(ctx, conf, false)
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
	err = serv.Start()
	if err != nil {
		logger.PrintLog(logger.ERROR, err.Error(), serv.LogEnabled)
	}
}
