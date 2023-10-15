package db

import (
	"context"
	"github.com/MaximMNsk/go-url-shortener/internal/util/logger"
	"github.com/MaximMNsk/go-url-shortener/server/config"
	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
)

var (
	DB *pgx.Conn
)

func Connect() error {
	ctx := context.Background()
	logger.PrintLog(logger.INFO, config.Config.Final.DB)
	database, err := pgx.Connect(ctx, config.Config.Final.DB)
	DB = database
	return err
}

func GetDB() *pgx.Conn {
	return DB
}

func Close() {
	ctx := context.Background()
	err := DB.Close(ctx)
	if err != nil {
		logger.PrintLog(logger.ERROR, "Can't close connection")
	}
}
