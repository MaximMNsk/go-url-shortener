package files

import (
	"encoding/json"
	"io"
	"os"
)

type JsonData struct {
	Link      string
	ShortLink string
	Id        string
}

func (jsonData *JsonData) Get(fileName string) {
	var savedData []JsonData
	jsonString := getData(fileName)
	_ = json.Unmarshal([]byte(jsonString), &savedData)
	for _, v := range savedData {
		if v.Id == jsonData.Id || v.Link == jsonData.Link {
			jsonData.Id = v.Id
			jsonData.Link = v.Link
			jsonData.ShortLink = v.ShortLink
		}
	}
}

func getData(fileName string) string {
	var result string
	data := make([]byte, 256)
	f, _ := os.OpenFile(fileName, os.O_RDONLY, 0644)
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			panic(err)
		}
	}(f)

	for {
		n, err := f.Read(data)
		if err == io.EOF { // если конец файла
			break // выходим из цикла
		}
		result += string(data[:n])
	}

	return result
}

func (jsonData JsonData) Set(fileName string) {
	var toSave []JsonData
	var savedData []JsonData
	jsonString := getData(fileName)
	_ = json.Unmarshal([]byte(jsonString), &savedData)
	toSave = append(savedData, jsonData)
	content, _ := json.Marshal(toSave)

	isOk := saveData(content, fileName)
	if !isOk {
		panic("Saving error")
	}
}

func saveData(data []byte, fileName string) bool {
	f, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return false
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			panic(err)
		}
	}(f)

	_, err = f.Write(data)
	if err != nil {
		return false
	} else {
		return true
	}
}
