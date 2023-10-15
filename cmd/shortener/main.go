package main

import (
	"encoding/json"
	"github.com/MaximMNsk/go-url-shortener/internal/models/files"
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

/**
 * Request type handlers
 */

func handlePing(res http.ResponseWriter, req *http.Request) {
	err := db.Connect()
	defer db.Close()
	if err != nil {
		logger.PrintLog(logger.ERROR, err.Error())
		httpResp.InternalError(res)
		return
	}
	httpResp.Ok(res)
}

func handleGET(res http.ResponseWriter, req *http.Request) {

	linkFilePath := confModule.Config.Final.LinkFile

	// Пришел ид
	linkData := files.JSONDataGet{}
	requestID := chi.URLParam(req, "query")
	logger.PrintLog(logger.INFO, requestID)
	linkData.ID = requestID
	// Проверяем, есть ли ссылка
	err := linkData.Get(linkFilePath)
	if err != nil {
		logger.PrintLog(logger.WARN, "File exception: "+err.Error())
		//httpResp.InternalError(res)
		//return
	}
	logger.PrintLog(logger.INFO, "Received link: "+linkData.Link)
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

	linkFilePath := confModule.Config.Final.LinkFile

	// Пришел урл
	linkID := rand.RandStringBytes(8)
	linkDataGet := files.JSONDataGet{}
	linkDataGet.Link = string(contentBody)
	// Проверяем, есть ли он (пока без валидаций).
	err := linkDataGet.Get(linkFilePath)
	if err != nil {
		logger.PrintLog(logger.WARN, "File exception: "+err.Error())
		//httpResp.InternalError(res)
		//return
	}
	shortLink := linkDataGet.ShortLink
	// Если нет, генерим ид, сохраняем
	if linkDataGet.ID == "" {
		linkDataSet := files.JSONDataSet{}
		linkDataSet.Link = string(contentBody)
		linkDataSet.ShortLink = shorter.GetShortURL(confModule.Config.Final.ShortURLAddr, linkID)
		linkDataSet.ID = linkID
		err := linkDataSet.Set(linkFilePath)
		if err != nil {
			logger.PrintLog(logger.ERROR, "Can not set link data: "+err.Error())
			httpResp.InternalError(res)
			return
		}
		shortLink = linkDataSet.ShortLink
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

	linkFilePath := confModule.Config.Final.LinkFile

	// Пришел урл
	linkID := rand.RandStringBytes(8)
	linkDataGet := files.JSONDataGet{}
	var linkData input
	err := json.Unmarshal(contentBody, &linkData)
	if err != nil {
		httpResp.InternalError(res)
		return
	}
	linkDataGet.Link = linkData.URL
	err = linkDataGet.Get(linkFilePath)
	if err != nil {
		logger.PrintLog(logger.WARN, "File exception: "+err.Error())
		//httpResp.InternalError(res)
		//return
	}
	shortLink := linkDataGet.ShortLink
	// Если нет, генерим ид, сохраняем
	if linkDataGet.ID == "" {
		linkDataSet := files.JSONDataSet{}
		linkDataSet.Link = linkData.URL
		linkDataSet.ShortLink = shorter.GetShortURL(confModule.Config.Final.ShortURLAddr, linkID)
		linkDataSet.ID = linkID
		err := linkDataSet.Set(linkFilePath)
		if err != nil {
			httpResp.InternalError(res)
			return
		}
		shortLink = linkDataSet.ShortLink
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

	//thisPath, _ := filepath.Abs("")
	logger.PrintLog(logger.INFO, "File path: "+confModule.Config.Final.LinkFile)
	//logger.PrintLog(logger.INFO, "This path: "+thisPath)

	logger.PrintLog(logger.INFO, "Declaring router")
	r := chi.NewRouter().With(extlogger.Log).With(compress.GzipHandler).With(handleOther)
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
