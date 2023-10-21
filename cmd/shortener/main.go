package main

import (
	"encoding/json"
	"github.com/MaximMNsk/go-url-shortener/internal/models/database"
	"github.com/MaximMNsk/go-url-shortener/internal/models/files"
	"github.com/MaximMNsk/go-url-shortener/internal/models/memory"
	"github.com/MaximMNsk/go-url-shortener/internal/storage/db"
	"github.com/MaximMNsk/go-url-shortener/internal/util/extlogger"
	"github.com/MaximMNsk/go-url-shortener/internal/util/hash/sha1hash"
	"github.com/MaximMNsk/go-url-shortener/internal/util/logger"
	"github.com/MaximMNsk/go-url-shortener/internal/util/shorter"
	"github.com/MaximMNsk/go-url-shortener/server/compress"
	confModule "github.com/MaximMNsk/go-url-shortener/server/config"
	httpResp "github.com/MaximMNsk/go-url-shortener/server/http"
	"github.com/go-chi/chi/v5"
	"io"
	"net/http"
	"strings"
	"sync"
)

var Storage string

/**
 * Request type handlers
 */

type JSONData struct {
	Link          string
	ShortLink     string
	ID            string
	CorrelationID string
}

func load(i JSONData, storage string, mu *sync.Mutex) (JSONData, error) {

	mu.Lock()
	defer mu.Unlock()

	if storage == "database" {
		linkData := database.JSONData{
			ID:            i.ID,
			Link:          i.Link,
			ShortLink:     i.ShortLink,
			CorrelationID: i.CorrelationID,
		}
		err := linkData.Get()
		return JSONData(linkData), err
	}
	if storage == "files" {
		linkData := files.JSONData{
			ID:            i.ID,
			Link:          i.Link,
			ShortLink:     i.ShortLink,
			CorrelationID: i.CorrelationID,
		}
		err := linkData.Get()
		return JSONData(linkData), err
	}
	if storage == "memory" {
		linkData := memory.JSONData{
			ID:            i.ID,
			Link:          i.Link,
			ShortLink:     i.ShortLink,
			CorrelationID: i.CorrelationID,
		}
		err := linkData.Get()
		return JSONData(linkData), err
	}
	return JSONData{}, nil
}

func store(i JSONData, storage string, mu *sync.Mutex) error {

	mu.Lock()
	defer mu.Unlock()

	if storage == "database" {
		linkData := database.JSONData{
			ID:            i.ID,
			Link:          i.Link,
			ShortLink:     i.ShortLink,
			CorrelationID: i.CorrelationID,
		}
		err := linkData.Set()
		return err
	}
	if storage == "files" {
		linkData := files.JSONData{
			ID:            i.ID,
			Link:          i.Link,
			ShortLink:     i.ShortLink,
			CorrelationID: i.CorrelationID,
		}
		err := linkData.Set()
		return err
	}
	if storage == "memory" {
		linkData := memory.JSONData{
			ID:            i.ID,
			Link:          i.Link,
			ShortLink:     i.ShortLink,
			CorrelationID: i.CorrelationID,
		}
		err := linkData.Set()
		return err
	}
	return nil
}

func handleGET(res http.ResponseWriter, req *http.Request) {

	var mx sync.Mutex

	// Пришел ид
	requestID := req.URL.Path[1:]
	linkData := JSONData{}
	linkData.ID = requestID
	linkData.CorrelationID = requestID
	linkData, err := load(linkData, Storage, &mx)
	if err != nil {
		logger.PrintLog(logger.WARN, "File exception: "+err.Error())
	}
	logger.PrintLog(logger.INFO, "Received link: "+linkData.Link)
	// Проверяем, есть ли ссылка
	if linkData.Link != "" {
		additional := httpResp.Additional{
			Place:     "header",
			OuterData: "Location",
			InnerData: linkData.Link,
		}
		// Если есть, отдаем 307 редирект
		logger.PrintLog(logger.INFO, "Success")
		httpResp.TempRedirect(res, additional)
		return
	} else {
		// Если нет, отдаем BadRequest
		logger.PrintLog(logger.WARN, "Not success")
		httpResp.BadRequest(res)
		return
	}
}

/**
 * Обработка POST
 */
func handlePOST(res http.ResponseWriter, req *http.Request) {

	var mx sync.Mutex

	contentBody, errBody := io.ReadAll(req.Body)
	defer req.Body.Close()
	if errBody != nil {
		httpResp.BadRequest(res)
		return
	}

	// Пришел урл
	linkData := JSONData{}
	linkData.Link = string(contentBody)
	linkID := sha1hash.Create(linkData.Link, 8)

	linkData.Link = string(contentBody)
	linkData.ShortLink = shorter.GetShortURL(confModule.Config.Final.ShortURLAddr, linkID)
	linkData.ID = linkID

	additional := httpResp.Additional{
		Place:     "body",
		InnerData: linkData.ShortLink,
	}

	err := store(linkData, Storage, &mx)
	if err != nil {
		logger.PrintLog(logger.ERROR, "Can not set link data: "+err.Error())
		if strings.Contains(err.Error(), `SQLSTATE 23505`) {
			httpResp.Conflict(res, additional)
			return
		}
		httpResp.InternalError(res)
		return
	}

	// Отдаем 201 ответ с шортлинком
	httpResp.Created(res, additional)
}

type controllers map[string]bool

