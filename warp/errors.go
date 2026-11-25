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
// has no body. A value that cannot be written as JSON, such as a NaN or a
// channel, is logged and answered as a 500.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	if status == http.StatusNoContent {
		w.WriteHeader(status)
		return
	}
	if rv := reflect.ValueOf(v); rv.Kind() == reflect.Slice && rv.IsNil() {
		v = []any{}
	}
	b, err := json.Marshal(v)
	if err != nil {
		slog.Error("warp: the answer cannot be written as JSON", "status", status, "error", err)
		status, b = http.StatusInternalServerError, internalBody
	} else {
		b = append(b, '\n')
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(b)
}

// internal is the answer to an error the client is not told about.
func internal() *Error {
	return &Error{Status: http.StatusInternalServerError, Code: "internal", Message: "Something went wrong on the server; its log has the details."}
}

// internalBody is internal()'s answer, newline included. It is shared by
// every request: it is written as it is, never appended to.
var internalBody = func() []byte {
	b, _ := json.Marshal(errorBody{internal()})
	return append(b, '\n')
}()

type errorBody struct {
	Error *Error `json:"error"`
}

// WriteError answers with err in the error shape. An *Error, even wrapped, is
// written as it is; anything else, and an *Error whose Status is not from 400
// to 599, is logged and answered as a 500 that does not carry its text.
func WriteError(w http.ResponseWriter, r *http.Request, err error) {
	var e *Error
	switch {
	case !errors.As(err, &e):
		slog.Error("api error", "method", r.Method, "path", r.URL.Path, "error", err)
		e = internal()
	case e.Status < 400 || e.Status > 599:
		slog.Error("api error with a status that is not an error status", "method", r.Method, "path", r.URL.Path, "status", e.Status, "error", err)
		e = internal()
	}
	WriteJSON(w, e.Status, errorBody{e})
}
