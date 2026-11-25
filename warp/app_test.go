package warp

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"
)

// testMod is a module that logs each step of its life and can be made to
// fail any of them.
type testMod struct {
	name                                       string
	log                                        *[]string
	registerErr, migrateErr, startErr, stopErr error
	register                                   func(app *App)
}

// logMu guards the logs testMod and the tests' handlers write to.
var logMu sync.Mutex

func note(log *[]string, entry string) {
	logMu.Lock()
	defer logMu.Unlock()
	*log = append(*log, entry)
}

func notes(log *[]string) []string {
	logMu.Lock()
	defer logMu.Unlock()
	return slices.Clone(*log)
}

func mod(name string, log *[]string) *testMod { return &testMod{name: name, log: log} }

func (m *testMod) Name() string { return m.name }

func (m *testMod) Register(app *App) error {
	note(m.log, "register:"+m.name)
	if m.register != nil {
		m.register(app)
	}
	return m.registerErr
}

func (m *testMod) Migrate(ctx context.Context) error {
	note(m.log, "migrate:"+m.name)
	return m.migrateErr
}

func (m *testMod) Start(ctx context.Context) error {
	note(m.log, "start:"+m.name)
	return m.startErr
}

func (m *testMod) Stop(ctx context.Context) error {
	note(m.log, "stop:"+m.name)
	if _, ok := ctx.Deadline(); !ok {
		return errors.New("Stop got a context with no deadline")
	}
	return m.stopErr
}

// captureLog sends slog's and log's output to a buffer for the rest of the test.
func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	old, oldOut, oldFlags := slog.Default(), log.Writer(), log.Flags()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	t.Cleanup(func() {
		slog.SetDefault(old)
		log.SetOutput(oldOut)
		log.SetFlags(oldFlags)
	})
	return &buf
}

func serve(t *testing.T, h http.Handler, method, path string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(method, path, nil))
	return w
}

func text(s string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, s) })
}

func errText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func TestNewCarriesConfigMistakes(t *testing.T) {
	app := New(ConfigFrom(map[string]string{"PORT": "abc"}))
	want := `PORT is "abc"; set it to a number from 1 to 65535 in .env or the environment.`
	if got := errText(app.Err()); got != want {
		t.Fatalf("Err() = %q, want %q", got, want)
	}
	app.Handle("x", text("x"))
	got := strings.Split(errText(app.Err()), "\n")
	if len(got) != 2 || got[0] != want || !strings.HasPrefix(got[1], "warp: ") {
		t.Fatalf("Err() = %q", got)
	}
	ln := listenLocal(t)
	if err := app.Run(context.Background(), ln, nil); errText(err) != errText(app.Err()) {
		t.Fatalf("Run = %v", err)
	}
	if !closed(ln.Addr().String()) {
		t.Fatal("the listener is still open")
	}
}

func TestUseRegistersInOrder(t *testing.T) {
	var log []string
	app := New(Config{})
	app.Use(mod("a", &log), mod("b", &log))
	app.Use(mod("c", &log))
	if want := []string{"register:a", "register:b", "register:c"}; !reflect.DeepEqual(log, want) {
		t.Fatalf("log %v, want %v", log, want)
	}
	if err := app.Err(); err != nil {
		t.Fatal(err)
	}
}

func TestDuplicateModule(t *testing.T) {
	var log []string
	app := New(Config{})
	app.Use(mod("auth", &log), mod("auth", &log))
	if want := `warp: two modules are named "auth"; give one another name.`; errText(app.Err()) != want {
		t.Fatalf("Err() = %q, want %q", errText(app.Err()), want)
	}
	if want := []string{"register:auth"}; !reflect.DeepEqual(log, want) {
		t.Fatalf("log %v, want %v", log, want)
	}
}

func TestRegisterError(t *testing.T) {
	var log []string
	m := mod("auth", &log)
	m.registerErr = errors.New("AUTH_SECRET is not set; set it in .env.")
	app := New(Config{})
	app.Use(m)
	want := `warp: module "auth" could not register: AUTH_SECRET is not set; set it in .env.`
	if errText(app.Err()) != want {
		t.Fatalf("Err() = %q, want %q", errText(app.Err()), want)
	}
	if !errors.Is(app.Err(), m.registerErr) {
		t.Fatal("Err() does not wrap the module's error")
	}
}

