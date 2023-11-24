package memory

import (
	"context"
	"encoding/json"
	"errors"
	memoryStorage "github.com/MaximMNsk/go-url-shortener/internal/storage/memory"
	"github.com/MaximMNsk/go-url-shortener/internal/util/logger"
	"github.com/MaximMNsk/go-url-shortener/internal/util/shorter"
	confModule "github.com/MaximMNsk/go-url-shortener/server/config"
	"sync"
)

type MemStorage struct {
	Link        string `json:"original_url"`
	ShortLink   string `json:"short_url"`
	ID          string `json:"correlation_id"`
	DeletedFlag bool   `json:"is_deleted"`
	Ctx         context.Context
	Storage     memoryStorage.Storage
}

func (jsonData *MemStorage) Init(link, shortLink, id string, isDeleted bool, ctx context.Context) {
	jsonData.ID = id
	jsonData.Link = link
	jsonData.ShortLink = shortLink
	jsonData.Ctx = ctx
	jsonData.DeletedFlag = isDeleted
}

func (jsonData *MemStorage) Get() (string, bool, error) {

	logger.PrintLog(logger.INFO, "Get from memory")

	var mx sync.Mutex
	mx.Lock()
	defer mx.Unlock()

	storageData := jsonData.Storage.Get()

	if len(storageData) == 0 {
		return "", false, errors.New("data not found")
	}

	for _, v := range storageData {
		if v.ID == jsonData.ID || v.Link == jsonData.Link {
			return v.Link, v.DeletedFlag, nil
		}
	}
	return "", false, errors.New("data not found")
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
		Link:        jsonData.Link,
		ShortLink:   jsonData.ShortLink,
		ID:          jsonData.ID,
		DeletedFlag: jsonData.DeletedFlag,
	}
	jsonData.Storage.Set(toStore)

	return nil
}

type outputBatch struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
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
			Link:        savingData[i].Link,
			ShortLink:   shortLink,
			ID:          savingData[i].ID,
			DeletedFlag: savingData[i].DeletedFlag,
		}
		jsonData.Storage.Set(toStore)
		outputData = append(outputData, outputBatch{ShortURL: shortLink, CorrelationID: v.ID})
	}

	JSONResp, err := json.Marshal(outputData)
	if err != nil {
		logger.PrintLog(logger.WARN, err.Error())
	}

	return JSONResp, nil
}

type JSONCutted struct {
	Link      string `json:"original_url"`
	ShortLink string `json:"short_url"`
}

func (jsonData *MemStorage) HandleUserUrls() ([]byte, error) {
	storage := jsonData.Storage.Get()
	if len(storage) > 0 {
		var resp JSONCutted
		var batchResp []JSONCutted
		for _, v := range storage {
			resp.Link = v.Link
			resp.ShortLink = v.ShortLink
			batchResp = append(batchResp, resp)
		}
		JSONResp, err := json.Marshal(batchResp)
		return JSONResp, err
	}
	return nil, nil
}

func (jsonData *MemStorage) HandleUserUrlsDelete() {
}
func (jsonData *MemStorage) AsyncSaver() {
}
