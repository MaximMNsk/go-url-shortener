package model

import "context"

type Storable interface {
	Init(link, shortLink, id string, ctx context.Context)
	Get() (string, error)
	Set() error
	BatchSet() ([]byte, error)
	HandleUserUrls() ([]byte, error)
}
