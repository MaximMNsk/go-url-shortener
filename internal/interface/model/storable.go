package model

import "context"

type Storable interface {
	Init(link, shortLink, id string, isDeleted bool, ctx context.Context)
	Get() (string, bool, error)
	Set() error
	BatchSet() ([]byte, error)
	HandleUserUrls() ([]byte, error)
	HandleUserUrlsDelete() error
}
