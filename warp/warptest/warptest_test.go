package warptest

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/flakerimi/weft/warp"
)

type hello struct {
	Name string `json:"name"`
}

type mod struct {
	started, stopped *int
	failRegister     bool
	failStart        bool
}

func (m mod) Name() string { return "mod" }

func (m mod) Register(app *warp.App) error {
	if m.failRegister {
		return errors.New("mod: the route is wrong, fix it.")
	}
	warp.Get(app, "/hello", func(ctx context.Context, _ struct{}) (hello, error) {
		r := warp.Request(ctx)
		name := "nobody"
		if c, err := r.Cookie("who"); err == nil {
			name = c.Value
		}
		return hello{name}, nil
	})
	warp.Post(app, "/login", func(ctx context.Context, _ struct{}) (hello, error) {
		h := warp.ResponseHeader(ctx)
		h.Add("Set-Cookie", "who=ada; Path=/; Secure")
		return hello{"ada"}, nil
	})
	warp.Post(app, "/logout", func(ctx context.Context, _ struct{}) (hello, error) {
		warp.ResponseHeader(ctx).Add("Set-Cookie", "who=; Path=/; Max-Age=0")
		return hello{}, nil
	})
	warp.Get(app, "/missing", func(ctx context.Context, _ struct{}) (hello, error) {
		return hello{}, warp.NotFound("No such thing.")
	})
	app.Handle("POST /echo", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b := make([]byte, 100)
		n, _ := r.Body.Read(b)
		w.Write([]byte(r.Header.Get("Content-Type") + "|" + string(b[:n])))
	}))
	app.Handle("POST /origin", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(r.Header.Get("Origin") + "|" + r.Host))
	}))
	app.Handle("GET /raw", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		w.Write([]byte(r.URL.Query().Get("body")))
	}))
	app.Handle("GET /plain", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json"))
	}))
	app.Handle("GET /expired", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "who", Value: "", Path: "/", Expires: time.Now().Add(-time.Hour)})
	}))
	app.Handle("PATCH /verb", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(r.Method))
	}))
	app.Handle("DELETE /verb", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(r.Method))
	}))
	app.Handle("POST /bare", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("[" + r.Header.Get("Content-Type") + "]"))
	}))
	return nil
}

func (m mod) Start(ctx context.Context) error {
	if m.failStart {
		return errors.New("mod: no database, check DATABASE_URL.")
	}
	if m.started != nil {
		*m.started++
	}
	return nil
}

func (m mod) Stop(ctx context.Context) error {
	if m.stopped != nil {
		*m.stopped++
	}
	return nil
}

func TestClientKeepsCookies(t *testing.T) {
	c := New(t, nil, mod{})
	var h hello
	c.Get("/api/hello").Want(200).JSON(&h)
	if h.Name != "nobody" {
		t.Fatalf("name = %q", h.Name)
	}
	c.Post("/api/login", nil).Want(201)
	c.Get("/api/hello").Want(200).JSON(&h)
	if h.Name != "ada" {
		t.Fatalf("a Secure cookie must be kept and sent, name = %q", h.Name)
	}
	c.Post("/api/logout", nil).Want(201)
	c.Get("/api/hello").Want(200).JSON(&h)
	if h.Name != "nobody" {
		t.Fatalf("Max-Age=0 must delete the cookie, name = %q", h.Name)
	}
}

func TestForkHasNone(t *testing.T) {
	c := New(t, nil, mod{})
	c.Post("/api/login", nil).Want(201)
	f := c.Fork()
	var h hello
	f.Get("/api/hello").Want(200).JSON(&h)
	if h.Name != "nobody" || f.App != c.App {
		t.Fatalf("fork: name %q, same app %v", h.Name, f.App == c.App)
	}
	c.Get("/api/hello").Want(200).JSON(&h)
	if h.Name != "ada" {
		t.Fatalf("the original lost its cookie: %q", h.Name)
	}
}

func TestBodies(t *testing.T) {
	c := New(t, nil, mod{})
	if got := string(c.Post("/api/bare", nil).Want(200).Body); got != "[]" {
		t.Fatalf("nil body: %q", got)
	}
	if got := string(c.Post("/api/echo", map[string]int{"a": 1}).Want(200).Body); got != `application/json|{"a":1}` {
		t.Fatalf("map body: %q", got)
	}
	if got := string(c.Post("/api/echo", "raw").Want(200).Body); got != "application/json|raw" {
		t.Fatalf("string body: %q", got)
	}
	if got := string(c.Post("/api/echo", []byte("by")).Want(200).Body); got != "application/json|by" {
		t.Fatalf("bytes body: %q", got)
	}
}

func TestSendsSameOrigin(t *testing.T) {
	c := New(t, nil, mod{})
	if got := string(c.Post("/api/origin", nil).Want(200).Body); got != "http://warptest.local|warptest.local" {
		t.Fatalf("origin|host = %q", got)
	}
}

func TestJSONAndError(t *testing.T) {
	c := New(t, nil, mod{})
	if err := c.Get("/api/hello").Error(); err != nil {
		t.Fatalf("2xx error = %v", err)
	}
	e := c.Get("/api/missing").Error()
	if e == nil || e.Status != 404 || e.Code != "not_found" || e.Message != "No such thing." {
		t.Fatalf("error = %+v", e)
	}
}

