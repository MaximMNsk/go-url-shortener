package main

import (
	"context"
	"errors"
	"github.com/MaximMNsk/go-url-shortener/internal/util/extlogger"
	"github.com/MaximMNsk/go-url-shortener/internal/util/logger"
	"github.com/MaximMNsk/go-url-shortener/server/auth/cookie"
	"github.com/MaximMNsk/go-url-shortener/server/compress"
	confModule "github.com/MaximMNsk/go-url-shortener/server/config"
	"github.com/MaximMNsk/go-url-shortener/server/server"
	"github.com/go-chi/chi/v5"
	"net/http"
)

/**
 * Request type handlers
 */

func main() {

	logger.PrintLog(logger.INFO, "Start newServ")
	logger.PrintLog(logger.INFO, "Handle config")

	ctx := context.Background()

	var conf confModule.OuterConfig
	err := conf.InitConfig(false)
	if err != nil {
		logger.PrintLog(logger.FATAL, "Can't handle config. "+err.Error())
		return
	}

	storage, err := server.ChooseStorage(ctx, conf)
	if err != nil {
		var serverHandlersErr *server.ErrorHandlers
		if errors.As(err, &serverHandlersErr) {
			logger.PrintLog(logger.FATAL, "Can't handle storage.")
			logger.PrintLog(logger.FATAL, err.Error())
			return
		}
		logger.PrintLog(logger.FATAL, "Can't create storage environment.")
		logger.PrintLog(logger.FATAL, errors.Unwrap(err).Error())
		return
	}

	defer storage.Destroy()

	newServ := server.NewServ(conf, storage, ctx)

	logger.PrintLog(logger.INFO, "Declaring router")

	newServ.Routers = chi.NewRouter().
		With(extlogger.Log).
		With(compress.GzipHandler).
		With(server.HandleOther)
	newServ.Routers.Route("/", func(r chi.Router) {
		newServ.Routers.Group(func(r chi.Router) {
			r.Use(cookie.AuthSetter)
			r.Post(`/`, newServ.HandlePOST)
			r.Post(`/api/{query}`, newServ.HandleAPI)
			r.Post(`/api/shorten/{query}`, newServ.HandleAPI)
			r.Get(`/ping`, newServ.HandlePing)
			r.Get(`/{query}`, newServ.HandleGET)
		})
		newServ.Routers.Group(func(r chi.Router) {
			r.Use(cookie.AuthChecker)
			r.Delete(`/api/user/{query}`, newServ.HandleAPI)
			r.Get(`/api/user/{query}`, newServ.HandleAPI)
		})
	})

	logger.PrintLog(logger.INFO, "Starting newServ")
	err = http.ListenAndServe(conf.Final.AppAddr, newServ.Routers)
	if err != nil {
		logger.PrintLog(logger.FATAL, "Can't start newServ. "+err.Error())
	}
}
