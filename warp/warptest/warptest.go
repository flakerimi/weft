// Package warptest gives a test an app and a client that keeps cookies, for
// testing modules built on warp. The client calls the app's handler
// directly: there is no listener.
package warptest

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/flakerimi/weft/warp"
)

// origin is the site the client pretends to be on: the request URL's host
// and the Origin header, so the app's cross-site check lets it through.
const origin = "http://warptest.local"

// New builds an app from vars and modules, fails the test when the app has
// registration mistakes or does not start, and stops the app when the test
// ends.
func New(t testing.TB, vars map[string]string, modules ...warp.Module) *Client {
	t.Helper()
	app := warp.New(warp.ConfigFrom(vars))
	app.Use(modules...)
	if err := app.Err(); err != nil {
		t.Fatalf("warptest: the app has registration mistakes, fix them: %v", err)
	}
	if err := app.Start(context.Background()); err != nil {
		t.Fatalf("warptest: the app did not start, fix the module: %v", err)
	}
	t.Cleanup(func() {
		if err := app.Stop(context.Background()); err != nil {
			t.Errorf("warptest: the app did not stop cleanly: %v", err)
		}
	})
	return &Client{App: app, t: t, jar: map[string]*http.Cookie{}}
}

// Client sends requests to an app and keeps the cookies it is given.
type Client struct {
	App *warp.App

	t   testing.TB
	mu  sync.Mutex
	jar map[string]*http.Cookie
}

// Get sends a GET to path, given as the app sees it, with /api in front.
func (c *Client) Get(path string) *Response {
	c.t.Helper()
	return c.send(http.MethodGet, path, nil, false)
}

// Post sends a POST to path. A nil body sends none; any other body is sent
// as JSON, a []byte or string as it is.
func (c *Client) Post(path string, body any) *Response {
	c.t.Helper()
	return c.send(http.MethodPost, path, body, true)
}

// Patch sends a PATCH to path, with a body as Post does.
func (c *Client) Patch(path string, body any) *Response {
	c.t.Helper()
	return c.send(http.MethodPatch, path, body, true)
}

// Delete sends a DELETE to path.
func (c *Client) Delete(path string) *Response {
	c.t.Helper()
	return c.send(http.MethodDelete, path, nil, false)
}

// Fork returns a client for the same app with no cookies.
func (c *Client) Fork() *Client {
	return &Client{App: c.App, t: c.t, jar: map[string]*http.Cookie{}}
}

// send builds the request and calls Do. The cookie jar is keyed by name only:
// it does not model a browser's path, domain or expiry rules (beyond deleting
// a cookie with Max-Age below zero or an Expires in the past), so do not rely
// on it for them.
func (c *Client) send(method, path string, body any, hasBody bool) *Response {
	c.t.Helper()
	var rd io.Reader
	if hasBody && body != nil {
		switch b := body.(type) {
		case []byte:
			rd = bytes.NewReader(b)
		case string:
			rd = bytes.NewReader([]byte(b))
		default:
			enc, err := json.Marshal(body)
			if err != nil {
				c.t.Fatalf("warptest: the body of %s %s cannot be encoded as JSON, send a value that can be: %v", method, path, err)
			}
			rd = bytes.NewReader(enc)
		}
	}
	r := httptest.NewRequest(method, origin+path, rd)
	r.Header.Set("Origin", origin)
	if rd != nil {
		r.Header.Set("Content-Type", "application/json")
	}
	return c.Do(r)
}

// Do sends r as it is, adding only the stored cookies, and stores the
// cookies the answer sets. It leaves Origin alone, so a test can send a
// cross-site request.
func (c *Client) Do(r *http.Request) *Response {
	c.t.Helper()
	c.mu.Lock()
	for _, ck := range c.jar {
		r.AddCookie(&http.Cookie{Name: ck.Name, Value: ck.Value})
	}
	c.mu.Unlock()

	rec := httptest.NewRecorder()
	c.App.Handler().ServeHTTP(rec, r)
	res := rec.Result()
	defer res.Body.Close()
	data, _ := io.ReadAll(res.Body)

	c.mu.Lock()
	for _, ck := range res.Cookies() {
		if ck.MaxAge < 0 || (!ck.Expires.IsZero() && ck.Expires.Before(time.Now())) {
			delete(c.jar, ck.Name)
			continue
		}
		c.jar[ck.Name] = ck
	}
	c.mu.Unlock()

	return &Response{Status: res.StatusCode, Header: res.Header, Body: data, t: c.t, method: r.Method, path: r.URL.Path}
}

// Response is an app's answer.
type Response struct {
	Status int
	Header http.Header
	Body   []byte

	t      testing.TB
	method string
	path   string
}

// JSON decodes the body into v and fails the test when the status is not 2xx
// or the body does not decode.
func (r *Response) JSON(v any) {
	r.t.Helper()
	if r.Status < 200 || r.Status > 299 {
		r.t.Fatalf("warptest: %s %s answered %d, not a 2xx, so its body is not the value to decode, use Error() for a failing answer: %s", r.method, r.path, r.Status, r.Body)
	}
	if err := json.Unmarshal(r.Body, v); err != nil {
		r.t.Fatalf("warptest: %s %s answered %d with a body that is not the JSON expected (%v), check the route: %s", r.method, r.path, r.Status, err, r.Body)
	}
}

// Error returns the error the answer carries, with Status set from the
// response, or nil for a 2xx. It fails the test when another answer is not
// in the error shape.
func (r *Response) Error() *warp.Error {
	r.t.Helper()
	if r.Status >= 200 && r.Status < 300 {
		return nil
	}
	var b struct {
		Error *warp.Error `json:"error"`
	}
	if err := json.Unmarshal(r.Body, &b); err != nil || b.Error == nil || b.Error.Code == "" {
		r.t.Fatalf("warptest: %s %s answered %d with a body that is not the error shape, make the route answer with a warp.Error: %s", r.method, r.path, r.Status, r.Body)
	}
	b.Error.Status = r.Status
	return b.Error
}

// Want fails the test, printing the body, when the status is not status, and
// returns the response otherwise.
func (r *Response) Want(status int) *Response {
	r.t.Helper()
	if r.Status != status {
		r.t.Fatalf("warptest: %s %s answered %d, wanted %d, body: %s", r.method, r.path, r.Status, status, r.Body)
	}
	return r
}
