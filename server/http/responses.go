package http

import (
	"net/http"
)

/**
 * Responses
 */

type Additional struct {
	Place     string
	OuterData string
	InnerData string
}

func BadRequest(w http.ResponseWriter) {
	http.Error(w, "400 bad request", http.StatusBadRequest)
}

func InternalError(w http.ResponseWriter) {
	http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
}

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

func Ok(w http.ResponseWriter) {
	addData := Additional{}
	successAnswer(w, http.StatusOK, addData)
}

func Created(w http.ResponseWriter, addData Additional) {
	successAnswer(w, http.StatusCreated, addData)
}

func Conflict(w http.ResponseWriter, addData Additional) {
	successAnswer(w, http.StatusConflict, addData)
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

func CreatedJSON(w http.ResponseWriter, addData Additional) {
	successAnswerJSON(w, http.StatusCreated, addData)
}

func ConflictJSON(w http.ResponseWriter, addData Additional) {
	successAnswerJSON(w, http.StatusConflict, addData)
}

func OkAdditionalJSON(w http.ResponseWriter, addData Additional) {
	successAnswerJSON(w, http.StatusOK, addData)
}

func NoContent(w http.ResponseWriter, addData Additional) {
	successAnswerJSON(w, http.StatusNoContent, addData)
}

func Unauthorized(w http.ResponseWriter, addData Additional) {
	successAnswerJSON(w, http.StatusUnauthorized, addData)
}
