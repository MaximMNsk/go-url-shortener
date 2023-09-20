package main

import (
	"fmt"
	"github.com/MaximMNsk/go-url-shortener/internal/models/files"
	"github.com/MaximMNsk/go-url-shortener/internal/util/pathhandler"
	"github.com/MaximMNsk/go-url-shortener/internal/util/rand"
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
	} else {
		httpResp.BadRequest(res)
	}
}

func handleOther(res http.ResponseWriter) {
	httpResp.BadRequest(res)
}

/**
 * Route handlers
 */

func handleMainPage(res http.ResponseWriter, req *http.Request) {
	currentMethod := req.Method

	if currentMethod == "POST" {
		handlePOST(res, req)
	} else if currentMethod == "GET" {
		handleGET(res, req)
	} else {
		handleOther(res)
	}
}

/**
 * Executor
 */

func main() {

	err := confModule.HandleConfig()
	if err != nil {
		panic(err)
	}

	r := chi.NewRouter()
	r.Route("/", func(r chi.Router) {
		r.Post(`/`, handleMainPage)
		r.Get(`/{test}`, handleMainPage)
	})

	err = http.ListenAndServe(confModule.Config.Final.AppAddr, r)
	if err != nil {
		panic(err)
	}

}
