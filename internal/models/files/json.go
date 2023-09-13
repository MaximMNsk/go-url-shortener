package files

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

type JSONDataGet struct {
	Link      string
	ShortLink string
	ID        string
}

type JSONDataSet JSONDataGet

func (jsonData *JSONDataGet) Get(fileName string) {
	var savedData []JSONDataGet
	jsonString := getData(fileName)
	_ = json.Unmarshal([]byte(jsonString), &savedData)
	for _, v := range savedData {
		if v.ID == jsonData.ID || v.Link == jsonData.Link {
			jsonData.ID = v.ID
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

func (jsonData JSONDataSet) Set(fileName string) {
	var toSave []JSONDataSet
	var savedData []JSONDataSet
	jsonString := getData(fileName)
	err := json.Unmarshal([]byte(jsonString), &savedData)
	if err != nil {
		fmt.Println(err)
	}
	toSave = append(savedData, jsonData)
	content, _ := json.Marshal(toSave)

	isOk := saveData(content, fileName)
	if !isOk {
		panic("Saving error")
	}
}

func saveData(data []byte, fileName string) bool {
	f, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
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
