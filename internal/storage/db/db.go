package db

import (
	"context"
	"github.com/MaximMNsk/go-url-shortener/server/config"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pashagolub/pgxmock/v3"
	"time"
)

func Connect(ctx context.Context, conf config.OuterConfig) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(conf.Final.DB)
	if err != nil {
		return nil, err
	}
	cfg.MaxConns = 16
	cfg.MinConns = 1
	cfg.HealthCheckPeriod = 1 * time.Minute
	cfg.MaxConnLifetime = 1 * time.Hour
	cfg.MaxConnIdleTime = 5 * time.Minute
	cfg.ConnConfig.ConnectTimeout = 2 * time.Second

	database, err := pgxpool.NewWithConfig(ctx, cfg)
	return database, err
}

func MockConnect(ctx context.Context, conf config.OuterConfig) (pgxmock.PgxPoolIface, error) {
	//cfg, err := pgxpool.ParseConfig(conf.Final.DB)
	//if err != nil {
	//	return nil, err
	//}
	//cfg.MaxConns = 16
	//cfg.MinConns = 1
	//cfg.HealthCheckPeriod = 1 * time.Minute
	//cfg.MaxConnLifetime = 1 * time.Hour
	//cfg.MaxConnIdleTime = 5 * time.Minute
	//cfg.ConnConfig.ConnectTimeout = 2 * time.Second

	database, err := pgxmock.NewPool()
	return database, err
}

func Close(DB *pgxpool.Pool) {
	DB.Close()
}
