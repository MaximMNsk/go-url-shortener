package memory

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/MaximMNsk/go-url-shortener/internal/storage/memory"
	"github.com/MaximMNsk/go-url-shortener/internal/util/logger"
	"github.com/MaximMNsk/go-url-shortener/internal/util/shorter"
	confModule "github.com/MaximMNsk/go-url-shortener/server/config"
	"sync"
)

type MemStorage struct {
	Link      string `json:"original_url"`
	ShortLink string `json:"short_url"`
	ID        string `json:"correlation_id"`
	Ctx       context.Context
	Storage   memoryStorage.Storage
}

func (jsonData *MemStorage) Init(link, shortLink, id string, ctx context.Context) {
	jsonData.ID = id
	jsonData.Link = link
	jsonData.ShortLink = shortLink
	jsonData.Ctx = ctx
}

func (jsonData *MemStorage) Get() (string, error) {

	logger.PrintLog(logger.INFO, "Get from memory")

	var mx sync.Mutex
	mx.Lock()
	defer mx.Unlock()

	storageData := jsonData.Storage.Get()

	if len(storageData) == 0 {
		return "", errors.New("data not found")
	}

	for _, v := range storageData {
		if v.ID == jsonData.ID || v.Link == jsonData.Link {
			return v.Link, nil
		}
	}
	return "", errors.New("data not found")
}

func (jsonData *MemStorage) Set() error {

	logger.PrintLog(logger.INFO, "Set to memory")

	var mx sync.Mutex
	mx.Lock()
	defer mx.Unlock()

	storageData := jsonData.Storage.Get()

	if len(storageData) != 0 {
		for _, v := range storageData {
			if v.Link == jsonData.Link {
				return nil
			}
		}
	}

	var toStore = memoryStorage.StorageItem{
		Link:      jsonData.Link,
		ShortLink: jsonData.ShortLink,
		ID:        jsonData.ID,
	}
	jsonData.Storage.Set(toStore)

	return nil
}

type outputBatch struct {
	CorrelationID string `json:"correlation_id"`
	ShortUrl      string `json:"short_url"`
}

func (jsonData *MemStorage) BatchSet() ([]byte, error) {

	var mx sync.Mutex
	mx.Lock()
	defer mx.Unlock()

	var savingData []MemStorage
	var outputData []outputBatch

	err := json.Unmarshal([]byte(jsonData.Link), &savingData)
	if err != nil {
		return nil, err
	}

	for i, v := range savingData {
		shortLink := shorter.GetShortURL(confModule.Config.Final.ShortURLAddr, v.ID)
		savingData[i].ShortLink = shortLink
		//storage[savingData[i].ID] = savingData[i]
		var toStore = memoryStorage.StorageItem{
			Link:      savingData[i].Link,
			ShortLink: shortLink,
			ID:        savingData[i].ID,
		}
		jsonData.Storage.Set(toStore)
		outputData = append(outputData, outputBatch{ShortUrl: shortLink, CorrelationID: v.ID})
	}

	JSONResp, err := json.Marshal(outputData)
	if err != nil {
		logger.PrintLog(logger.WARN, err.Error())
	}

	return JSONResp, nil
}
