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

type ErrorFile struct {
	layer          string
	parentFuncName string
	funcName       string
	message        string
}

func (e *ErrorFile) Error() string {
	return fmt.Sprintf("[%s](%s/%s): %s", e.layer, e.parentFuncName, e.funcName, e.message)
}

const layer = `File`

type FileStorage struct {
	Link        string `json:"original_url"`
	ShortLink   string `json:"short_url"`
	ID          string `json:"correlation_id"`
	DeletedFlag bool   `json:"is_deleted"`
	Ctx         context.Context
}

func (jsonData *FileStorage) Init(link, shortLink, id string, isDeleted bool, ctx context.Context) {
	jsonData.ID = id
	jsonData.Link = link
	jsonData.ShortLink = shortLink
	jsonData.Ctx = ctx
	jsonData.DeletedFlag = isDeleted
}

func (jsonData *FileStorage) Destroy() {
}

func (jsonData *FileStorage) Ping() (bool, error) {
	return true, nil
}

type inputOutputData struct {
	Link        string `json:"original_url"`
	ShortLink   string `json:"short_url"`
	ID          string `json:"correlation_id"`
	DeletedFlag bool   `json:"is_deleted"`
}

func (jsonData *FileStorage) Get() (string, bool, error) {

	fileName := confModule.Config.Final.LinkFile
	var savedData []inputOutputData
	getErr := ErrorFile{
		layer:          layer,
		parentFuncName: `-`,
		funcName:       `Get`,
	}

	jsonString, err := getData(fileName)
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
		if v.ID == jsonData.ID || v.Link == jsonData.Link {
			return v.Link, v.DeletedFlag, nil
		}
	}
	getErr.message = `no data found`
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
	if err != nil {
		getDataErr.message = err.Error()
		return "[]", &getDataErr
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
		err = errors.New(`data absent`)
		getDataErr.message = err.Error()
		return "[]", &getDataErr
	}

	return result, nil
}

func (jsonData *FileStorage) Set() error {

	fileName := confModule.Config.Final.LinkFile

	var toSave []inputOutputData
	var savedData []inputOutputData

	errSet := ErrorFile{
		layer:          layer,
		parentFuncName: `-`,
		funcName:       `Set`,
	}

	preparedData := inputOutputData{
		Link:        jsonData.Link,
		ShortLink:   jsonData.ShortLink,
		ID:          jsonData.ID,
		DeletedFlag: jsonData.DeletedFlag,
	}

	jsonString, err := getData(fileName)
	if err != nil {
		errSet.message = `can't get data`
		return fmt.Errorf(errSet.Error()+`: %w`, err)
	}

	err = json.Unmarshal([]byte(jsonString), &savedData)
	if err != nil {
		errSet.message = `cannot parse json data`
		return fmt.Errorf(errSet.Error()+`: %w`, err)
	}

	toSave = append(savedData, preparedData)
	var content []byte
	content, err = json.Marshal(toSave)
	if err != nil {
		errSet.message = `cannot marshal json data`
		return fmt.Errorf(errSet.Error()+`: %w`, err)
	}

	isOk := saveData(content, fileName)
	if !isOk {
		err = errors.New("cannot save")
		errSet.message = `saving data`
		return fmt.Errorf(errSet.Error()+`: %w`, err)
	}
	return nil
}

func saveData(data []byte, fileName string) bool {

	var mx sync.Mutex
	mx.Lock()
	defer mx.Unlock()

	f, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	defer func(f *os.File) {
		err = f.Close()
	}(f)
	if err != nil {
		return false
	}

	_, err = f.Write(data)
	if err != nil {
		return false
	} else {
		return true
	}
}

func MakeStorageFile(fileName string) error {

	errMakeFile := ErrorFile{
		layer:          layer,
		funcName:       `MakeStorageFile`,
		parentFuncName: `ChooseStorage`,
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

type outputBatch struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
}

func (jsonData *FileStorage) BatchSet() ([]byte, error) {

	var mx sync.Mutex
	mx.Lock()
	defer mx.Unlock()

	errBatchSet := ErrorFile{
		layer:          layer,
		funcName:       `BatchSet`,
		parentFuncName: `-`,
	}

	var savingData []FileStorage
	var outputData []outputBatch

	err := json.Unmarshal([]byte(jsonData.Link), &savingData)
	if err != nil {
		errBatchSet.message = `unmarshal error`
		return nil, fmt.Errorf(errBatchSet.Error()+`: %w`, err)
	}

	for i, v := range savingData {
		shortLink := shorter.GetShortURL(confModule.Config.Final.ShortURLAddr, v.ID)
		savingData[i].ID = v.ID
		savingData[i].ShortLink = shortLink

		outputData = append(outputData, outputBatch{ShortURL: shortLink, CorrelationID: v.ID})
	}

	///////// Current logic
	var savedData []FileStorage

	fileName := confModule.Config.Final.LinkFile
	var jsonString string
	jsonString, err = getData(fileName)
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

	isOk := saveData(content, fileName)
	if !isOk {
		errBatchSet.message = `can't save`
		return []byte(""), &errBatchSet
	}
	//////// End logic

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

func (jsonData *FileStorage) HandleUserUrls() ([]byte, error) {
	var savedData []JSONCutted

	errHandleUserUrls := ErrorFile{
		layer:          layer,
		funcName:       `BatchSet`,
		parentFuncName: `-`,
	}

	fileName := confModule.Config.Final.LinkFile
	jsonString, err := getData(fileName)
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

func (jsonData *FileStorage) HandleUserUrlsDelete() {
}

func (jsonData *FileStorage) AsyncSaver() {
}