func TestCrossSiteThroughDoIsRefused(t *testing.T) {
	c := New(t, nil, mod{})
	r := httptest.NewRequest("POST", "http://warptest.local/api/login", nil)
	r.Header.Set("Origin", "http://evil.example")
	c.Do(r).Want(403)
	c.Post("/api/login", nil).Want(201)
}

// fake is a testing.TB whose failures are recorded and stop the goroutine.
type fake struct {
	testing.TB
	msg      string
	failed   bool
	cleanups []func()
}

func (f *fake) Helper() {}
func (f *fake) Fatalf(format string, args ...any) {
	f.failed = true
	f.msg = sprintf(format, args...)
	runtime.Goexit()
}
func (f *fake) Errorf(format string, args ...any) { f.failed = true; f.msg = sprintf(format, args...) }
func (f *fake) Cleanup(fn func())                 { f.cleanups = append(f.cleanups, fn) }

func sprintf(format string, args ...any) string {
	return strings.TrimSpace(strings.ReplaceAll(fmt.Sprintf(format, args...), "\n", " "))
}

// run runs fn on its own goroutine so a Fatalf can end it.
func run(f *fake, fn func()) {
	done := make(chan struct{})
	go func() { defer close(done); fn() }()
	<-done
}

func TestWantFails(t *testing.T) {
	c := New(t, nil, mod{})
	f := &fake{TB: t}
	c.t = f
	run(f, func() { c.Get("/api/missing").Want(200) })
	for _, s := range []string{"GET", "/api/missing", "404", "200", "No such thing."} {
		if !f.failed || !strings.Contains(f.msg, s) {
			t.Fatalf("message %q lacks %q", f.msg, s)
		}
	}
}

func TestNewFailsOnMistakes(t *testing.T) {
	f := &fake{TB: t}
	run(f, func() { New(f, nil, mod{failRegister: true}) })
	if !f.failed || !strings.Contains(f.msg, "the route is wrong") {
		t.Fatalf("register mistake: %q", f.msg)
	}
	f = &fake{TB: t}
	run(f, func() { New(f, nil, mod{failStart: true}) })
	if !f.failed || !strings.Contains(f.msg, "DATABASE_URL") {
		t.Fatalf("start mistake: %q", f.msg)
	}
}

func TestNewStartsAndStops(t *testing.T) {
	var started, stopped int
	f := &fake{TB: t}
	New(f, nil, mod{started: &started, stopped: &stopped})
	if started != 1 || stopped != 0 {
		t.Fatalf("started %d stopped %d", started, stopped)
	}
	for _, fn := range f.cleanups {
		fn()
	}
	if stopped != 1 {
		t.Fatalf("stopped %d after cleanup", stopped)
	}
}

// fails runs fn against a fake TB and returns the failure message.
func fails(t *testing.T, fn func(c *Client)) string {
	t.Helper()
	c := New(t, nil, mod{})
	f := &fake{TB: t}
	c.t = f
	run(f, func() { fn(c) })
	if !f.failed {
		t.Fatal("the test did not fail")
	}
	return f.msg
}

func TestJSONFailsOnNonJSON(t *testing.T) {
	msg := fails(t, func(c *Client) {
		var h hello
		c.Get("/api/plain").JSON(&h)
	})
	if !strings.Contains(msg, "not json") {
		t.Fatalf("message %q", msg)
	}
}

func TestJSONFailsOnFailingStatus(t *testing.T) {
	msg := fails(t, func(c *Client) {
		var h hello
		c.Get("/api/missing").JSON(&h)
	})
	for _, s := range []string{"404", "No such thing.", "Error()"} {
		if !strings.Contains(msg, s) {
			t.Fatalf("message %q lacks %q", msg, s)
		}
	}
}

func TestErrorFailsOnOtherShapes(t *testing.T) {
	for _, body := range []string{"not json", "{}", `{"error":null}`, `{"error":{}}`} {
		msg := fails(t, func(c *Client) { c.Get("/api/raw?body=" + url.QueryEscape(body)).Error() })
		if !strings.Contains(msg, "error shape") {
			t.Fatalf("body %q: message %q", body, msg)
		}
	}
}

func TestErrorIsNilFor2xx(t *testing.T) {
	c := New(t, nil, mod{})
	if e := c.Get("/api/plain").Error(); e != nil {
		t.Fatalf("error = %v", e)
	}
}

func TestPatchAndDelete(t *testing.T) {
	c := New(t, nil, mod{})
	if got := string(c.Patch("/api/verb", nil).Want(200).Body); got != "PATCH" {
		t.Fatalf("patch: %q", got)
	}
	if got := string(c.Delete("/api/verb").Want(200).Body); got != "DELETE" {
		t.Fatalf("delete: %q", got)
	}
}

func TestPastExpiresDeletesCookie(t *testing.T) {
	c := New(t, nil, mod{})
	var h hello
	c.Post("/api/login", nil).Want(201)
	c.Get("/api/expired").Want(200)
	c.Get("/api/hello").Want(200).JSON(&h)
	if h.Name != "nobody" {
		t.Fatalf("a past Expires must delete the cookie, name = %q", h.Name)
	}
}
