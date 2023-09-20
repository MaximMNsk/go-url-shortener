package http

import (
	confModule "github.com/MaximMNsk/go-url-shortener/server/config"
	"net/http"
	"strconv"
)

/**
 * Responses
 */

func BadRequest(w http.ResponseWriter) {
	http.Error(w, "400 bad request", http.StatusBadRequest)
}

func InternalError(w http.ResponseWriter) {
	http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
}

func successAnswer(w http.ResponseWriter, status int, additionalData confModule.Additional) {
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

func Created(w http.ResponseWriter, addData confModule.Additional) {
	successAnswer(w, http.StatusCreated, addData)
}

func TempRedirect(w http.ResponseWriter, addData confModule.Additional) {
	successAnswer(w, http.StatusTemporaryRedirect, addData)
}