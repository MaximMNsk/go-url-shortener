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

func SuccessAnswer(w http.ResponseWriter, status int, addData string, placeData string) {
	w.Header().Add("Content-Type", "text/plain")
	dataLength := len(addData)
	w.Header().Add("Content-Length", strconv.Itoa(dataLength))
	if placeData == "header" {
		w.Header().Add("Location", addData)
	}
	w.WriteHeader(status)
	if addData != "" && placeData == "body" {
		_, _ = w.Write([]byte(addData))
	}
}

func Created(w http.ResponseWriter, addData string) {
	SuccessAnswer(w, http.StatusCreated, addData, "body")
}

func TempRedirect(w http.ResponseWriter, addData string) {
	SuccessAnswer(w, http.StatusTemporaryRedirect, addData, "header")
}

func getShortURL(linkID string) string {
	return fmt.Sprintf("%s:%s/%s", LocalHost, LocalPort, linkID)
}

func handleGET(res http.ResponseWriter, req *http.Request) {
	contentType := req.Header.Get("Content-Type")
	_, errBody := io.ReadAll(req.Body)

	if strings.Contains(contentType, "text/plain") && errBody == nil {
		// Пришел ид
		linkData := files.JSONDataGet{}
		requestID := req.URL.String()
		requestID = requestID[1:]
		linkData.ID = requestID
		// Проверяем, есть ли он.
		//fmt.Println(linkData)
		linkData.Get(LinkFile)
		//fmt.Println(linkData)
		if linkData.Link != "" {
			// Если есть, отдаем 307 редирект
			TempRedirect(res, linkData.Link)
		} else {
			// Если нет, отдаем BadRequest
			BadRequest(res)
		}
	}
}

func handlePOST(res http.ResponseWriter, req *http.Request) {
	currentPath := req.URL.Path
	contentType := req.Header.Get("Content-Type")
	contentBody, errBody := io.ReadAll(req.Body)
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
}

func handleOther(res http.ResponseWriter, req *http.Request) {
	BadRequest(res)
}

func handleMainPage(res http.ResponseWriter, req *http.Request) {
	currentMethod := req.Method

	if currentMethod == "POST" {
		handlePOST(res, req)
	} else if currentMethod == "GET" {
		handleGET(res, req)
	} else {
		handleOther(res, req)
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
