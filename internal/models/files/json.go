package files

import (
	"encoding/json"
	"errors"
	"github.com/MaximMNsk/go-url-shortener/internal/util/logger"
	"io"
	"os"
	"path/filepath"
)

type JSONDataGet struct {
	Link      string
	ShortLink string
	ID        string
}

type JSONDataSet JSONDataGet

func (jsonData *JSONDataGet) Get(fileName string) error {
	var savedData []JSONDataSet
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

func (jsonData JSONDataSet) Set(fileName string) error {
	var toSave []JSONDataSet
	var savedData []JSONDataSet
	jsonString, err := getData(fileName)
	if err == nil {
		err = json.Unmarshal([]byte(jsonString), &savedData)
		if err == nil {
			toSave = append(savedData, jsonData)
			var content []byte
			content, err = json.Marshal(toSave)
			if err == nil {
				isOk := saveData(content, fileName)
				if !isOk {
					err = errors.New("cant save")
				}
			}
		}
	}
	return err
}

func saveData(data []byte, fileName string) bool {
	var dir = filepath.Dir(fileName)
	_, err := os.Stat(dir)
	if err != nil {
		logger.PrintLog(logger.WARN, "Cannot get stat directory: "+err.Error())
	}
	if os.IsNotExist(err) {
		err = os.Mkdir(dir, 0644)
		if err != nil {
			logger.PrintLog(logger.ERROR, "Cannot create directory: "+err.Error())
		}
	}

	logger.PrintLog(logger.INFO, "Directory created")

	f, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	defer func(f *os.File) {
		err = f.Close()
	}(f)
	if err != nil {
		logger.PrintLog(logger.ERROR, "Cannot create or open file: "+err.Error())
		return false
	}

	logger.PrintLog(logger.INFO, "File "+fileName+" successfully opened or created")

	_, err = f.Write(data)
	if err != nil {
		return false
	} else {
		return true
	}
}