func TestHandleUnderAPI(t *testing.T) {
	app := New(Config{})
	app.Use(&testMod{name: "ping", log: new([]string), register: func(app *App) {
		app.Handle("GET /ping", text("pong"))
	}})
	if err := app.Err(); err != nil {
		t.Fatal(err)
	}
	if w := serve(t, app.Handler(), "GET", "/api/ping"); w.Code != 200 || w.Body.String() != "pong" {
		t.Fatalf("status %d, body %q", w.Code, w.Body.String())
	}
	if want := []Route{{Method: "GET", Path: "/api/ping"}}; !reflect.DeepEqual(app.Routes(), want) {
		t.Fatalf("Routes() = %+v, want %+v", app.Routes(), want)
	}
}

func TestHandleWithoutMethod(t *testing.T) {
	app := New(Config{})
	app.Handle("/any", text("any"))
	for _, method := range []string{"GET", "POST", "DELETE"} {
		if w := serve(t, app.Handler(), method, "/api/any"); w.Code != 200 || w.Body.String() != "any" {
			t.Fatalf("%s: status %d, body %q", method, w.Code, w.Body.String())
		}
	}
	if want := []Route{{Path: "/api/any"}}; !reflect.DeepEqual(app.Routes(), want) {
		t.Fatalf("Routes() = %+v, want %+v", app.Routes(), want)
	}
}

func TestHandleStatusIsRecorded(t *testing.T) {
	app := New(Config{})
	app.Handle("POST /jobs", text("queued"), Status(202))
	if w := serve(t, app.Handler(), "POST", "/api/jobs"); w.Code != 200 {
		t.Fatalf("status %d, want the handler's own 200", w.Code)
	}
	if want := []Route{{Method: "POST", Path: "/api/jobs", Status: 202}}; !reflect.DeepEqual(app.Routes(), want) {
		t.Fatalf("Routes() = %+v, want %+v", app.Routes(), want)
	}
}

func TestHandleRefusesAPIPrefix(t *testing.T) {
	app := New(Config{})
	app.Handle("GET /api/x", text("x"))
	if want := `warp: route "/api/x" already gets /api in front; write it as "/x".`; errText(app.Err()) != want {
		t.Fatalf("Err() = %q, want %q", errText(app.Err()), want)
	}
	if len(app.Routes()) != 0 {
		t.Fatalf("Routes() = %+v", app.Routes())
	}
}

func TestHandleRefusesPathWithoutSlash(t *testing.T) {
	app := New(Config{})
	app.Handle("GET x", text("x"))
	if want := `warp: route "GET x" has a path that does not start with /; write it as "GET /x".`; errText(app.Err()) != want {
		t.Fatalf("Err() = %q, want %q", errText(app.Err()), want)
	}
}

func TestRouteConflict(t *testing.T) {
	app := New(Config{})
	app.Handle("GET /x", text("1"))
	app.Handle("GET /x", text("2"))
	err := errText(app.Err())
	if !strings.HasPrefix(err, `warp: route "GET /x" cannot be registered: `) || !strings.Contains(err, "conflicts") ||
		strings.Contains(err, "\n") || strings.Contains(err, "registered at") {
		t.Fatalf("Err() = %q", err)
	}
	app.Handle("GET /{bad", text("3"))
	if lines := strings.Split(errText(app.Err()), "\n"); len(lines) != 2 || !strings.HasPrefix(lines[1], `warp: route "GET /{bad" cannot be registered: `) {
		t.Fatalf("Err() = %q", lines)
	}
	if want := []Route{{Method: "GET", Path: "/api/x"}}; !reflect.DeepEqual(app.Routes(), want) {
		t.Fatalf("Routes() = %+v, want %+v", app.Routes(), want)
	}
}

func TestNotFoundShape(t *testing.T) {
	app := New(Config{})
	app.Handle("GET /hello", text("hi"))
	w := serve(t, app.Handler(), "GET", "/api/x")
	if w.Code != 404 || w.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("status %d, content-type %q", w.Code, w.Header().Get("Content-Type"))
	}
	want := decode(t, `{"error":{"code":"not_found","message":"Nothing is at /api/x."}}`)
	if got := decode(t, w.Body.String()); !reflect.DeepEqual(got, want) {
		t.Fatalf("body %v", got)
	}
}

