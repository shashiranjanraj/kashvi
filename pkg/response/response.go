package response

import (
	"encoding/json"
	"net/http"

	"github.com/shashiranjanraj/kashvi/pkg/orm"
)

type envelope struct {
	Status  int         `json:"status"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Errors  interface{} `json:"errors,omitempty"`
}

func write(w http.ResponseWriter, status int, body envelope) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body) //nolint:errcheck
}

// Success sends a 200 JSON response with data.
func Success(w http.ResponseWriter, data interface{}) {
	write(w, http.StatusOK, envelope{Status: http.StatusOK, Data: data})
}

// Created sends a 201 JSON response with data.
func Created(w http.ResponseWriter, data interface{}) {
	write(w, http.StatusCreated, envelope{Status: http.StatusCreated, Data: data})
}

// Error sends a JSON error response.
func Error(w http.ResponseWriter, status int, message string) {
	write(w, status, envelope{Status: status, Message: message})
}

// ValidationError sends a 422 with field-level error map.
func ValidationError(w http.ResponseWriter, errs map[string]string) {
	write(w, http.StatusUnprocessableEntity, envelope{
		Status:  http.StatusUnprocessableEntity,
		Message: "Validation failed",
		Errors:  errs,
	})
}

// Paginated sends a 200 response with data and pagination metadata.
func Paginated(w http.ResponseWriter, data interface{}, pagination orm.Pagination) {
	body := map[string]interface{}{
		"items":      data,
		"pagination": pagination,
	}
	write(w, http.StatusOK, envelope{Status: http.StatusOK, Data: body})
}

// Unauthorized sends a 401.
func Unauthorized(w http.ResponseWriter) {
	Error(w, http.StatusUnauthorized, "Unauthorized")
}

// Forbidden sends a 403.
func Forbidden(w http.ResponseWriter) {
	Error(w, http.StatusForbidden, "Forbidden")
}

// NotFound sends a 404.
func NotFound(w http.ResponseWriter) {
	Error(w, http.StatusNotFound, "Not found")
}
