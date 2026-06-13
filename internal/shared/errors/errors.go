package errors

import (
	"errors"
	"net/http"
)

type AppError struct {
	Code    int    
	Message string 
	Err     error  
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

func (e *AppError) Unwrap() error { return e.Err }

func New(code int, msg string) *AppError {
	return &AppError{Code: code, Message: msg}
}

func Wrap(code int, msg string, err error) *AppError {
	return &AppError{Code: code, Message: msg, Err: err}
}

func ToHTTPStatus(err error) int {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code
	}
	return http.StatusInternalServerError
}

func IsAppError(err error) bool {
	var ae *AppError
	return errors.As(err, &ae)
}

var (
	ErrNotFound            = New(http.StatusNotFound, "resource not found")
	ErrUnauthorized        = New(http.StatusUnauthorized, "authentication required")
	ErrForbidden           = New(http.StatusForbidden, "access denied")
	ErrConflict            = New(http.StatusConflict, "resource already exists")
	ErrUnprocessable       = New(http.StatusUnprocessableEntity, "request could not be processed")
	ErrInternalServer      = New(http.StatusInternalServerError, "internal server error")
	ErrBadRequest          = New(http.StatusBadRequest, "bad request")
	ErrServiceUnavailable  = New(http.StatusServiceUnavailable, "service temporarily unavailable")
)