func TestMethodNotAllowedShape(t *testing.T) {
	app := New(Config{})
	app.Handle("GET /hello", text("hi"))
	w := serve(t, app.Handler(), "POST", "/api/hello")
	if w.Code != 405 || w.Header().Get("Allow") != "GET, HEAD" {
		t.Fatalf("status %d, Allow %q", w.Code, w.Header().Get("Allow"))
	}
	want := decode(t, `{"error":{"code":"method_not_allowed","message":"/api/hello does not take POST; use GET."}}`)
	if got := decode(t, w.Body.String()); !reflect.DeepEqual(got, want) {
		t.Fatalf("body %v", got)
	}

	app.Handle("POST /hello", text("posted"))
	app.Handle("PATCH /hello", text("patched"))
	w = serve(t, app.Handler(), "DELETE", "/api/hello")
	if w.Code != 405 || w.Header().Get("Allow") != "GET, HEAD, PATCH, POST" {
		t.Fatalf("status %d, Allow %q", w.Code, w.Header().Get("Allow"))
	}
	if got := decode(t, w.Body.String())["error"].(map[string]any)["message"]; got != "/api/hello does not take DELETE; use GET, PATCH or POST." {
		t.Fatalf("message %q", got)
	}
}

func TestAllowedList(t *testing.T) {
	for allow, want := range map[string]string{
		"GET, HEAD":              "GET",
		"GET, HEAD, POST":        "GET or POST",
		"GET, HEAD, POST, PATCH": "GET, POST or PATCH",
		"POST":                   "POST",
		"HEAD, OPTIONS":          "HEAD or OPTIONS",
	} {
		if got := allowedList(allow); got != want {
			t.Errorf("allowedList(%q) = %q, want %q", allow, got, want)
		}
	}
}

func TestRedirectPassesThrough(t *testing.T) {
	app := New(Config{})
	app.Handle("GET /files/", text("files"))
	if w := serve(t, app.Handler(), "GET", "/api/files"); w.Code != http.StatusTemporaryRedirect || w.Header().Get("Location") != "/api/files/" {
		t.Fatalf("status %d, Location %q", w.Code, w.Header().Get("Location"))
	}
}

func TestPathValues(t *testing.T) {
	app := New(Config{})
	app.Handle("GET /items/{id}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, r.PathValue("id"))
	}))
	if w := serve(t, app.Handler(), "GET", "/api/items/7"); w.Body.String() != "7" {
		t.Fatalf("body %q", w.Body.String())
	}
}

func TestPanicIs500(t *testing.T) {
	logs := captureLog(t)
	app := New(Config{})
	app.Handle("GET /boom", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("db password hunter2")
	}))
	app.Handle("GET /ok", text("ok"))
	h := app.Handler()
	w := serve(t, h, "GET", "/api/boom")
	if w.Code != 500 || strings.Contains(w.Body.String(), "hunter2") {
		t.Fatalf("status %d, body %q", w.Code, w.Body.String())
	}
	if code := decode(t, w.Body.String())["error"].(map[string]any)["code"]; code != "internal" {
		t.Fatalf("code %v", code)
	}
	if !strings.Contains(logs.String(), "db password hunter2") || !strings.Contains(logs.String(), "goroutine") {
		t.Fatalf("log has no panic value and stack: %q", logs.String())
	}
	if w := serve(t, h, "GET", "/api/ok"); w.Code != 200 {
		t.Fatalf("second request: status %d", w.Code)
	}
}

func TestPanicAbortHandlerIsRepanicked(t *testing.T) {
	app := New(Config{})
	app.Handle("GET /abort", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic(http.ErrAbortHandler)
	}))
	defer func() {
		if v := recover(); v != http.ErrAbortHandler {
			t.Fatalf("recovered %v, want http.ErrAbortHandler", v)
		}
	}()
	serve(t, app.Handler(), "GET", "/api/abort")
}

func TestPanicInWrapIs500(t *testing.T) {
	captureLog(t)
	app := New(Config{})
	app.Wrap(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { panic("wrap") })
	})
	app.Handle("GET /ok", text("ok"))
	if w := serve(t, app.Handler(), "GET", "/api/ok"); w.Code != 500 {
		t.Fatalf("status %d", w.Code)
	}
}

func TestWrapOrder(t *testing.T) {
	var order []string
	mark := func(name string) func(http.Handler) http.Handler {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				order = append(order, name)
				next.ServeHTTP(w, r)
			})
		}
	}
	app := New(Config{})
	app.Wrap(mark("wrap 1"))
	app.Wrap(mark("wrap 2"))
	app.Handle("GET /x", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		order = append(order, "handler")
	}), With(mark("with 1")), With(mark("with 2")))
	serve(t, app.Handler(), "GET", "/api/x")
	if want := []string{"wrap 1", "wrap 2", "with 1", "with 2", "handler"}; !reflect.DeepEqual(order, want) {
		t.Fatalf("order %v, want %v", order, want)
	}
}
