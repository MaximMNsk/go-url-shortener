package models

import (
	"context"
)

type Storable interface {
	Init(link, shortLink, id string, isDeleted bool, ctx context.Context)
	Get() (string, bool, error)
	Set() error
	Ping() (bool, error)
	BatchSet() ([]byte, error)
	HandleUserUrls() ([]byte, error)
	HandleUserUrlsDelete()
	AsyncSaver()
	Destroy()
}
