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

// ErrorMemory - определение ошибки слоя хранилища в памяти.
type ErrorMemory struct {
	layer          string
	parentFuncName string
	funcName       string
	message        string
}

// Error - заменяем стандартный вызов метода своим.
func (e *ErrorMemory) Error() string {
	return fmt.Sprintf("[%s](%s/%s): %s", e.layer, e.parentFuncName, e.funcName, e.message)
}

const layer = `Memory`

// MemStorage - основная структура хранения.
type MemStorage struct {
	Storage memoryStorage.Storage
	Cfg     confModule.OuterConfig
}

// Init - метод создает для каждого запроса объект.
func (ms *MemStorage) Init() error {
	return nil
}

// Destroy - метод утилизирует объект для работы с хранилищем.
func (ms *MemStorage) Destroy() {
	ms.Storage.Clear()
}

// Ping - метод для проверки работоспособности хранилища.
func (ms *MemStorage) Ping(ctx context.Context) (bool, error) {
	return true, nil
}

// Get - возвращает инфо о сохраненном и сокращенном УРЛ.
// Первый возвращаемый параметр - сокращенный УРЛ,
// второй - флаг присутствия,
// третий - ошибка выполнения.
func (ms *MemStorage) Get(ctx context.Context, shortLink string) (string, bool, error) {

	var mx sync.Mutex
	mx.Lock()
	defer mx.Unlock()

	storageData := ms.Storage.Get()

	errGet := ErrorMemory{
		layer:          layer,
		funcName:       `Get`,
		parentFuncName: `-`,
	}

	if len(storageData) == 0 {
		errGet.message = "data not found"
		return ``, false, &errGet
	}

	for _, v := range storageData {
		if v.ID == shortLink || v.Link == shortLink {
			return v.Link, v.DeletedFlag, nil
		}
	}
	errGet.message = "data not found"
	return ``, false, &errGet
}

// Set - сохраняет и сокращает УРЛ.
// Возвращает статус работы в виде ошибки.
func (ms *MemStorage) Set(ctx context.Context, originalLink string, shortLink string, hashLink string, userID int) error {

	var mx sync.Mutex
	mx.Lock()
	defer mx.Unlock()

	storageData := ms.Storage.Get()

	if len(storageData) != 0 {
		for _, v := range storageData {
			if v.Link == originalLink {
				return nil
			}
		}
	}

	var toStore = memoryStorage.StorageItem{
		Link:        originalLink,
		ShortLink:   shortLink,
		ID:          hashLink,
		DeletedFlag: false,
	}
	ms.Storage.Set(toStore)

	return nil
}

type outputBatch struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
}

// BatchSet - сохраняет и сокращает УРЛ пакетно.
// Возвращает слайс сокращенных УРЛ в байт-формате, а так же результат выполнения.
func (ms *MemStorage) BatchSet(ctx context.Context, data []byte, userID int) ([]byte, error) {

	var mx sync.Mutex
	mx.Lock()
	defer mx.Unlock()

	errBatchSet := ErrorMemory{
		layer:          layer,
		funcName:       `BatchSet`,
		parentFuncName: `-`,
	}

	var savingData []memoryStorage.StorageItem
	var outputData []outputBatch

	err := json.Unmarshal(data, &savingData)
	if err != nil {
		errBatchSet.message = `unmarshal error`
		return nil, fmt.Errorf(errBatchSet.Error()+`: %w`, err)
	}

	for i, v := range savingData {
		shortLink := shorter.GetShortURL(ms.Cfg.Final.ShortURLAddr, v.ID)
		savingData[i].ShortLink = shortLink
		var toStore = memoryStorage.StorageItem{
			Link:        savingData[i].Link,
			ShortLink:   shortLink,
			ID:          savingData[i].ID,
			DeletedFlag: false,
		}
		ms.Storage.Set(toStore)
		outputData = append(outputData, outputBatch{ShortURL: shortLink, CorrelationID: v.ID})
	}

	JSONResp, err := json.Marshal(outputData)
	if err != nil {
		errBatchSet.message = `marshal error`
		return nil, fmt.Errorf(errBatchSet.Error()+`: %w`, err)
	}

	return JSONResp, nil
}

// JSONCut - структура хранения входных/выходных данных для каждого УРЛ в пачке.
type JSONCut struct {
	Link      string `json:"original_url"`
	ShortLink string `json:"short_url"`
}

// HandleUserUrls - возвращает слайс УРЛ, сохраненных текущим пользователем.
// Так же возвращает результат обработки запроса.
func (ms *MemStorage) HandleUserUrls(ctx context.Context, userID int) ([]byte, error) {

	errHandleUserUrls := ErrorMemory{
		layer:          layer,
		funcName:       `HandleUserUrls`,
		parentFuncName: `-`,
	}

	storage := ms.Storage.Get()
	if len(storage) > 0 {
		var resp JSONCut
		var batchResp []JSONCut
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

// HandleUserUrlsDelete - удаляет переданные УРЛ текущего пользователя.
// Отправляет данные в канал, из которого асинхронно вычитываются УРЛ и удаляются.
func (ms *MemStorage) HandleUserUrlsDelete(links string, userID int) {
}

// AsyncSaver - метод-демон, который работает асинхронно.
// Слушает канал, в который передаются УРЛ для удаления и обрабатывает их.
func (ms *MemStorage) AsyncSaver() {
}
