package main

import (
	"encoding/json"
	"github.com/MaximMNsk/go-url-shortener/internal/interface/model"
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

/**
 * Request type handlers
 */

type InputData struct {
	Link      string
	ShortLink string
	ID        string
}

func handleGET(res http.ResponseWriter, req *http.Request) {

	var mx sync.Mutex
	mx.Lock()
	defer mx.Unlock()

	// Пришел ид
	requestID := req.URL.Path[1:]
	linkData := InputData{
		ID:        requestID,
		ShortLink: ``,
		Link:      ``,
	}

	storage := initStorage(&linkData)
	saved, err := storage.Get()
	if err != nil {
		logger.PrintLog(logger.WARN, "Get exception: "+err.Error())
		httpResp.BadRequest(res)
		return
	}

	if saved != "" {
		additional := httpResp.Additional{
			Place:     "header",
			OuterData: "Location",
			InnerData: saved,
		}
		// Если есть, отдаем 307 редирект
		logger.PrintLog(logger.INFO, "Success")
		httpResp.TempRedirect(res, additional)
		return
	}

	// Если нет, отдаем BadRequest
	logger.PrintLog(logger.WARN, "Not success")
	httpResp.BadRequest(res)
}

/**
 * Обработка POST
 */
func handlePOST(res http.ResponseWriter, req *http.Request) {

	var mx sync.Mutex
	mx.Lock()
	defer mx.Unlock()

	contentBody, errBody := io.ReadAll(req.Body)
	defer req.Body.Close()
	if errBody != nil {
		httpResp.BadRequest(res)
		return
	}

	// Пришел урл
	linkID := sha1hash.Create(string(contentBody), 8)

	linkData := InputData{
		ID:        linkID,
		Link:      string(contentBody),
		ShortLink: shorter.GetShortURL(confModule.Config.Final.ShortURLAddr, linkID),
	}

	additional := httpResp.Additional{
		Place:     "body",
		InnerData: linkData.ShortLink,
	}

	storage := initStorage(&linkData)
	err := storage.Set()
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

	var mx sync.Mutex
	mx.Lock()
	defer mx.Unlock()

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

	var reqData = InputData{
		Link: string(contentBody),
	}
	storage := initStorage(&reqData)

	resData, err := storage.BatchSet()
	if err != nil && !strings.Contains(err.Error(), `SQLSTATE 23505`) {
		httpResp.BadRequest(res)
		return
	}

	additional := httpResp.Additional{
		Place:     "body",
		InnerData: string(resData),
	}

	httpResp.CreatedJSON(res, additional)
}

type input struct {
	URL string `json:"url"`
}

type output struct {
	Result string `json:"result"`
}

func handleAPIShorten(res http.ResponseWriter, req *http.Request) {

	contentBody, errBody := io.ReadAll(req.Body)
	defer req.Body.Close()
	if errBody != nil {
		httpResp.BadRequest(res)
		return
	}

	// Пришел урл, парсим его
	var apiData input
	err := json.Unmarshal(contentBody, &apiData)
	if err != nil {
		httpResp.InternalError(res)
		return
	}
	linkID := sha1hash.Create(apiData.URL, 8)
	linkData := InputData{
		Link:      apiData.URL,
		ShortLink: shorter.GetShortURL(confModule.Config.Final.ShortURLAddr, linkID),
		ID:        linkID,
	}

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

	storage := initStorage(&linkData)
	err = storage.Set()
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

	var mx sync.Mutex
	mx.Lock()
	defer mx.Unlock()

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
func initStorage(data *InputData) model.Storable {
	var storage model.Storable
	if confModule.Config.Env.DB != "" || confModule.Config.Flag.DB != "" {
		storage = &database.DBStorage{
			ID:        data.ID,
			Link:      data.Link,
			ShortLink: data.ShortLink,
		}
		return storage
	}
	if confModule.Config.Env.LinkFile != "" || confModule.Config.Flag.LinkFile != "" {
		storage = &files.FileStorage{
			ID:        data.ID,
			Link:      data.Link,
			ShortLink: data.ShortLink,
		}
		return storage
	}
	storage = &memory.MemStorage{
		ID:        data.ID,
		Link:      data.Link,
		ShortLink: data.ShortLink,
	}
	return storage
}

func main() {

	logger.PrintLog(logger.INFO, "Start server")
	logger.PrintLog(logger.INFO, "Handle config")

	err := confModule.HandleConfig()
	if err != nil {
		logger.PrintLog(logger.FATAL, "Can't handle config. "+err.Error())
	}

	if confModule.Config.Final.DB != "" {
		err := db.Connect()
		if err != nil {
			logger.PrintLog(logger.ERROR, "Failed connect to DB")
		}
		database.PrepareDB(db.GetDB())
		defer func() {
			err := db.Close()
			if err != nil {
				logger.PrintLog(logger.ERROR, "Failed close connection to DB")
			}
		}()
	}

	logger.PrintLog(logger.INFO, "Declaring router")
	r := chi.NewRouter().
		With(extlogger.Log).
		With(compress.GzipHandler).
		With(handleOther)
	r.Route("/", func(r chi.Router) {
		r.Post(`/`, handlePOST)
		r.Post(`/api/{query}`, handleAPI)
		r.Post(`/api/shorten/{query}`, handleAPI)
		r.Get(`/ping`, handlePing)
		r.Get(`/{query}`, handleGET)
	})

	logger.PrintLog(logger.INFO, "Starting server")
	err = http.ListenAndServe(confModule.Config.Final.AppAddr, r)
	if err != nil {
		logger.PrintLog(logger.FATAL, "Can't start server. "+err.Error())
	}
}
