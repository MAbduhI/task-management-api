package domain

import (
	"errors"
	"net/http"
	"time"
)

var (
	ErrNotFound            = errors.New("resource not found")
	ErrBadRequest          = errors.New("bad request")
	ErrUnauthorized        = errors.New("unauthorized")
	ErrForbidden           = errors.New("forbidden")
	ErrConflict            = errors.New("conflict")
	ErrValidation          = errors.New("validation failed")
	ErrInternalServer      = errors.New("internal server error")
	ErrIdempotencyConflict = errors.New("request with this idempotency key is already in progress")
)

type AppError struct {
	Status    int       `json:"status"`
	Code      string    `json:"code"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	Err       error     `json:"-"`
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Err
}

func NewAppError(status int, code string, message string, underlying error) *AppError {
	return &AppError{
		Status:    status,
		Code:      code,
		Message:   message,
		Timestamp: time.Now().UTC(),
		Err:       underlying,
	}
}

func ErrBadReq(code, message string, err error) *AppError {
	if code == "" {
		code = "BAD_REQUEST"
	}
	return NewAppError(http.StatusBadRequest, code, message, err)
}

func ErrUnauth(message string) *AppError {
	return NewAppError(http.StatusUnauthorized, "UNAUTHORIZED", message, ErrUnauthorized)
}

func ErrForbid(message string) *AppError {
	return NewAppError(http.StatusForbidden, "FORBIDDEN", message, ErrForbidden)
}

func ErrNotFoundCustom(code, message string) *AppError {
	if code == "" {
		code = "NOT_FOUND"
	}
	return NewAppError(http.StatusNotFound, code, message, ErrNotFound)
}

func ErrConflictCustom(code, message string) *AppError {
	if code == "" {
		code = "CONFLICT"
	}
	return NewAppError(http.StatusConflict, code, message, ErrConflict)
}

func ErrInternal(err error) *AppError {
	return NewAppError(http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Internal server error", err)
}
