package files

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
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
	defer func(f *os.File) {
		err = f.Close()
	}(f)
	if err == nil {
		for {
			n, errRead := f.Read(data)
			if errRead == io.EOF { // если конец файла
				break // выходим из цикла
			}
			result += string(data[:n])
		}
	}
	if strings.Contains(err.Error(), "The system cannot find the path specified") {
		return "[]", nil
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
	//fmt.Println(dir)

	err := os.Mkdir(dir, 0777)
	if err != nil {
		return false
	}

	f, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return false
	}
	defer func(f *os.File) {
		err = f.Close()
	}(f)

	_, err = f.Write(data)
	if err != nil {
		return false
	} else {
		return true
	}
}
