package db

import (
	"context"
	"github.com/MaximMNsk/go-url-shortener/server/config"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"time"
)

// Connect - создает пул подключений к БД.
// Использует данные из переданной конфигурации.
func Connect(ctx context.Context, conf config.OuterConfig) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(conf.Final.DB)
	if err != nil {
		return nil, err
	}
	cfg.MaxConns = 4
	cfg.MinConns = 1
	cfg.HealthCheckPeriod = 5 * time.Second
	cfg.MaxConnLifetime = 1 * time.Hour
	cfg.MaxConnIdleTime = 1 * time.Minute
	cfg.ConnConfig.ConnectTimeout = 10 * time.Second

	database, err := pgxpool.NewWithConfig(ctx, cfg)
	return database, err
}

// Close - закрывает переданный пул подключений.
func Close(DB *pgxpool.Pool) {
	DB.Close()
}
