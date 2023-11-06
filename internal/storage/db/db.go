package db

import (
	"context"
	"github.com/MaximMNsk/go-url-shortener/internal/util/logger"
	"github.com/MaximMNsk/go-url-shortener/server/config"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
)

var (
	DB *pgxpool.Pool
)

var ctx context.Context

func Connect() error {
	ctx = context.Background()
	logger.PrintLog(logger.INFO, config.Config.Final.DB)
	//database, err := pgx.Connect(ctx, config.Config.Final.DB)
	database, err := pgxpool.New(ctx, config.Config.Final.DB)
	DB = database
	return err
}

func GetDB() *pgxpool.Pool {
	return DB
}

func GetCtx() context.Context {
	return ctx
}

func Close() {
	DB.Close()
}
