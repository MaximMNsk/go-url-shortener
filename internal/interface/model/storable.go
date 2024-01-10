package model

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Storable interface {
	Init(link, shortLink, id string, isDeleted bool, ctx context.Context, pool *pgxpool.Pool)
	Get() (string, bool, error)
	Set() error
	Ping() bool
	BatchSet() ([]byte, error)
	HandleUserUrls() ([]byte, error)
	HandleUserUrlsDelete()
	AsyncSaver()
}
