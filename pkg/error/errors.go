package error

import (
	"errors"
	"net/http"
)

type AppError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Status  int    `json:"status"`
}

func (e *AppError) Error() string {
	return e.Message
}

var (
	ErrBadRequest     = &AppError{Code: "BAD_REQUEST", Message: "Bad request", Status: http.StatusBadRequest}
	ErrUnauthorized   = &AppError{Code: "UNAUTHORIZED", Message: "Unauthorized", Status: http.StatusUnauthorized}
	ErrNotFound       = &AppError{Code: "NOT_FOUND", Message: "Not found", Status: http.StatusNotFound}
	ErrInternalServer = &AppError{Code: "INTERNAL_ERROR", Message: "Internal server error", Status: http.StatusInternalServerError}
	ErrConflict       = &AppError{Code: "CONFLICT", Message: "Conflict", Status: http.StatusConflict}
)

func NewBadRequest(message string) *AppError {
	return &AppError{Code: "BAD_REQUEST", Message: message, Status: http.StatusBadRequest}
}

func NewUnauthorized(message string) *AppError {
	return &AppError{Code: "UNAUTHORIZED", Message: message, Status: http.StatusUnauthorized}
}

func NewNotFound(message string) *AppError {
	return &AppError{Code: "NOT_FOUND", Message: message, Status: http.StatusNotFound}
}

func NewInternalServer(message string) *AppError {
	return &AppError{Code: "INTERNAL_ERROR", Message: message, Status: http.StatusInternalServerError}
}

func NewConflict(message string) *AppError {
	return &AppError{Code: "CONFLICT", Message: message, Status: http.StatusConflict}
}

func MapError(err error) *AppError {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr
	}
	
	// Map common errors
	switch err.Error() {
	case "invalid email format", "password must be at least 8 characters":
		return NewBadRequest(err.Error())
	case "invalid email or password":
		return NewUnauthorized(err.Error())
	case "user not found":
		return NewNotFound(err.Error())
	default:
		return NewInternalServer("An unexpected error occurred")
	}
}