func handleAPI(res http.ResponseWriter, req *http.Request) {

	ctrl := chi.URLParam(req, "query")

	availableCurls := make(controllers)
	availableCurls["shorten"] = true
	availableCurls["batch"] = true

	if !availableCurls[ctrl] {
		httpResp.BadRequest(res)
		return
	}

	if ctrl == "shorten" {
		handleAPIShorten(res, req)
		return
	}

	if ctrl == "batch" {
		handleAPIBatch(res, req)
		return
	}
}

func handleAPIBatch(res http.ResponseWriter, req *http.Request) {

	contentBody, errBody := io.ReadAll(req.Body)
	defer req.Body.Close()
	if errBody != nil {
		httpResp.BadRequest(res)
		return
	}

	if Storage == "database" {
		batchData := database.BatchStruct{
			Content: contentBody,
		}
		batchResp, err := database.HandleBatch(&batchData)
		if err != nil {
			if !strings.Contains(err.Error(), `SQLSTATE 23505`) {
				logger.PrintLog(logger.ERROR, err.Error())
				httpResp.InternalError(res)
				return
			}
		}
		additional := httpResp.Additional{
			Place:     "body",
			InnerData: string(batchResp),
		}
		httpResp.CreatedJSON(res, additional)
	}

	if Storage == "files" {
		batchData := files.BatchStruct{
			Content: contentBody,
		}
		batchResp, err := files.HandleBatch(&batchData)
		if err != nil {
			logger.PrintLog(logger.ERROR, err.Error())
			httpResp.InternalError(res)
			return
		}
		additional := httpResp.Additional{
			Place:     "body",
			InnerData: string(batchResp),
		}
		httpResp.CreatedJSON(res, additional)
	}

	if Storage == "memory" {
		batchData := memory.BatchStruct{
			Content: contentBody,
		}
		batchResp, err := memory.HandleBatch(&batchData)
		if err != nil {
			logger.PrintLog(logger.ERROR, err.Error())
			httpResp.InternalError(res)
			return
		}
		additional := httpResp.Additional{
			Place:     "body",
			InnerData: string(batchResp),
		}
		httpResp.CreatedJSON(res, additional)
	}
}

type input struct {
	URL string `json:"url"`
}
type output struct {
	Result string `json:"result"`
}

func handleAPIShorten(res http.ResponseWriter, req *http.Request) {

	var mx sync.Mutex

	contentBody, errBody := io.ReadAll(req.Body)
	defer req.Body.Close()
	if errBody != nil {
		httpResp.BadRequest(res)
		return
	}

	// Пришел урл
	var apiData input
	err := json.Unmarshal(contentBody, &apiData)
	if err != nil {
		httpResp.InternalError(res)
		return
	}
	linkData := JSONData{}
	linkID := sha1hash.Create(linkData.Link, 8)
	linkData.Link = apiData.URL
	linkData.ShortLink = shorter.GetShortURL(confModule.Config.Final.ShortURLAddr, linkID)
	linkData.ID = linkID

	var resp output
	resp.Result = linkData.ShortLink
	var JSONResp []byte
	JSONResp, err = json.Marshal(resp)
	if err != nil {
		httpResp.InternalError(res)
		return
	}
	additional := httpResp.Additional{
		Place:     "body",
		InnerData: string(JSONResp),
	}

	err = store(linkData, Storage, &mx)
	if err != nil {
		if strings.Contains(err.Error(), `SQLSTATE 23505`) {
			httpResp.ConflictJSON(res, additional)
			return
		}
		httpResp.InternalError(res)
		return
	}

	// Отдаем 201 ответ с шортлинком
	httpResp.CreatedJSON(res, additional)
}

func handlePing(res http.ResponseWriter, req *http.Request) {
	if db.GetDB() == nil {
		err := db.Connect()
		defer func() {
			_ = db.Close()
		}()
		if err != nil {
			logger.PrintLog(logger.ERROR, err.Error())
			httpResp.InternalError(res)
			return
		}
	}
	httpResp.Ok(res)
}

func handleOther(next http.Handler) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		if req.Method == "GET" || req.Method == "POST" {
			next.ServeHTTP(res, req)
		} else {
			httpResp.BadRequest(res)
		}
	})
}

/**
 * Executor
 */

func main() {

	logger.PrintLog(logger.INFO, "Start server")

	logger.PrintLog(logger.INFO, "Handle config")
	err := confModule.HandleConfig()
	if err != nil {
		logger.PrintLog(logger.FATAL, "Can't handle config. "+err.Error())
	}

	Storage = "memory"
	if confModule.Config.Env.DB != "" || confModule.Config.Flag.DB != "" {
		Storage = "database"
	}
	if confModule.Config.Env.LinkFile != "" || confModule.Config.Flag.LinkFile != "" {
		Storage = "files"
	}

	logger.PrintLog(logger.INFO, "Storage: "+Storage)

	if Storage == "database" {
		_ = db.Connect()
		defer func() {
			_ = db.Close()
		}()
		database.PrepareDB(db.GetDB())
	}

	logger.PrintLog(logger.INFO, "Declaring router")
	r := chi.NewRouter().
		With(extlogger.Log).
		With(compress.GzipHandler).
		With(handleOther)
	r.Route("/", func(r chi.Router) {
		r.Post(`/`, handlePOST)
		r.Post(`/api/shorten/{query}`, handleAPI)
		r.Post(`/api/{query}`, handleAPI)
		r.Get(`/ping`, handlePing)
		r.Get(`/{query}`, handleGET)
	})

	logger.PrintLog(logger.INFO, "Starting server")
	err = http.ListenAndServe(confModule.Config.Final.AppAddr, r)
	if err != nil {
		logger.PrintLog(logger.FATAL, "Can't start server. "+err.Error())
	}
}
