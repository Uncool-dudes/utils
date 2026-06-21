package middleware

import (
	"encoding/json"
	"errors"
	"net/http"
)

// HTTPError is a request error with an HTTP status code, machine-readable code, and message.
// Return one from a HandlerFunc to produce a structured JSON error response.
type HTTPError struct {
	Status  int    `json:"-"`
	Code    string `json:"code"`
	Message string `json:"message"`
	cause   error
}

func (e *HTTPError) Error() string { return e.Message }
func (e *HTTPError) Unwrap() error { return e.cause }

func BadRequest(code, msg string) *HTTPError {
	return &HTTPError{http.StatusBadRequest, code, msg, nil}
}

func Unauthorized(code, msg string) *HTTPError {
	return &HTTPError{http.StatusUnauthorized, code, msg, nil}
}

func NotFound(code, msg string) *HTTPError { return &HTTPError{http.StatusNotFound, code, msg, nil} }

func Unprocessable(code, msg string) *HTTPError {
	return &HTTPError{http.StatusUnprocessableEntity, code, msg, nil}
}

func Internal(err error) *HTTPError {
	return &HTTPError{http.StatusInternalServerError, "ERR_INTERNAL", "internal server error", err}
}

// HandlerFunc is an http.HandlerFunc that returns an error.
// Use Handle to adapt it to the standard http.Handler interface.
type HandlerFunc func(w http.ResponseWriter, r *http.Request) error

// Handle adapts an error-returning HandlerFunc to http.HandlerFunc.
// HTTPError values map to their status code; all others become 500.
func Handle(h HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := h(w, r); err != nil {
			WriteError(w, err)
		}
	}
}

// WriteError writes a JSON error body. Safe to call from plain http.HandlerFunc.
func WriteError(w http.ResponseWriter, err error) {
	var he *HTTPError
	if errors.As(err, &he) {
		writeJSON(w, he.Status, he)
		return
	}
	writeJSON(w, http.StatusInternalServerError, &HTTPError{Code: "ERR_INTERNAL", Message: "internal server error"})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
