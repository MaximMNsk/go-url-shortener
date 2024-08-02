// Package http - слой отправки ответов сервера.
package http

import (
	"net/http"
)

// Additional - структура для передачи дополнительных данных для ответа сервера.
type Additional struct {
	Place     string
	OuterData string
	InnerData string
}

// BadRequest - отдает по http статус 400.
func BadRequest(w http.ResponseWriter) {
	http.Error(w, "400 bad request", http.StatusBadRequest)
}

// InternalError - отдает по http статус 500.
func InternalError(w http.ResponseWriter) {
	http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
}

// TempRedirect - отдает по http статус 307, а так же УРЛ в заголовке.
func TempRedirect(w http.ResponseWriter, addData Additional) {
	successAnswer(w, http.StatusTemporaryRedirect, addData)
}
func successAnswer(w http.ResponseWriter, status int, additionalData Additional) {
	w.Header().Add("Content-Type", "text/plain")
	if additionalData.Place == "header" {
		w.Header().Add(additionalData.OuterData, additionalData.InnerData)
	}
	w.WriteHeader(status)
	if additionalData.Place == "body" {
		_, _ = w.Write([]byte(additionalData.InnerData))
	}
}

// Ok - отдает по http статус 200.
func Ok(w http.ResponseWriter) {
	addData := Additional{}
	successAnswer(w, http.StatusOK, addData)
}

// Created - отдает по http статус 201, а так же УРЛ в теле ответа.
func Created(w http.ResponseWriter, addData Additional) {
	successAnswer(w, http.StatusCreated, addData)
}

// Conflict - отдает по http статус 409.
func Conflict(w http.ResponseWriter, addData Additional) {
	successAnswer(w, http.StatusConflict, addData)
}

// Forbidden - отдает по http статус 403.
func Forbidden(w http.ResponseWriter) {
	addData := Additional{}
	successAnswer(w, http.StatusForbidden, addData)
}

func successAnswerJSON(w http.ResponseWriter, status int, additionalData Additional) {
	w.Header().Add("Content-Type", "application/json")
	if additionalData.Place == "header" {
		w.Header().Add(additionalData.OuterData, additionalData.InnerData)
	}
	w.WriteHeader(status)
	if additionalData.Place == "body" {
		_, _ = w.Write([]byte(additionalData.InnerData))
	}
}

// CreatedJSON - отдает по http статус 201, а так же УРЛ в теле ответа в формате JSON.
func CreatedJSON(w http.ResponseWriter, addData Additional) {
	successAnswerJSON(w, http.StatusCreated, addData)
}

// ConflictJSON - отдает по http статус 409.
func ConflictJSON(w http.ResponseWriter, addData Additional) {
	successAnswerJSON(w, http.StatusConflict, addData)
}

// OkAdditionalJSON - отдает по http статус 200.
func OkAdditionalJSON(w http.ResponseWriter, addData Additional) {
	successAnswerJSON(w, http.StatusOK, addData)
}

// NoContent - отдает по http статус 204.
func NoContent(w http.ResponseWriter, addData Additional) {
	successAnswerJSON(w, http.StatusNoContent, addData)
}

// Unauthorized - отдает по http статус 401.
func Unauthorized(w http.ResponseWriter, addData Additional) {
	successAnswerJSON(w, http.StatusUnauthorized, addData)
}

// Accepted - отдает по http статус 202.
func Accepted(w http.ResponseWriter, addData Additional) {
	successAnswerJSON(w, http.StatusAccepted, addData)
}

// Gone - отдает по http статус 410.
func Gone(w http.ResponseWriter, addData Additional) {
	successAnswerJSON(w, http.StatusGone, addData)
}

// Shutdown - заглушка на время остановки сервера. Отдает 503 ответ.
func Shutdown(w http.ResponseWriter) {
	http.Error(w, "503 service unavailable", http.StatusServiceUnavailable)
}
