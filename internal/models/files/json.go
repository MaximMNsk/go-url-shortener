package files

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/MaximMNsk/go-url-shortener/internal/util/shorter"
	confModule "github.com/MaximMNsk/go-url-shortener/server/config"
	"io"
	"os"
	"path/filepath"
	"sync"
)

// ErrorFile - определение ошибки слоя файлового хранилища.
type ErrorFile struct {
	layer          string
	parentFuncName string
	funcName       string
	message        string
}

// Error - заменяем стандартный вызов метода своим.
func (e *ErrorFile) Error() string {
	return fmt.Sprintf("[%s](%s/%s): %s", e.layer, e.parentFuncName, e.funcName, e.message)
}

// layer - слой приложения. Используется для логирования.
const layer = `File`

// FileStorage - основная структура хранения.
type FileStorage struct {
	Cfg confModule.OuterConfig
}

// Init - метод создает для каждого запроса объект.
func (fs *FileStorage) Init() error {
	err := MakeStorageFile(fs.Cfg.Final.LinkFile)
	return err
}

// Destroy - метод утилизирует объект для работы с хранилищем.
func (fs *FileStorage) Destroy() {
}

// Ping - метод для проверки работоспособности хранилища.
func (fs *FileStorage) Ping(ctx context.Context) (bool, error) {
	return true, nil
}

type inputOutputData struct {
	Link        string `json:"original_url"`
	ShortLink   string `json:"short_url"`
	ID          string `json:"correlation_id"`
	DeletedFlag bool   `json:"is_deleted"`
}

// Get - возвращает инфо о сохраненном и сокращенном УРЛ.
// Первый возвращаемый параметр - сокращенный УРЛ,
// второй - флаг присутствия,
// третий - ошибка выполнения.
func (fs *FileStorage) Get(ctx context.Context, shortLink string) (string, bool, error) {

	var savedData []inputOutputData
	getErr := ErrorFile{
		layer:          layer,
		parentFuncName: `-`,
		funcName:       `Get`,
	}

	jsonString, err := getData(fs.Cfg.Final.LinkFile)
	if err != nil {
		getErr.message = `get data error`
		return "", false, fmt.Errorf(getErr.Error()+`: %w`, err)
	}
	err = json.Unmarshal([]byte(jsonString), &savedData)
	if err != nil {
		getErr.message = `json parse error`
		return "", false, fmt.Errorf(getErr.Error()+`: %w`, err)
	}
	for _, v := range savedData {
		if v.ID == shortLink || v.Link == shortLink {
			return v.Link, v.DeletedFlag, nil
		}
	}
	getErr.message = `no data found in ` + fs.Cfg.Final.LinkFile
	return "", false, fmt.Errorf(`%w`, &getErr)
}

func getData(fileName string) (string, error) {

	var mx sync.Mutex
	mx.Lock()
	defer mx.Unlock()

	getDataErr := ErrorFile{
		layer:          layer,
		parentFuncName: `Get`,
		funcName:       `getData`,
	}

	var result string
	data := make([]byte, 256)
	f, err := os.OpenFile(fileName, os.O_RDONLY, 0644)
	var osType *os.PathError
	if errors.As(err, &osType) {
		getDataErr.message = err.Error()
		return "[]", nil
	}
	defer f.Close()

	for {
		n, errRead := f.Read(data)
		if errRead == io.EOF { // если конец файла
			break // выходим из цикла
		}
		if errRead != nil {
			getDataErr.message = errRead.Error()
			return "[]", &getDataErr
		}
		result += string(data[:n])
	}

	if result == "" {
		return "[]", nil
	}

	return result, nil
}

// Set - сохраняет и сокращает УРЛ.
// Возвращает статус работы в виде ошибки.
func (fs *FileStorage) Set(ctx context.Context, originalLink string, shortLink string, hashLink string, userID int) error {

	var toSave []inputOutputData
	var toLoad []inputOutputData

	errSet := ErrorFile{
		layer:          layer,
		parentFuncName: `-`,
		funcName:       `Set`,
	}

	preparedData := inputOutputData{
		Link:        originalLink,
		ShortLink:   shortLink,
		ID:          hashLink,
		DeletedFlag: false,
	}

	jsonString, err := getData(fs.Cfg.Final.LinkFile)
	if err != nil {
		errSet.message = `can't get data`
		return fmt.Errorf(errSet.Error()+`: %w`, err)
	}

	err = json.Unmarshal([]byte(jsonString), &toLoad)
	if err != nil {
		errSet.message = `cannot parse json data`
		return fmt.Errorf(errSet.Error()+`: %w`, err)
	}

	toSave = append(toLoad, preparedData)
	var content []byte
	content, err = json.Marshal(toSave)
	if err != nil {
		errSet.message = `cannot marshal json data`
		return fmt.Errorf(errSet.Error()+`: %w`, err)
	}

	saveErr := saveData(content, fs.Cfg.Final.LinkFile)
	if saveErr != nil {
		errSet.message = `saving data in ` + fs.Cfg.Final.LinkFile
		return fmt.Errorf(errSet.Error()+`: %w`, saveErr)
	}
	return nil
}

