package models

import (
	"context"
	"github.com/MaximMNsk/go-url-shortener/server/config"
)

type Storable interface {
	Init(link, shortLink, id string, isDeleted bool, ctx context.Context, cfg config.OuterConfig)
	Get() (string, bool, error)
	Set() error
	Ping() (bool, error)
	BatchSet() ([]byte, error)
	HandleUserUrls() ([]byte, error)
	HandleUserUrlsDelete()
	AsyncSaver()
	Destroy()
}
