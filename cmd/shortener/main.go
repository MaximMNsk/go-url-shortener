package main

import (
	"fmt"
	"github.com/MaximMNsk/go-url-shortener/internal/models/files"
	"github.com/MaximMNsk/go-url-shortener/internal/util/rand"
	"io"
	"net/http"
)

const LocalHost = "http://localhost"
const LocalPort = "8080"
const LinkFile = "internal/storage/links.json"

func BadRequest(w http.ResponseWriter) {
	http.Error(w, "400 bad request", http.StatusBadRequest)
}

func Created(w http.ResponseWriter, addData string) {
	w.Header().Add("Content-Type", "text/plain")
	w.WriteHeader(http.StatusCreated)
	if addData != "" {
		_, _ = w.Write([]byte(addData))
	}
}

func getShortUrl(linkId string) string {
	return fmt.Sprintf("%s:%s/%s", LocalHost, LocalPort, linkId)
}

func handleMainPage(res http.ResponseWriter, req *http.Request) {
	currentMethod := req.Method
	currentPath := req.URL.Path
	contentType := req.Header.Get("Content-Type")
	contentBody, errBody := io.ReadAll(req.Body)

	if currentMethod == "POST" {
		if currentPath == "/" && contentType == "text/plain" && errBody == nil {
			// Пришел урл
			linkId := rand.RandStringBytes(8)
			linkData := files.JsonData{}
			linkData.Link = string(contentBody)
			// Проверяем, есть ли он (пока без валидаций).
			linkData.Get(LinkFile)
			// Если нет, генерим ид, сохраняем
			if linkData.Id == "" {
				linkData.ShortLink = getShortUrl(linkId)
				linkData.Id = linkId
				linkData.Set(LinkFile)
			}
			// Отдаем 307 редирект
			http.Redirect(res, req, linkData.ShortLink, 307)
		} else {
			BadRequest(res)
		}
	} else if currentMethod == "GET" {
		if contentType == "text/plain" && errBody == nil {
			// Пришел ид
			linkData := files.JsonData{}
			requestId := req.URL.String()
			requestId = requestId[1:]
			//fmt.Println(requestId)
			linkData.Id = requestId
			// Проверяем, есть ли он.
			linkData.Get(LinkFile)
			if linkData.Link != "" {
				// Если есть, отдаем 201 ответ с шортлинком
				Created(res, linkData.ShortLink)
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
