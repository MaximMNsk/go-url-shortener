package memory

import (
	"encoding/json"
	"github.com/MaximMNsk/go-url-shortener/internal/util/logger"
	"github.com/MaximMNsk/go-url-shortener/internal/util/rand"
	"github.com/MaximMNsk/go-url-shortener/internal/util/shorter"
	confModule "github.com/MaximMNsk/go-url-shortener/server/config"
	"sync"
)

type JSONData struct {
	Link          string `json:"original_url"`
	ShortLink     string
	ID            string
	CorrelationID string `json:"correlation_id"`
}

var storage []JSONData

func (jsonData *JSONData) Get() error {
	logger.PrintLog(logger.INFO, "Get from memory")
	for _, v := range storage {
		if v.ID == jsonData.ID || v.Link == jsonData.Link {
			jsonData.ID = v.ID
			jsonData.Link = v.Link
			jsonData.ShortLink = v.ShortLink
			jsonData.CorrelationID = v.CorrelationID
		}
	}
	return nil
}

func (jsonData *JSONData) Set() error {
	logger.PrintLog(logger.INFO, "Set to memory")
	for _, v := range storage {
		if v.Link == jsonData.Link {
			return nil
		}
	}

	storage = append(storage, *jsonData)

	return nil
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
	storage = append(storage, savingData...)
	//////// End logic

	JSONResp, err := json.Marshal(savingData)
	if err != nil {
		logger.PrintLog(logger.WARN, err.Error())
	}

	return JSONResp, nil
}
