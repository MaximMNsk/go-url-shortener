package memory

import (
	"github.com/MaximMNsk/go-url-shortener/internal/util/logger"
)

type JSONData struct {
	Link      string
	ShortLink string
	ID        string
}

var storage []JSONData

func (jsonData *JSONData) Get() error {
	logger.PrintLog(logger.INFO, "Get from memory")
	for _, v := range storage {
		if v.ID == jsonData.ID || v.Link == jsonData.Link {
			jsonData.ID = v.ID
			jsonData.Link = v.Link
			jsonData.ShortLink = v.ShortLink
		}
	}
	return nil
}

func (jsonData *JSONData) Set() error {
	logger.PrintLog(logger.INFO, "Set to memory")
	for _, v := range storage {
		if v.ID == jsonData.ID {
			return nil
		}
	}

	storage = append(storage, *jsonData)

	return nil
}
