package memory

import (
	"encoding/json"
	"errors"
	"github.com/MaximMNsk/go-url-shortener/internal/util/logger"
	"github.com/MaximMNsk/go-url-shortener/internal/util/shorter"
	confModule "github.com/MaximMNsk/go-url-shortener/server/config"
)

type MemStorage struct {
	Link      string `json:"original_url"`
	ShortLink string `json:"short_url"`
	ID        string `json:"correlation_id"`
}

var storage = make(map[string]MemStorage)

func (jsonData *MemStorage) Get() (string, error) {
	logger.PrintLog(logger.INFO, "Get from memory")
	for _, v := range storage {
		if v.ID == jsonData.ID || v.Link == jsonData.Link {
			return v.Link, nil
		}
	}
	return "", errors.New("data not found")
}

func (jsonData *MemStorage) Set() error {
	logger.PrintLog(logger.INFO, "Set to memory")

	for _, v := range storage {
		if v.Link == jsonData.Link {
			return nil
		}
	}

	storage[jsonData.ID] = *jsonData

	return nil
}

func (jsonData *MemStorage) BatchSet() ([]byte, error) {

	var savingData []MemStorage

	err := json.Unmarshal([]byte(jsonData.Link), &savingData)
	if err != nil {
		return nil, err
	}

	for i, v := range savingData {
		//savingData[i].ID = sha1hash.Create(v.Link, 8)
		savingData[i].ShortLink = shorter.GetShortURL(confModule.Config.Final.ShortURLAddr, v.ID)
		storage[savingData[i].ID] = savingData[i]
	}

	JSONResp, err := json.Marshal(savingData)
	if err != nil {
		logger.PrintLog(logger.WARN, err.Error())
	}

	return JSONResp, nil
}
