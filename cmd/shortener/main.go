package main

import (
	"github.com/MaximMNsk/go-url-shortener/internal/models/database"
	"github.com/MaximMNsk/go-url-shortener/internal/storage/db"
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

	conf, err := confModule.HandleConfig()
	if err != nil {
		logger.PrintLog(logger.FATAL, "Can't handle config. "+err.Error())
	}

	if confModule.Config.Env.DB != `` || confModule.Config.Flag.DB != `` {
		err := db.Connect()
		if err != nil {
			logger.PrintLog(logger.ERROR, "Failed connect to DB")
		}
		database.PrepareDB(db.GetDB())
		defer func() {
			err := db.Close()
			if err != nil {
				logger.PrintLog(logger.ERROR, "Failed close connection to DB")
			}
		}()
	}

	storage := server.InitStorage()
	newServ := server.NewServ(conf, storage)

	logger.PrintLog(logger.INFO, "Declaring router")

	newServ.Routers = chi.NewRouter().
		With(extlogger.Log).
		With(compress.GzipHandler).
		With(server.HandleOther)
	newServ.Routers.Route("/", func(r chi.Router) {
		r.Post(`/`, newServ.HandlePOST)
		r.Post(`/api/{query}`, newServ.HandleAPI)
		r.Post(`/api/shorten/{query}`, newServ.HandleAPI)
		r.Get(`/ping`, newServ.HandlePing)
		r.Get(`/{query}`, newServ.HandleGET)
		newServ.Routers.Group(func(r chi.Router) {
			r.Use(cookie.AuthHandler)
			r.Get(`/api/user/{query}`, newServ.HandleAPI)
		})
	})

	logger.PrintLog(logger.INFO, "Starting newServ")

	err = http.ListenAndServe(confModule.Config.Final.AppAddr, newServ.Routers)
	if err != nil {
		logger.PrintLog(logger.FATAL, "Can't start newServ. "+err.Error())
	}
}
