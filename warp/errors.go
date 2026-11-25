package warp

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"reflect"
)

// Error is the one error shape every API answer goes through. A handler that
// returns an *Error chooses what the client sees; any other error is logged
// and answered as a 500 without its text.
type Error struct {
	Status  int               `json:"-"`
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

// Error returns the message.
func (e *Error) Error() string { return e.Message }

// BadRequest is a 400 answer.
func BadRequest(message string) *Error {
	return &Error{Status: http.StatusBadRequest, Code: "bad_request", Message: message}
}

// Unauthorized is a 401 answer.
func Unauthorized(message string) *Error {
	return &Error{Status: http.StatusUnauthorized, Code: "unauthorized", Message: message}
}

// Forbidden is a 403 answer.
func Forbidden(message string) *Error {
	return &Error{Status: http.StatusForbidden, Code: "forbidden", Message: message}
}

// NotFound is a 404 answer.
func NotFound(message string) *Error {
	return &Error{Status: http.StatusNotFound, Code: "not_found", Message: message}
}

// Conflict is a 409 answer.
func Conflict(message string) *Error {
	return &Error{Status: http.StatusConflict, Code: "conflict", Message: message}
}

// TooLarge is a 413 answer.
func TooLarge(message string) *Error {
	return &Error{Status: http.StatusRequestEntityTooLarge, Code: "too_large", Message: message}
}

// UnsupportedMedia is a 415 answer.
func UnsupportedMedia(message string) *Error {
	return &Error{Status: http.StatusUnsupportedMediaType, Code: "unsupported_media_type", Message: message}
}

// Invalid is a 422 answer that names each field that needs attention.
func Invalid(fields map[string]string) *Error {
	message := fmt.Sprintf("%d fields need attention.", len(fields))
	if len(fields) == 1 {
		message = "1 field needs attention."
	}
	return &Error{Status: http.StatusUnprocessableEntity, Code: "validation", Message: message, Fields: fields}
}

// TooMany is a 429 answer.
func TooMany(message string) *Error {
	return &Error{Status: http.StatusTooManyRequests, Code: "rate_limited", Message: message}
}

// WriteJSON answers with v as JSON. A nil slice is written as [], and a 204
// has no body.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	if status == http.StatusNoContent {
		w.WriteHeader(status)
		return
	}
	if rv := reflect.ValueOf(v); rv.Kind() == reflect.Slice && rv.IsNil() {
		v = []any{}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// WriteError answers with err in the error shape. An *Error, even wrapped, is
// written as it is; anything else is logged and answered as a 500 that does
// not carry its text.
func WriteError(w http.ResponseWriter, r *http.Request, err error) {
	var e *Error
	if !errors.As(err, &e) {
		slog.Error("api error", "method", r.Method, "path", r.URL.Path, "error", err)
		e = &Error{Status: http.StatusInternalServerError, Code: "internal", Message: "Something went wrong on the server; its log has the details."}
	}
	WriteJSON(w, e.Status, struct {
		Error *Error `json:"error"`
	}{e})
}
