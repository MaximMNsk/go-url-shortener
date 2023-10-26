package model

type Storable interface {
	Get() (string, error)
	Set() error
	BatchSet() ([]byte, error)
}
