package main

import (
	"fmt"
	"github.com/MaximMNsk/go-url-shortener/internal/models/files"
	"github.com/MaximMNsk/go-url-shortener/internal/util/rand"
	"io"
	"net/http"
	"strconv"
	"strings"
)

const LocalHost = "http://localhost"
const LocalPort = "8080"
const LinkFile = "internal/storage/links.json"

func BadRequest(w http.ResponseWriter) {
	http.Error(w, "400 bad request", http.StatusBadRequest)
}

func Created(w http.ResponseWriter, addData string) {
	w.Header().Add("Content-Type", "text/plain")
	dataLength := len(addData)
	w.Header().Add("Content-Length", strconv.Itoa(dataLength))
	w.WriteHeader(http.StatusCreated)
	if addData != "" {
		_, _ = w.Write([]byte(addData))
	}
}

func TempRedirect(w http.ResponseWriter, req *http.Request, addData string) {
	http.Redirect(w, req, addData, http.StatusTemporaryRedirect)
}

func getShortURL(linkID string) string {
	return fmt.Sprintf("%s:%s/%s", LocalHost, LocalPort, linkID)
}

func handleMainPage(res http.ResponseWriter, req *http.Request) {
	currentMethod := req.Method
	currentPath := req.URL.Path
	contentType := req.Header.Get("Content-Type")
	contentBody, errBody := io.ReadAll(req.Body)

	if currentMethod == "POST" {
		if currentPath == "/" && strings.Contains(contentType, "text/plain") && errBody == nil {
			// Пришел урл
			linkID := rand.RandStringBytes(8)
			linkDataGet := files.JSONDataGet{}
			linkDataGet.Link = string(contentBody)
			// Проверяем, есть ли он (пока без валидаций).
			linkDataGet.Get(LinkFile)
			shortLink := linkDataGet.ShortLink
			// Если нет, генерим ид, сохраняем
			if linkDataGet.ID == "" {
				linkDataSet := files.JSONDataSet{}
				linkDataSet.Link = string(contentBody)
				linkDataSet.ShortLink = getShortURL(linkID)
				linkDataSet.ID = linkID
				linkDataSet.Set(LinkFile)
				shortLink = linkDataSet.ShortLink
			}
			// Отдаем 201 ответ с шортлинком
			Created(res, shortLink)
		} else {
			BadRequest(res)
		}
	} else if currentMethod == "GET" {
		if strings.Contains(contentType, "text/plain") && errBody == nil {
			// Пришел ид
			linkData := files.JSONDataGet{}
			requestID := req.URL.String()
			requestID = requestID[1:]
			linkData.ID = requestID
			// Проверяем, есть ли он.
			linkData.Get(LinkFile)
			if linkData.Link != "" {
				// Если есть, отдаем 307 редирект
				TempRedirect(res, req, linkData.Link)
			} else {
				// Если нет, отдаем BadRequest
				BadRequest(res)
			}
		}
	} else {
		BadRequest(res)
	}
}

func main() {

	mux := http.NewServeMux()
	mux.HandleFunc(`/`, handleMainPage)

	err := http.ListenAndServe(`:8080`, mux)
	if err != nil {
		panic(err)
	}

}
