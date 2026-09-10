package errors

import (
	"fmt"
	"net/http"
)

type APIError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("api error %d: %s", e.Code, e.Message)
}

func NotFound(msg string) *APIError {
	return &APIError{Code: http.StatusNotFound, Message: msg}
}

func BadRequest(msg string) *APIError {
	return &APIError{Code: http.StatusBadRequest, Message: msg}
}

func Internal(msg string) *APIError {
	return &APIError{Code: http.StatusInternalServerError, Message: msg}
}

func Conflict(msg string) *APIError {
	return &APIError{Code: http.StatusConflict, Message: msg}
}
