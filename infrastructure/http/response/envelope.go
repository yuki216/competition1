package response

import (
	"encoding/json"
	"net/http"
)

type Envelope struct {
	Status  bool        `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

func WriteJSON(w http.ResponseWriter, statusCode int, status bool, message string, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)

	envelope := Envelope{
		Status:  status,
		Message: message,
		Data:    data,
	}

	json.NewEncoder(w).Encode(envelope)
}

func Success(w http.ResponseWriter, statusCode int, message string, data interface{}) {
	WriteJSON(w, statusCode, true, message, data)
}

func Error(w http.ResponseWriter, statusCode int, message string) {
	WriteJSON(w, statusCode, false, message, nil)
}

func BadRequest(w http.ResponseWriter, message string) {
	Error(w, http.StatusBadRequest, message)
}

func Unauthorized(w http.ResponseWriter, message string) {
	Error(w, http.StatusUnauthorized, message)
}

func Forbidden(w http.ResponseWriter, message string) {
	Error(w, http.StatusForbidden, message)
}

func NotFound(w http.ResponseWriter, message string) {
	Error(w, http.StatusNotFound, message)
}

func Conflict(w http.ResponseWriter, message string) {
	Error(w, http.StatusConflict, message)
}

func UnprocessableEntity(w http.ResponseWriter, message string) {
	Error(w, http.StatusUnprocessableEntity, message)
}

func InternalServerError(w http.ResponseWriter, message string) {
	Error(w, http.StatusInternalServerError, message)
}