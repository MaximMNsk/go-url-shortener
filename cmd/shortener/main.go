package main

import (
	"encoding/json"
	"github.com/MaximMNsk/go-url-shortener/internal/models/database"
	"github.com/MaximMNsk/go-url-shortener/internal/models/files"
	"github.com/MaximMNsk/go-url-shortener/internal/models/memory"
	"github.com/MaximMNsk/go-url-shortener/internal/storage/db"
	"github.com/MaximMNsk/go-url-shortener/internal/util/extlogger"
	"github.com/MaximMNsk/go-url-shortener/internal/util/logger"
	"github.com/MaximMNsk/go-url-shortener/internal/util/rand"
	"github.com/MaximMNsk/go-url-shortener/internal/util/shorter"
	"github.com/MaximMNsk/go-url-shortener/server/compress"
	confModule "github.com/MaximMNsk/go-url-shortener/server/config"
	httpResp "github.com/MaximMNsk/go-url-shortener/server/http"
	"github.com/go-chi/chi/v5"
	"io"
	"net/http"
)

var Storage string

/**
 * Request type handlers
 */

func handlePing(res http.ResponseWriter, req *http.Request) {
	if db.GetDB() != nil {
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

type JSONData struct {
	Link      string
	ShortLink string
	ID        string
}

func load(i JSONData, storage string) (JSONData, error) {
	if storage == "database" {
		linkData := database.JSONData{
			ID:        i.ID,
			Link:      i.Link,
			ShortLink: i.ShortLink,
		}
		err := linkData.Get()
		return JSONData(linkData), err
	}
	if storage == "files" {
		linkData := files.JSONData{
			ID:        i.ID,
			Link:      i.Link,
			ShortLink: i.ShortLink,
		}
		err := linkData.Get()
		return JSONData(linkData), err
	}
	if storage == "memory" {
		linkData := memory.JSONData{
			ID:        i.ID,
			Link:      i.Link,
			ShortLink: i.ShortLink,
		}
		err := linkData.Get()
		return JSONData(linkData), err
	}
	return JSONData{}, nil
}

func store(i JSONData, storage string) error {
	if storage == "database" {
		linkData := database.JSONData{
			ID:        i.ID,
			Link:      i.Link,
			ShortLink: i.ShortLink,
		}
		err := linkData.Set()
		return err
	}
	if storage == "files" {
		linkData := files.JSONData{
			ID:        i.ID,
			Link:      i.Link,
			ShortLink: i.ShortLink,
		}
		err := linkData.Set()
		return err
	}
	if storage == "memory" {
		linkData := memory.JSONData{
			ID:        i.ID,
			Link:      i.Link,
			ShortLink: i.ShortLink,
		}
		err := linkData.Set()
		return err
	}
	return nil
}

func handleGET(res http.ResponseWriter, req *http.Request) {

	// Пришел ид
	requestID := req.URL.Path[1:]
	linkData := JSONData{}
	linkData.ID = requestID
	linkData, err := load(linkData, Storage)
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

	contentBody, errBody := io.ReadAll(req.Body)
	defer req.Body.Close()
	if errBody != nil {
		httpResp.BadRequest(res)
		return
	}

	// Пришел урл
	linkID := rand.RandStringBytes(8)
	linkData := JSONData{}
	linkData.Link = string(contentBody)
	linkData, err := load(linkData, Storage)
	if err != nil {
		logger.PrintLog(logger.WARN, "File exception: "+err.Error())
	}
	shortLink := linkData.ShortLink
	// Если нет, генерим ид, сохраняем
	if linkData.ID == "" {
		linkData.Link = string(contentBody)
		linkData.ShortLink = shorter.GetShortURL(confModule.Config.Final.ShortURLAddr, linkID)
		linkData.ID = linkID
		//err := linkDataSet.Set()
		err := store(linkData, Storage)
		if err != nil {
			logger.PrintLog(logger.ERROR, "Can not set link data: "+err.Error())
			httpResp.InternalError(res)
			return
		}
		shortLink = linkData.ShortLink
	}
	// Отдаем 201 ответ с шортлинком
	shortLinkByte := []byte(shortLink)

	additional := httpResp.Additional{
		Place:     "body",
		InnerData: string(shortLinkByte),
	}
	httpResp.Created(res, additional)
}

type input struct {
	URL string `json:"url"`
}
type output struct {
	Result string `json:"result"`
}

func handleAPI(res http.ResponseWriter, req *http.Request) {

	contentBody, errBody := io.ReadAll(req.Body)
	if errBody != nil {
		httpResp.BadRequest(res)
		return
	}

	// Пришел урл
	linkID := rand.RandStringBytes(8)
	//linkData := storage.JSONData{}
	var apiData input
	err := json.Unmarshal(contentBody, &apiData)
	if err != nil {
		httpResp.InternalError(res)
		return
	}
	linkData := JSONData{}
	linkData.Link = apiData.URL
	linkData, err = load(linkData, Storage)
	if err != nil {
		logger.PrintLog(logger.WARN, "File exception: "+err.Error())
		//httpResp.InternalError(res)
		//return
	}
	shortLink := linkData.ShortLink
	// Если нет, генерим ид, сохраняем
	if linkData.ID == "" {
		linkData.Link = apiData.URL
		linkData.ShortLink = shorter.GetShortURL(confModule.Config.Final.ShortURLAddr, linkID)
		linkData.ID = linkID
		err := store(linkData, Storage)
		if err != nil {
			httpResp.InternalError(res)
			return
		}
		shortLink = linkData.ShortLink
	}
	var resp output
	resp.Result = shortLink
	var JSONResp []byte
	JSONResp, err = json.Marshal(resp)
	if err != nil {
		httpResp.InternalError(res)
		return
	}
	// Отдаем 201 ответ с шортлинком
	additional := httpResp.Additional{
		Place:     "body",
		InnerData: string(JSONResp),
	}
	httpResp.CreatedJSON(res, additional)
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

	//thisPath, _ := filepath.Abs("")
	logger.PrintLog(logger.INFO, "File path: "+confModule.Config.Final.LinkFile)
	//logger.PrintLog(logger.INFO, "This path: "+thisPath)

	logger.PrintLog(logger.INFO, "Declaring router")
	r := chi.NewRouter().
		With(extlogger.Log).
		With(compress.GzipHandler).
		With(handleOther)
	r.Route("/", func(r chi.Router) {
		r.Post(`/`, handlePOST)
		r.Post(`/api/shorten`, handleAPI)
		r.Get(`/ping`, handlePing)
		r.Get(`/{query}`, handleGET)
	})

	logger.PrintLog(logger.INFO, "Starting server")
	err = http.ListenAndServe(confModule.Config.Final.AppAddr, r)
	if err != nil {
		logger.PrintLog(logger.FATAL, "Can't start server. "+err.Error())
	}
}
