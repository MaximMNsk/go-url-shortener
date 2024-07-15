// Package db - создает и закрывает коннект-пул с БД
package db

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/MaximMNsk/go-url-shortener/server/config"
)

// Pool - интерфейс пула соединений.
//
//go:generate go run github.com/vektra/mockery/v2@v2.43.0 --name=Pool
type Pool interface {
	Ping(ctx context.Context) error
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	Close()
}

// NewPool - создает пул подключений к БД.
// Использует данные из переданной конфигурации.
func NewPool(ctx context.Context, conf config.OuterConfig) (Pool, error) {
	cfg, err := pgxpool.ParseConfig(conf.Final.DB)
	if err != nil {
		return nil, err
	}
	cfg.MaxConns = 16
	cfg.MinConns = 1
	cfg.HealthCheckPeriod = 1 * time.Minute
	cfg.MaxConnLifetime = 1 * time.Hour
	cfg.MaxConnIdleTime = 1 * time.Minute
	cfg.ConnConfig.ConnectTimeout = 20 * time.Second

	database, err := pgxpool.NewWithConfig(ctx, cfg)
	return database, err
}
