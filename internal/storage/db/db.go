package db

import (
	"context"
	"github.com/MaximMNsk/go-url-shortener/internal/util/logger"
	"github.com/MaximMNsk/go-url-shortener/server/config"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"net"
	"time"
)

var (
	DB *pgxpool.Pool
)

var ctx context.Context

func Connect() error {
	ctx = context.Background()
	logger.PrintLog(logger.INFO, config.Config.Final.DB)
	cfg, err := pgxpool.ParseConfig(config.Config.Final.DB)
	cfg.MaxConns = 8
	cfg.MinConns = 4
	cfg.HealthCheckPeriod = 1 * time.Minute
	cfg.MaxConnLifetime = 1 * time.Hour
	cfg.MaxConnIdleTime = 5 * time.Minute
	cfg.ConnConfig.ConnectTimeout = 1 * time.Second
	cfg.ConnConfig.DialFunc = (&net.Dialer{
		KeepAlive: cfg.HealthCheckPeriod,
		Timeout:   cfg.ConnConfig.ConnectTimeout,
	}).DialContext

	database, err := pgxpool.NewWithConfig(ctx, cfg)
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