func saveData(data []byte, fileName string) error {

	var mx sync.Mutex
	mx.Lock()
	defer mx.Unlock()

	errSaveData := ErrorFile{
		layer:          layer,
		funcName:       `saveData`,
		parentFuncName: `Set|BatchSet`,
	}

	f, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	defer func(f *os.File) {
		err = f.Close()
	}(f)
	if err != nil {
		errSaveData.message = `cannot create or open ` + fileName
		return fmt.Errorf(errSaveData.Error()+`: %w`, err)
	}

	_, err = f.Write(data)
	if err != nil {
		errSaveData.message = `cannot write ` + fileName
		return fmt.Errorf(errSaveData.Error()+`: %w`, err)
	}

	return nil
}

// MakeStorageFile - создает файл в файловой системе для хранения данных УРЛ.
func MakeStorageFile(fileName string) error {

	errMakeFile := ErrorFile{
		layer:          layer,
		funcName:       `MakeStorageFile`,
		parentFuncName: `ChooseStorage`,
	}

	var fileExists = true
	if _, err := os.Stat(fileName); errors.Is(err, os.ErrNotExist) {
		fileExists = false
	}

	if fileExists {
		return nil
	}

	var dir = filepath.Dir(fileName)

	_, err := os.Stat(dir)
	if err != nil {
		errMakeFile.message = `cannot get fs info`
		return fmt.Errorf(errMakeFile.Error()+`: %w`, err)
	}
	if os.IsNotExist(err) {
		err = os.Mkdir(dir, 0644)
		if err != nil {
			errMakeFile.message = `cannot create directory: ` + dir
			return fmt.Errorf(errMakeFile.Error()+`: %w`, err)
		}
	}
	_, err = os.Create(fileName)
	if err != nil {
		errMakeFile.message = `cannot create file: ` + fileName
		return fmt.Errorf(errMakeFile.Error()+`: %w`, err)
	}
	return nil
}

type ioBatch struct {
	CorrelationID string `json:"correlation_id"`
	OriginalLink  string `json:"original_url"`
	ShortLink     string
}

// BatchSet - сохраняет и сокращает УРЛ пакетно.
// Возвращает слайс сокращенных УРЛ в байт-формате, а так же результат выполнения.
func (fs *FileStorage) BatchSet(ctx context.Context, data []byte, userID int) ([]byte, error) {

	var mx sync.Mutex
	mx.Lock()
	defer mx.Unlock()

	errBatchSet := ErrorFile{
		layer:          layer,
		funcName:       `BatchSet`,
		parentFuncName: `-`,
	}

	var savingData []ioBatch
	var outputData []ioBatch

	err := json.Unmarshal(data, &savingData)
	if err != nil {
		errBatchSet.message = `unmarshal error`
		return nil, fmt.Errorf(errBatchSet.Error()+`: %w`, err)
	}

	for i, v := range savingData {
		shortLink := shorter.GetShortURL(fs.Cfg.Final.ShortURLAddr, v.CorrelationID)

		savingData[i].CorrelationID = v.CorrelationID
		savingData[i].ShortLink = shortLink

		outputData = append(outputData, ioBatch{ShortLink: shortLink, CorrelationID: v.CorrelationID})
	}

	var savedData []ioBatch

	var jsonString string
	jsonString, err = getData(fs.Cfg.Final.LinkFile)
	if err != nil {
		errBatchSet.message = `get data error`
		return nil, fmt.Errorf(errBatchSet.Error()+`: %w`, err)
	}
	err = json.Unmarshal([]byte(jsonString), &savedData)
	if err != nil {
		errBatchSet.message = `unmarshal error`
		return nil, fmt.Errorf(errBatchSet.Error()+`: %w`, err)
	}

	toSave := append(savedData, savingData...)
	var content []byte
	content, err = json.Marshal(toSave)
	if err != nil {
		errBatchSet.message = `marshal error`
		return nil, fmt.Errorf(errBatchSet.Error()+`: %w`, err)
	}

	saveErr := saveData(content, fs.Cfg.Final.LinkFile)
	if saveErr != nil {
		errBatchSet.message = `can't save`
		return []byte(""), fmt.Errorf(errBatchSet.Error()+`: %w`, saveErr)
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
func (fs *FileStorage) HandleUserUrls(ctx context.Context, userID int) ([]byte, error) {
	var savedData []JSONCut

	errHandleUserUrls := ErrorFile{
		layer:          layer,
		funcName:       `BatchSet`,
		parentFuncName: `-`,
	}

	jsonString, err := getData(fs.Cfg.Final.LinkFile)
	if err != nil {
		errHandleUserUrls.message = `get data error`
		return nil, fmt.Errorf(errHandleUserUrls.Error()+`: %w`, err)
	}

	err = json.Unmarshal([]byte(jsonString), &savedData)
	if err != nil {
		errHandleUserUrls.message = `unmarshal error`
		return nil, fmt.Errorf(errHandleUserUrls.Error()+`: %w`, err)
	}

	if len(savedData) > 0 {
		var content []byte
		content, err = json.Marshal(savedData)
		if err != nil {
			errHandleUserUrls.message = `marshal error`
			return nil, fmt.Errorf(errHandleUserUrls.Error()+`: %w`, err)
		}
		return content, nil
	}
	return nil, nil
}

// HandleUserUrlsDelete - удаляет переданные УРЛ текущего пользователя.
// Отправляет данные в канал, из которого асинхронно вычитываются УРЛ и удаляются.
func (fs *FileStorage) HandleUserUrlsDelete(links string, userID int) {
}

// AsyncSaver - метод-демон, который работает асинхронно.
// Слушает канал, в который передаются УРЛ для удаления и обрабатывает их.
func (fs *FileStorage) AsyncSaver() {
}
