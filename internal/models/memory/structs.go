package memory

import (
	"context"
	"encoding/json"
	"fmt"
	memoryStorage "github.com/MaximMNsk/go-url-shortener/internal/storage/memory"
	"github.com/MaximMNsk/go-url-shortener/internal/util/shorter"
	confModule "github.com/MaximMNsk/go-url-shortener/server/config"
	"sync"
)

type ErrorMemory struct {
	layer          string
	parentFuncName string
	funcName       string
	message        string
}

func (e *ErrorMemory) Error() string {
	return fmt.Sprintf("[%s](%s/%s): %s", e.layer, e.parentFuncName, e.funcName, e.message)
}

const layer = `Memory`

type MemStorage struct {
	Link        string `json:"original_url"`
	ShortLink   string `json:"short_url"`
	ID          string `json:"correlation_id"`
	DeletedFlag bool   `json:"is_deleted"`
	Ctx         context.Context
	Storage     memoryStorage.Storage
	Cfg         confModule.OuterConfig
}

func (jsonData *MemStorage) Init(link, shortLink, id string, isDeleted bool, ctx context.Context, cfg confModule.OuterConfig) {
	jsonData.ID = id
	jsonData.Link = link
	jsonData.ShortLink = shortLink
	jsonData.Ctx = ctx
	jsonData.DeletedFlag = isDeleted
	jsonData.Cfg = cfg
}

func (jsonData *MemStorage) Destroy() {
}

func (jsonData *MemStorage) Ping() (bool, error) {
	return true, nil
}

func (jsonData *MemStorage) Get() (string, bool, error) {

	var mx sync.Mutex
	mx.Lock()
	defer mx.Unlock()

	storageData := jsonData.Storage.Get()

	errGet := ErrorMemory{
		layer:          layer,
		funcName:       `Get`,
		parentFuncName: `-`,
	}

	if len(storageData) == 0 {
		errGet.message = "data not found"
		return "", false, &errGet
	}

	for _, v := range storageData {
		if v.ID == jsonData.ID || v.Link == jsonData.Link {
			return v.Link, v.DeletedFlag, nil
		}
	}
	errGet.message = "data not found"
	return "", false, &errGet
}

func (jsonData *MemStorage) Set() error {

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

	errBatchSet := ErrorMemory{
		layer:          layer,
		funcName:       `BatchSet`,
		parentFuncName: `-`,
	}

	var savingData []MemStorage
	var outputData []outputBatch

	err := json.Unmarshal([]byte(jsonData.Link), &savingData)
	if err != nil {
		errBatchSet.message = `unmarshal error`
		return nil, fmt.Errorf(errBatchSet.Error()+`: %w`, err)
	}

	for i, v := range savingData {
		shortLink := shorter.GetShortURL(jsonData.Cfg.Final.ShortURLAddr, v.ID)
		savingData[i].ShortLink = shortLink
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
		errBatchSet.message = `marshal error`
		return nil, fmt.Errorf(errBatchSet.Error()+`: %w`, err)
	}

	return JSONResp, nil
}

type JSONCutted struct {
	Link      string `json:"original_url"`
	ShortLink string `json:"short_url"`
}

func (jsonData *MemStorage) HandleUserUrls() ([]byte, error) {

	errHandleUserUrls := ErrorMemory{
		layer:          layer,
		funcName:       `HandleUserUrls`,
		parentFuncName: `-`,
	}

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
		if err != nil {
			errHandleUserUrls.message = `marshal error`
			return nil, fmt.Errorf(errHandleUserUrls.Error()+`: %w`, err)
		}
		return JSONResp, nil
	}
	return nil, nil
}

func (jsonData *MemStorage) HandleUserUrlsDelete() {
}
func (jsonData *MemStorage) AsyncSaver() {
}
