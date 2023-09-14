package main

import (
	"fmt"
	"github.com/MaximMNsk/go-url-shortener/internal/models/files"
	"github.com/MaximMNsk/go-url-shortener/internal/util/rand"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
)

const LocalHost = "http://localhost"
const LocalPort = "8080"
const LinkFile = "internal/storage/links.json"

func BadRequest(w http.ResponseWriter) {
	http.Error(w, "400 bad request", http.StatusBadRequest)
}

type Additional struct {
	Place     string
	OuterData string
	InnerData string
}

func SuccessAnswer(w http.ResponseWriter, status int, additionalData Additional) {
	w.Header().Add("Content-Type", "text/plain")
	dataLength := len(additionalData.InnerData)
	w.Header().Add("Content-Length", strconv.Itoa(dataLength))
	if additionalData.Place == "header" {
		w.Header().Add(additionalData.OuterData, additionalData.InnerData)
	}
	w.WriteHeader(status)
	if additionalData.Place == "body" {
		_, _ = w.Write([]byte(additionalData.InnerData))
	}
}

func Created(w http.ResponseWriter, addData Additional) {
	SuccessAnswer(w, http.StatusCreated, addData)
}

func TempRedirect(w http.ResponseWriter, addData Additional) {
	SuccessAnswer(w, http.StatusTemporaryRedirect, addData)
}

func getShortURL(linkID string) string {
	return fmt.Sprintf("%s:%s/%s", LocalHost, LocalPort, linkID)
}

func handleGET(res http.ResponseWriter, req *http.Request) {
	_ = req.Header.Get("Content-Type")
	_, errBody := io.ReadAll(req.Body)

	//if strings.Contains(contentType, "text/plain") && errBody == nil {
	if errBody == nil {
		// Пришел ид
		linkData := files.JSONDataGet{}
		requestID := req.URL.Path
		requestID = requestID[1:]
		linkData.ID = requestID
		// Проверяем, есть ли он.
		linkData.Get(filepath.Join(os.Getenv("HOME"), LinkFile))
		if linkData.Link != "" {
			additional := Additional{
				Place:     "header",
				OuterData: "Location",
				InnerData: linkData.Link,
			}
			// Если есть, отдаем 307 редирект
			TempRedirect(res, additional)
		} else {
			// Если нет, отдаем BadRequest
			BadRequest(res)
		}
	} else {
		BadRequest(res)
	}
}

func handlePOST(res http.ResponseWriter, req *http.Request) {
	currentPath := req.URL.Path
	//contentType := req.Header.Get("Content-Type")
	contentBody, errBody := io.ReadAll(req.Body)
	if currentPath == "/" /*&& strings.Contains(contentType, "text/plain")*/ && errBody == nil {
		// Пришел урл
		linkID := rand.RandStringBytes(8)
		linkDataGet := files.JSONDataGet{}
		linkDataGet.Link = string(contentBody)
		// Проверяем, есть ли он (пока без валидаций).
		linkDataGet.Get(filepath.Join(os.Getenv("HOME"), LinkFile))
		shortLink := linkDataGet.ShortLink
		// Если нет, генерим ид, сохраняем
		if linkDataGet.ID == "" {
			linkDataSet := files.JSONDataSet{}
			linkDataSet.Link = string(contentBody)
			linkDataSet.ShortLink = getShortURL(linkID)
			linkDataSet.ID = linkID
			linkDataSet.Set(filepath.Join(os.Getenv("HOME"), LinkFile))
			shortLink = linkDataSet.ShortLink
		}
		// Отдаем 201 ответ с шортлинком
		additional := Additional{
			Place:     "body",
			InnerData: shortLink,
		}
		Created(res, additional)
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
