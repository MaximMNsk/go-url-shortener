package main

import (
	"encoding/json"
	"fmt"
	"github.com/MaximMNsk/go-url-shortener/internal/models/files"
	"github.com/MaximMNsk/go-url-shortener/internal/util/extlogger"
	"github.com/MaximMNsk/go-url-shortener/internal/util/logger"
	"github.com/MaximMNsk/go-url-shortener/internal/util/pathhandler"
	"github.com/MaximMNsk/go-url-shortener/internal/util/rand"
	"github.com/MaximMNsk/go-url-shortener/server/compress"
	confModule "github.com/MaximMNsk/go-url-shortener/server/config"
	httpResp "github.com/MaximMNsk/go-url-shortener/server/http"
	"github.com/go-chi/chi/v5"
	"io"
	"net/http"
	"path/filepath"
	"strings"
)

/**
 * Request type handlers
 */

func getShortURL(hostPort, linkID string) string {
	return fmt.Sprintf("%s/%s", hostPort, linkID)
}

func handleGET(res http.ResponseWriter, req *http.Request) {
	_ = req.Header.Get("Content-Type")
	rootPath, err := pathhandler.ProjectRoot()
	if err != nil {
		httpResp.InternalError(res)
		return
	}
	linkFilePath := filepath.Join(rootPath, confModule.LinkFile)
	//if strings.Contains(contentType, "text/plain") {
	// Пришел ид
	linkData := files.JSONDataGet{}
	requestID := req.URL.Path[1:]
	linkData.ID = requestID
	// Проверяем, есть ли он.
	err = linkData.Get(linkFilePath)
	if err != nil {
		httpResp.InternalError(res)
		return
	}
	if linkData.Link != "" {
		additional := confModule.Additional{
			Place:     "header",
			OuterData: "Location",
			InnerData: linkData.Link,
		}
		// Если есть, отдаем 307 редирект
		httpResp.TempRedirect(res, additional)
	} else {
		// Если нет, отдаем BadRequest
		httpResp.BadRequest(res)
	}
}

func handlePOST(res http.ResponseWriter, req *http.Request) {
	currentPath := req.URL.Path
	//contentType := req.Header.Get("Content-Type")
	contentBody, errBody := io.ReadAll(req.Body)
	if errBody != nil {
		httpResp.BadRequest(res)
		return
	}
	rootPath, err := pathhandler.ProjectRoot()
	if err != nil {
		httpResp.InternalError(res)
		return
	}
	linkFilePath := filepath.Join(rootPath, confModule.LinkFile)
	if currentPath == "/" /*&& strings.Contains(contentType, "text/plain")*/ {
		// Пришел урл
		linkID := rand.RandStringBytes(8)
		linkDataGet := files.JSONDataGet{}
		linkDataGet.Link = string(contentBody)
		// Проверяем, есть ли он (пока без валидаций).
		err := linkDataGet.Get(linkFilePath)
		if err != nil {
			httpResp.InternalError(res)
			return
		}
		shortLink := linkDataGet.ShortLink
		// Если нет, генерим ид, сохраняем
		if linkDataGet.ID == "" {
			linkDataSet := files.JSONDataSet{}
			linkDataSet.Link = string(contentBody)
			linkDataSet.ShortLink = getShortURL(confModule.Config.Final.ShortURLAddr, linkID)
			linkDataSet.ID = linkID
			err := linkDataSet.Set(linkFilePath)
			if err != nil {
				httpResp.InternalError(res)
				return
			}
			shortLink = linkDataSet.ShortLink
		}
		// Отдаем 201 ответ с шортлинком
		additional := confModule.Additional{
			Place:     "body",
			InnerData: shortLink,
		}
		httpResp.Created(res, additional)
		return
	} else {
		httpResp.BadRequest(res)
		return
	}
}

type input struct {
	URL string `json:"url"`
}
type output struct {
	Result string `json:"result"`
}

func handlePOSTOverJSON(res http.ResponseWriter, req *http.Request) {
	contentBody, errBody := io.ReadAll(req.Body)
	fmt.Println(contentBody)
	contentBody, errBody = compress.Decompress(contentBody)
	fmt.Println(errBody)
	fmt.Println(contentBody)
	if errBody != nil {
		httpResp.BadRequest(res)
		return
	}
	rootPath, err := pathhandler.ProjectRoot()
	if err != nil {
		httpResp.InternalError(res)
		return
	}
	linkFilePath := filepath.Join(rootPath, confModule.LinkFile)
	// Пришел урл
	linkID := rand.RandStringBytes(8)
	linkDataGet := files.JSONDataGet{}
	var linkData input
	err = json.Unmarshal(contentBody, &linkData)
	if err != nil {
		httpResp.InternalError(res)
		return
	}
	linkDataGet.Link = linkData.URL
	err = linkDataGet.Get(linkFilePath)
	if err != nil {
		httpResp.InternalError(res)
		return
	}
	shortLink := linkDataGet.ShortLink
	// Если нет, генерим ид, сохраняем
	if linkDataGet.ID == "" {
		linkDataSet := files.JSONDataSet{}
		linkDataSet.Link = linkData.URL
		linkDataSet.ShortLink = getShortURL(confModule.Config.Final.ShortURLAddr, linkID)
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
	additional := confModule.Additional{
		Place:     "body",
		InnerData: string(JSONResp),
	}
	httpResp.CreatedJSON(res, additional)
}

func handlePOSTOverJSONGzip(res http.ResponseWriter, req *http.Request) {
	fmt.Println(req.Body, res.Header())
}

func handleOther(res http.ResponseWriter) {
	httpResp.BadRequest(res)
}

/**
 * Route handlers
 */

func ServeHTTP(res http.ResponseWriter, req *http.Request) {
	currentMethod := req.Method

	if currentMethod == "POST" {
		uri := strings.Split(req.URL.Path, "/")
		var controller = "empty"
		if len(uri) >= 3 {
			controller = uri[2]
		}
		logger.PrintLog(logger.INFO, "Controller: "+controller)
		if controller == "shorten" {
			handlePOSTOverJSON(res, req)
			return
		}
		handlePOST(res, req)
		return
	} else if currentMethod == "GET" {
		handleGET(res, req)
		return
	} else {
		handleOther(res)
		return
	}
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

	logger.PrintLog(logger.INFO, "Declaring router")
	r := chi.NewRouter().With(extlogger.Log).With(compress.GzipHandler)
	r.Route("/", func(r chi.Router) {
		r.Post(`/`, ServeHTTP)
		r.Post(`/api/shorten`, ServeHTTP)
		r.Get(`/{test}`, ServeHTTP)
	})

	logger.PrintLog(logger.INFO, "Starting server")
	err = http.ListenAndServe(confModule.Config.Final.AppAddr, r)
	if err != nil {
		logger.PrintLog(logger.FATAL, "Can't start server. "+err.Error())
	}
}
