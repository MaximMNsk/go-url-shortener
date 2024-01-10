package main

import (
	"context"
	"github.com/MaximMNsk/go-url-shortener/internal/models/database"
	"github.com/MaximMNsk/go-url-shortener/internal/storage/db"
	"github.com/MaximMNsk/go-url-shortener/internal/util/extlogger"
	"github.com/MaximMNsk/go-url-shortener/internal/util/logger"
	"github.com/MaximMNsk/go-url-shortener/server/auth/cookie"
	"github.com/MaximMNsk/go-url-shortener/server/compress"
	confModule "github.com/MaximMNsk/go-url-shortener/server/config"
	"github.com/MaximMNsk/go-url-shortener/server/server"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"net/http"
)

/**
 * Request type handlers
 */

func main() {

	logger.PrintLog(logger.INFO, "Start newServ")
	logger.PrintLog(logger.INFO, "Handle config")

	ctx := context.Background()

	conf, err := confModule.HandleConfig()
	if err != nil {
		logger.PrintLog(logger.FATAL, "Can't handle config. "+err.Error())
	}

	var pgPool *pgxpool.Pool
	storage := server.ChooseStorage()
	if confModule.Config.Env.DB != `` || confModule.Config.Flag.DB != `` {
		pgPool, err = db.Connect(ctx)
		defer db.Close(pgPool)
		if err != nil {
			logger.PrintLog(logger.ERROR, "Failed connect to DB")
		}
		database.PrepareDB(pgPool, ctx)
	}
	newServ := server.NewServ(conf, storage, ctx, pgPool)

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
			r.Get(`/api/user/{query}`, newServ.HandleAPI)
		})
		newServ.Routers.Group(func(r chi.Router) {
			r.Use(cookie.AuthChecker)
			r.Delete(`/api/user/{query}`, newServ.HandleAPI)
		})
	})

	logger.PrintLog(logger.INFO, "Starting newServ")
	err = http.ListenAndServe(confModule.Config.Final.AppAddr, newServ.Routers)
	if err != nil {
		logger.PrintLog(logger.FATAL, "Can't start newServ. "+err.Error())
	}
}
