package apperror

import (
	"fmt"
	"net/http"
)

// Error represents a structured HTTP error.
// It implements the standard error interface.
type Error struct {
	StatusCode int
	Message    string
	Err        error // Optional internal error for logging purposes
}

// Error implements the standard error interface.
func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// Unwrap returns the underlying error.
func (e *Error) Unwrap() error {
	return e.Err
}

// New creates a new structured Error.
func New(statusCode int, message string, err error) *Error {
	return &Error{
		StatusCode: statusCode,
		Message:    message,
		Err:        err,
	}
}

// BadRequest returns a 400 Bad Request error.
func BadRequest(message string) *Error {
	return New(http.StatusBadRequest, message, nil)
}

// Unauthorized returns a 401 Unauthorized error.
func Unauthorized(message string) *Error {
	return New(http.StatusUnauthorized, message, nil)
}

// Forbidden returns a 403 Forbidden error.
func Forbidden(message string) *Error {
	return New(http.StatusForbidden, message, nil)
}

// NotFound returns a 404 Not Found error.
func NotFound(message string) *Error {
	return New(http.StatusNotFound, message, nil)
}

// Internal returns a 500 Internal Server Error.
// It wraps the original error 'err' so it can be logged, but the client
// only sees the generic "Internal Server Error" message, or the provided message.
func Internal(err error, message ...string) *Error {
	msg := "Internal Server Error"
	if len(message) > 0 && message[0] != "" {
		msg = message[0]
	}
	return New(http.StatusInternalServerError, msg, err)
}

// TryConvert attempts to convert a standard error to an *apperror.Error.
// If it fails, it returns an Internal Server Error wrapping the original error.
func TryConvert(err error) *Error {
	if err == nil {
		return nil
	}
	if appErr, ok := err.(*Error); ok {
		return appErr
	}
	return Internal(err)
}
