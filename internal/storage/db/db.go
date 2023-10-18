package db

import (
	"context"
	"github.com/MaximMNsk/go-url-shortener/internal/util/logger"
	"github.com/MaximMNsk/go-url-shortener/server/config"
	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
)

var (
	db *pgx.Conn
)

var ctx context.Context

func Connect() error {
	ctx = context.Background()
	logger.PrintLog(logger.INFO, config.Config.Final.DB)
	database, err := pgx.Connect(ctx, config.Config.Final.DB)
	db = database
	return err
}

func GetDB() *pgx.Conn {
	return db
}

func GetCtx() context.Context {
	return ctx
}

func Close() error {
	err := db.Close(ctx)
	if err != nil {
		logger.PrintLog(logger.ERROR, "Can't close connection")
		return err
	}
	return nil
}
