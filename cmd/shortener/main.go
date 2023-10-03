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
)

/**
 * Request type handlers
 */

func getShortURL(hostPort, linkID string) string {
	return fmt.Sprintf("%s/%s", hostPort, linkID)
}

func handleGET(res http.ResponseWriter, req *http.Request) {
	//contentType := req.Header.Get("Content-Type")
	//if contentType != "text/plain" {
	//	httpResp.BadRequest(res)
	//	return
	//}
	rootPath, err := pathhandler.ProjectRoot()
	if err != nil {
		httpResp.InternalError(res)
		return
	}
	linkFilePath := filepath.Join(rootPath, confModule.LinkFile)
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
	//contentType := req.Header.Get("Content-Type")
	//if contentType != "text/plain" {
	//	httpResp.BadRequest(res)
	//	return
	//}

	contentBody, errBody := io.ReadAll(req.Body)
	if errBody != nil {
		httpResp.BadRequest(res)
		return
	}

	contentBody, errDecompress := compress.HandleInputValue(contentBody)
	if errDecompress != nil {
		httpResp.InternalError(res)
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
	linkDataGet.Link = string(contentBody)
	// Проверяем, есть ли он (пока без валидаций).
	err = linkDataGet.Get(linkFilePath)
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
	shortLinkByte := []byte(shortLink)
	shortLinkByte, err = compress.HandleOutputValue(shortLinkByte)
	if err != nil {
		httpResp.InternalError(res)
		return
	}
	//fmt.Println(string(shortLinkByte))
	additional := confModule.Additional{
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

func handlePOSTOverJSON(res http.ResponseWriter, req *http.Request) {
	//contentType := req.Header.Get("Content-Type")
	//if contentType != "application/json" {
	//	httpResp.BadRequest(res)
	//	return
	//}

	contentBody, errBody := io.ReadAll(req.Body)
	if errBody != nil {
		httpResp.BadRequest(res)
		return
	}

	contentBody, errDecompress := compress.HandleInputValue(contentBody)
	//fmt.Println(errDecompress)
	if errDecompress != nil {
		httpResp.InternalError(res)
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
	JSONResp, err = compress.HandleOutputValue(JSONResp)
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

	logger.PrintLog(logger.INFO, "Declaring router")
	r := chi.NewRouter().With(extlogger.Log).With(compress.GzipHandler).With(handleOther)
	r.Route("/", func(r chi.Router) {
		r.Post(`/`, handlePOST)
		r.Post(`/api/shorten`, handlePOSTOverJSON)
		r.Get(`/{test}`, handleGET)
	})

	logger.PrintLog(logger.INFO, "Starting server")
	err = http.ListenAndServe(confModule.Config.Final.AppAddr, r)
	if err != nil {
		logger.PrintLog(logger.FATAL, "Can't start server. "+err.Error())
	}
}
