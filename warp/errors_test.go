package warp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"
)

func writeErr(t *testing.T, err error) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	WriteError(w, httptest.NewRequest("GET", "/api/x", nil), err)
	return w
}

func decode(t *testing.T, body string) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(body), &m); err != nil {
		t.Fatalf("body %q: %v", body, err)
	}
	return m
}

func TestWriteErrorShape(t *testing.T) {
	w := writeErr(t, Invalid(map[string]string{"email": "Enter an email address.", "name": "This is required."}))
	if w.Code != 422 {
		t.Fatalf("status %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("content-type %q", ct)
	}
	want := decode(t, `{"error":{"code":"validation","message":"2 fields need attention.","fields":{"email":"Enter an email address.","name":"This is required."}}}`)
	if got := decode(t, w.Body.String()); !reflect.DeepEqual(got, want) {
		t.Fatalf("body %v", got)
	}
}

func TestInvalidSingular(t *testing.T) {
	if m := Invalid(map[string]string{"a": "b"}).Message; m != "1 field needs attention." {
		t.Fatalf("message %q", m)
	}
}

func TestWriteErrorOmitsFields(t *testing.T) {
	w := writeErr(t, NotFound("No product has id 7."))
	if w.Code != 404 {
		t.Fatalf("status %d", w.Code)
	}
	e := decode(t, w.Body.String())["error"].(map[string]any)
	if _, ok := e["fields"]; ok {
		t.Fatalf("fields present: %v", e)
	}
}

func TestWriteErrorHidesPlainErrors(t *testing.T) {
	logs := captureLog(t)
	w := writeErr(t, errors.New("pq: password authentication failed"))
	if w.Code != 500 {
		t.Fatalf("status %d", w.Code)
	}
	if !strings.Contains(logs.String(), "pq: password authentication failed") {
		t.Fatalf("log has no error: %q", logs.String())
	}
	if strings.Contains(w.Body.String(), "pq") {
		t.Fatalf("body leaks: %s", w.Body)
	}
	e := decode(t, w.Body.String())["error"].(map[string]any)
	if e["code"] != "internal" || e["message"] != "Something went wrong on the server; its log has the details." {
		t.Fatalf("error %v", e)
	}
}

func TestWriteErrorWrapped(t *testing.T) {
	if w := writeErr(t, fmt.Errorf("load: %w", Forbidden("Ask an owner for access."))); w.Code != 403 {
		t.Fatalf("status %d", w.Code)
	}
}

func TestWriteJSONNilSlice(t *testing.T) {
	w := httptest.NewRecorder()
	WriteJSON(w, 200, []string(nil))
	if got := strings.TrimSpace(w.Body.String()); got != "[]" {
		t.Fatalf("body %q", got)
	}
}

func TestWriteJSONNoContent(t *testing.T) {
	w := httptest.NewRecorder()
	WriteJSON(w, 204, map[string]string{"a": "b"})
	if w.Code != 204 || w.Body.Len() != 0 {
		t.Fatalf("status %d body %q", w.Code, w.Body)
	}
}

func TestStatuses(t *testing.T) {
	for _, c := range []struct {
		err    *Error
		status int
		code   string
	}{
		{BadRequest("x"), http.StatusBadRequest, "bad_request"},
		{Unauthorized("x"), http.StatusUnauthorized, "unauthorized"},
		{Forbidden("x"), http.StatusForbidden, "forbidden"},
		{NotFound("x"), http.StatusNotFound, "not_found"},
		{Conflict("x"), http.StatusConflict, "conflict"},
		{TooLarge("x"), http.StatusRequestEntityTooLarge, "too_large"},
		{UnsupportedMedia("x"), http.StatusUnsupportedMediaType, "unsupported_media_type"},
		{Invalid(map[string]string{"a": "b"}), http.StatusUnprocessableEntity, "validation"},
		{TooMany("x"), http.StatusTooManyRequests, "rate_limited"},
	} {
		if c.err.Status != c.status || c.err.Code != c.code {
			t.Errorf("%s: got %d %s", c.code, c.err.Status, c.err.Code)
		}
	}
}

// An output that cannot be written as JSON is logged and answered as the
// plain 500, never as a 200 with an empty body.
func TestWriteJSONCannotEncode(t *testing.T) {
	logs := captureLog(t)
	type Reading struct {
		Value float64 `json:"value"`
	}
	app := New(Config{})
	Get(app, "/reading", func(ctx context.Context, in None) (Reading, error) { return Reading{math.NaN()}, nil })
	w := send(t, mustRegister(t, app), "GET", "/api/reading", "")
	if code, msg, _ := apiError(t, w); w.Code != 500 || code != "internal" || msg != "Something went wrong on the server; its log has the details." {
		t.Fatalf("status %d, code %q, message %q", w.Code, code, msg)
	}
	if !strings.Contains(logs.String(), "NaN") || !strings.Contains(logs.String(), "level=ERROR") {
		t.Fatalf("log %q", logs.String())
	}
	w = httptest.NewRecorder()
	WriteJSON(w, 200, map[string]any{"c": make(chan int)})
	if w.Code != 500 || !strings.Contains(w.Body.String(), `"internal"`) {
		t.Fatalf("a channel: status %d, body %q", w.Code, w.Body.String())
	}
}

// An *Error whose status is not an error status is answered as the plain
// 500, and logged.
func TestWriteErrorStatusOutOfRange(t *testing.T) {
	for _, status := range []int{0, 200, 302, 399, 600} {
		logs := captureLog(t)
		w := writeErr(t, &Error{Status: status, Code: "odd", Message: "Odd."})
		if code, _, _ := apiError(t, w); w.Code != 500 || code != "internal" || !strings.Contains(logs.String(), "level=ERROR") {
			t.Errorf("status %d: answered %d %q, log %q", status, w.Code, code, logs.String())
		}
	}
	for _, status := range []int{400, 599} {
		if w := writeErr(t, &Error{Status: status, Code: "edge", Message: "Edge."}); w.Code != status {
			t.Errorf("status %d: answered %d", status, w.Code)
		}
	}
}

// The 500 body for an unencodable value is shared: writing it must never
// write into it. Run under -race.
func TestWriteJSONCannotEncodeConcurrently(t *testing.T) {
	captureLog(t)
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			for range 20 {
				w := httptest.NewRecorder()
				WriteJSON(w, 200, math.NaN())
				if w.Code != 500 || !strings.HasSuffix(w.Body.String(), "}\n") {
					t.Errorf("status %d, body %q", w.Code, w.Body.String())
				}
			}
		})
	}
	wg.Wait()
}
