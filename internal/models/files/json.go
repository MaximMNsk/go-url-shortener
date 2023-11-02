package files

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/MaximMNsk/go-url-shortener/internal/util/logger"
	"github.com/MaximMNsk/go-url-shortener/internal/util/shorter"
	confModule "github.com/MaximMNsk/go-url-shortener/server/config"
	"io"
	"os"
	"path/filepath"
	"sync"
)

type FileStorage struct {
	Link      string `json:"original_url"`
	ShortLink string `json:"short_url"`
	ID        string `json:"correlation_id"`
	Ctx       context.Context
}

func (jsonData *FileStorage) Init(link, shortLink, id string, ctx context.Context) {
	jsonData.ID = id
	jsonData.Link = link
	jsonData.ShortLink = shortLink
	jsonData.Ctx = ctx
}

type inputOutputData struct {
	Link      string `json:"original_url"`
	ShortLink string `json:"short_url"`
	ID        string `json:"correlation_id"`
}

func (jsonData *FileStorage) Get() (string, error) {

	fileName := confModule.Config.Final.LinkFile
	logger.PrintLog(logger.INFO, "Get from file: "+fileName)
	var savedData []inputOutputData
	jsonString, err := getData(fileName)
	if err != nil {
		return "", err
	}
	err = json.Unmarshal([]byte(jsonString), &savedData)
	if err != nil {
		return "", err
	}
	for _, v := range savedData {
		if v.ID == jsonData.ID || v.Link == jsonData.Link {
			return v.Link, nil
		}
	}
	return "", errors.New("no data found")
}

func getData(fileName string) (string, error) {

	var mx sync.Mutex
	mx.Lock()
	defer mx.Unlock()

	var result string
	data := make([]byte, 256)
	f, err := os.OpenFile(fileName, os.O_RDONLY, 0644)
	if err != nil {
		return "[]", err
	}
	defer func(f *os.File) {
		err = f.Close()
		if err != nil {
			logger.PrintLog(logger.ERROR, "Close file error: "+err.Error())
		}
	}(f)

	for {
		n, errRead := f.Read(data)
		if errRead == io.EOF { // если конец файла
			break // выходим из цикла
		}
		if errRead != nil {
			return "[]", errRead
		}
		result += string(data[:n])
	}

	if result == "" {
		result = "[]"
	}

	return result, err
}

func (jsonData *FileStorage) Set() error {

	fileName := confModule.Config.Final.LinkFile
	logger.PrintLog(logger.INFO, "Set to file: "+fileName)

	var toSave []inputOutputData
	var savedData []inputOutputData

	preparedData := inputOutputData{
		Link:      jsonData.Link,
		ShortLink: jsonData.ShortLink,
		ID:        jsonData.ID,
	}

	jsonString, err := getData(fileName)
	if err != nil {
		logger.PrintLog(logger.ERROR, "Setter. Json string: "+jsonString+". Error: "+err.Error())
	}

	err = json.Unmarshal([]byte(jsonString), &savedData)
	if err != nil {
		logger.PrintLog(logger.ERROR, "Setter. Unmarshal json string: "+jsonString+". Error: "+err.Error())
	}

	toSave = append(savedData, preparedData)
	var content []byte
	content, err = json.Marshal(toSave)
	if err != nil {
		logger.PrintLog(logger.ERROR, "Setter. Marshal new json string: "+jsonString+". Error: "+err.Error())
	}

	isOk := saveData(content, fileName)
	if !isOk {
		err = errors.New("can't save")
		logger.PrintLog(logger.ERROR, "Setter. Save new content: "+string(content)+". Error: "+err.Error())
	}
	return err
}

func saveData(data []byte, fileName string) bool {

	var mx sync.Mutex
	mx.Lock()
	defer mx.Unlock()

	logger.PrintLog(logger.INFO, "Saver. Directory created")

	f, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	defer func(f *os.File) {
		err = f.Close()
	}(f)
	if err != nil {
		logger.PrintLog(logger.ERROR, "Saver. Cannot create or open file: "+err.Error())
		return false
	}

	logger.PrintLog(logger.INFO, "Saver. File "+fileName+" successfully opened or created")

	_, err = f.Write(data)
	if err != nil {
		return false
	} else {
		return true
	}
}

func MakeStorageFile(fileName string) error {
	var dir = filepath.Dir(fileName)
	logger.PrintLog(logger.INFO, "Saver. Extracted dir: "+dir)
	_, err := os.Stat(dir)
	if err != nil {
		logger.PrintLog(logger.WARN, "Saver. Cannot get stat directory: "+err.Error())
		return err
	}
	if os.IsNotExist(err) {
		logger.PrintLog(logger.INFO, "Saver. Creating dir: "+dir)
		err = os.Mkdir(dir, 0644)
		if err != nil {
			logger.PrintLog(logger.ERROR, "Saver. Cannot create directory: "+err.Error())
			return err
		}
	}
	_, err = os.Create(fileName)
	if err != nil {
		logger.PrintLog(logger.ERROR, "Saver. Cannot create file: "+err.Error())
		return err
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

	var savingData []FileStorage
	var outputData []outputBatch

	err := json.Unmarshal([]byte(jsonData.Link), &savingData)
	if err != nil {
		return nil, err
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
	jsonString, err := getData(fileName)
	if err != nil {
		jsonString = ""
	} else {
		err = json.Unmarshal([]byte(jsonString), &savedData)
		if err != nil {
			return nil, err
		}
	}

	toSave := append(savedData, savingData...)
	var content []byte
	content, err = json.Marshal(toSave)
	if err != nil {
		return nil, err
	}

	isOk := saveData(content, fileName)
	if !isOk {
		err = errors.New("can't save")
		return []byte(""), err
	}
	//////// End logic

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

func (jsonData *FileStorage) HandleUserUrls() ([]byte, error) {
	var savedData []JSONCutted

	fileName := confModule.Config.Final.LinkFile
	jsonString, err := getData(fileName)
	if err != nil {
		jsonString = ""
	} else {
		err = json.Unmarshal([]byte(jsonString), &savedData)
		if err != nil {
			return nil, err
		}
	}

	if len(savedData) > 0 {
		var content []byte
		content, err = json.Marshal(savedData)
		if err != nil {
			return nil, err
		}
		return content, nil
	}
	return nil, nil
}
