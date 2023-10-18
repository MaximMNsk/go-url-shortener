package files

import (
	"encoding/json"
	"errors"
	"github.com/MaximMNsk/go-url-shortener/internal/util/logger"
	"github.com/MaximMNsk/go-url-shortener/internal/util/rand"
	"github.com/MaximMNsk/go-url-shortener/internal/util/shorter"
	confModule "github.com/MaximMNsk/go-url-shortener/server/config"
	"io"
	"os"
	"path/filepath"
	"sync"
)

type JSONData struct {
	Link          string `json:"original_url"`
	ShortLink     string
	ID            string
	CorrelationID string `json:"correlation_id"`
}

func (jsonData *JSONData) Get() error {
	fileName := confModule.Config.Final.LinkFile
	logger.PrintLog(logger.INFO, "Get from file: "+fileName)
	var savedData []JSONData
	jsonString, err := getData(fileName)
	if err == nil {
		err = json.Unmarshal([]byte(jsonString), &savedData)
		if err == nil {
			for _, v := range savedData {
				if v.ID == jsonData.ID || v.Link == jsonData.Link {
					jsonData.ID = v.ID
					jsonData.Link = v.Link
					jsonData.ShortLink = v.ShortLink
				}
			}
		}
	}
	return err
}

func getData(fileName string) (string, error) {
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

func (jsonData *JSONData) Set() error {
	fileName := confModule.Config.Final.LinkFile
	logger.PrintLog(logger.INFO, "Set to file: "+fileName)
	var toSave []JSONData
	var savedData []JSONData
	jsonString, err := getData(fileName)
	if err != nil {
		logger.PrintLog(logger.ERROR, "Setter. Json string: "+jsonString+". Error: "+err.Error())
	}
	//if err == nil {
	err = json.Unmarshal([]byte(jsonString), &savedData)
	if err != nil {
		logger.PrintLog(logger.ERROR, "Setter. Unmarshal json string: "+jsonString+". Error: "+err.Error())
	}
	//if err == nil {
	toSave = append(savedData, *jsonData)
	var content []byte
	content, err = json.Marshal(toSave)
	if err != nil {
		logger.PrintLog(logger.ERROR, "Setter. Marshal new json string: "+jsonString+". Error: "+err.Error())
	}

	//if err == nil {
	isOk := saveData(content, fileName)
	if !isOk {
		err = errors.New("can't save")
		logger.PrintLog(logger.ERROR, "Setter. Save new content: "+string(content)+". Error: "+err.Error())
	}
	return err
}

func saveData(data []byte, fileName string) bool {
	var dir = filepath.Dir(fileName)
	logger.PrintLog(logger.INFO, "Saver. Extracted dir: "+dir)
	_, err := os.Stat(dir)
	if err != nil {
		logger.PrintLog(logger.WARN, "Saver. Cannot get stat directory: "+err.Error())
	}
	if os.IsNotExist(err) {
		logger.PrintLog(logger.INFO, "Saver. Creating dir: "+dir)
		err = os.Mkdir(dir, 0644)
		if err != nil {
			logger.PrintLog(logger.ERROR, "Saver. Cannot create directory: "+err.Error())
		}
	}

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

type BatchStruct struct {
	MX      sync.Mutex
	Content []byte
}

func HandleBatch(batchData *BatchStruct) ([]byte, error) {

	batchData.MX.Lock()
	defer batchData.MX.Unlock()

	var savingData []JSONData
	err := json.Unmarshal(batchData.Content, &savingData)
	if err != nil {
		return []byte(""), err
	}

	for i, _ := range savingData {
		linkID := rand.RandStringBytes(8)
		savingData[i].ID = linkID
		savingData[i].ShortLink = shorter.GetShortURL(confModule.Config.Final.ShortURLAddr, linkID)
	}

	///////// Current logic
	var savedData []JSONData

	fileName := confModule.Config.Final.LinkFile
	jsonString, err := getData(fileName)
	if err != nil {
		jsonString = ""
	} else {
		err = json.Unmarshal([]byte(jsonString), &savedData)
		if err != nil {
			return []byte(""), err
		}
	}

	toSave := append(savedData, savingData...)
	var content []byte
	content, err = json.Marshal(toSave)
	if err != nil {
		return []byte(""), err
	}

	isOk := saveData(content, fileName)
	if !isOk {
		err = errors.New("can't save")
		return []byte(""), err
	}
	//////// End logic

	JSONResp, err := json.Marshal(savingData)
	if err != nil {
		logger.PrintLog(logger.WARN, err.Error())
	}

	return JSONResp, nil
}
