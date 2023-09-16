package main

import (
	"fmt"
	"github.com/MaximMNsk/go-url-shortener/internal/models/files"
	"github.com/MaximMNsk/go-url-shortener/internal/util/pathhandler"
	"github.com/MaximMNsk/go-url-shortener/internal/util/rand"
	"github.com/caarlos0/env/v6"
	"github.com/go-chi/chi/v5"
	"io"
	"log"
	"net/http"
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
	var envCfg Config

	err := env.Parse(&envCfg)
	if err != nil {
		log.Fatal(err)
	}

	if envCfg.ShortUrlHost != "" {
		return fmt.Sprintf("%s/%s", envCfg.ShortUrlHost, linkID)
	} else if flagShortURLAddr != "" {
		return fmt.Sprintf("%s/%s", flagShortURLAddr, linkID)
	}
	return fmt.Sprintf("%s:%s/%s", LocalHost, LocalPort, linkID)
}

func handleGET(res http.ResponseWriter, req *http.Request) {
	_ = req.Header.Get("Content-Type")
	_, errBody := io.ReadAll(req.Body)
	rootPath, _ := pathhandler.ProjectRoot()
	linkFilePath := filepath.Join(rootPath, LinkFile)
	//if strings.Contains(contentType, "text/plain") {
	if errBody == nil {
		// Пришел ид
		linkData := files.JSONDataGet{}
		requestID := req.URL.Path[1:]
		linkData.ID = requestID
		// Проверяем, есть ли он.
		linkData.Get(linkFilePath)
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
	//}
}

func handlePOST(res http.ResponseWriter, req *http.Request) {
	currentPath := req.URL.Path
	//contentType := req.Header.Get("Content-Type")
	contentBody, errBody := io.ReadAll(req.Body)
	rootPath, _ := pathhandler.ProjectRoot()
	linkFilePath := filepath.Join(rootPath, LinkFile)

	if currentPath == "/" /*&& strings.Contains(contentType, "text/plain")*/ && errBody == nil {
		// Пришел урл
		linkID := rand.RandStringBytes(8)
		linkDataGet := files.JSONDataGet{}
		linkDataGet.Link = string(contentBody)
		// Проверяем, есть ли он (пока без валидаций).
		linkDataGet.Get(linkFilePath)
		shortLink := linkDataGet.ShortLink
		// Если нет, генерим ид, сохраняем
		if linkDataGet.ID == "" {
			linkDataSet := files.JSONDataSet{}
			linkDataSet.Link = string(contentBody)
			linkDataSet.ShortLink = getShortURL(linkID)
			linkDataSet.ID = linkID
			linkDataSet.Set(linkFilePath)
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

func handleOther(res http.ResponseWriter) {
	BadRequest(res)
}

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

type Config struct {
	AppHost      string `env:"SERVER_ADDRESS"`
	ShortUrlHost string `env:"BASE_URL"`
}

func main() {

	var err error
	var appHost string

	r := chi.NewRouter()
	r.Route("/", func(r chi.Router) {
		r.Post(`/`, handleMainPage)
		r.Get(`/{test}`, handleMainPage)
	})

	var envCfg Config

	err = env.Parse(&envCfg)
	if err != nil {
		log.Fatal(err)
	}

	if envCfg.AppHost == "" {
		parseFlags()
		appHost = flagRunAddr
	} else {
		appHost = envCfg.AppHost
	}

	err = http.ListenAndServe(appHost, r)
	if err != nil {
		log.Fatal(err)
	}

}
