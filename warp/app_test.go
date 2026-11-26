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
	if want := []Route{{Module: "ping", Name: "ping", Method: "GET", Path: "/api/ping"}}; !reflect.DeepEqual(app.Routes(), want) {
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
	if want := []Route{{Name: "any", Path: "/api/any"}}; !reflect.DeepEqual(app.Routes(), want) {
		t.Fatalf("Routes() = %+v, want %+v", app.Routes(), want)
	}
}

func TestHandleStatusIsRecorded(t *testing.T) {
	app := New(Config{})
	app.Handle("POST /jobs", text("queued"), Status(202))
	if w := serve(t, app.Handler(), "POST", "/api/jobs"); w.Code != 200 {
		t.Fatalf("status %d, want the handler's own 200", w.Code)
	}
	if want := []Route{{Name: "post_jobs", Method: "POST", Path: "/api/jobs", Status: 202}}; !reflect.DeepEqual(app.Routes(), want) {
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
	if want := `warp: route "GET /x" conflicts with GET /x, registered before it; give one of them another path.`; err != want {
		t.Fatalf("Err() = %q, want %q", err, want)
	}
	other := New(Config{})
	other.Handle("GET /a/{x}", text("1"))
	other.Handle("GET /{y}/b", text("2"))
	if want := `warp: route "GET /{y}/b" conflicts with GET /a/{x}, registered before it; give one of them another path.`; errText(other.Err()) != want {
		t.Fatalf("Err() = %q, want %q", errText(other.Err()), want)
	}
	app.Handle("GET /{bad", text("3"))
	if lines := strings.Split(errText(app.Err()), "\n"); len(lines) != 2 || !strings.HasPrefix(lines[1], `warp: route "GET /{bad" cannot be registered: `) {
		t.Fatalf("Err() = %q", lines)
	}
	if want := []Route{{Name: "x", Method: "GET", Path: "/api/x"}}; !reflect.DeepEqual(app.Routes(), want) {
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

// inModule is a module named name whose Register runs register.
func inModule(name string, register func(app *App)) *testMod {
	return &testMod{name: name, log: new([]string), register: register}
}

func TestRouteModuleAndName(t *testing.T) {
	app := New(Config{})
	app.Use(inModule("greet", func(app *App) {
		Get(app, "/hello", func(context.Context, None) (Item, error) { return Item{}, nil })
		app.Use(inModule("inner", func(app *App) {
			app.Handle("GET /inner", text("inner"))
		}))
		Post(app, "/hello/greet", func(context.Context, None) (Item, error) { return Item{}, nil })
	}))
	app.Handle("GET /", text("index"))
	app.Handle("DELETE /line-items/{id}", text("deleted"))
	app.Handle("GET /Files/{rest...}", text("files"))
	app.Handle("PUT /{id}", text("put"))
	app.Handle("POST /{$}", text("post"))
	if err := app.Err(); err != nil {
		t.Fatal(err)
	}
	type named struct{ Module, Name, Method, Path string }
	var got []named
	for _, r := range app.Routes() {
		got = append(got, named{r.Module, r.Name, r.Method, r.Path})
	}
	want := []named{
		{"greet", "hello", "GET", "/api/hello"},
		{"inner", "inner", "GET", "/api/inner"},
		{"greet", "post_hello_greet", "POST", "/api/hello/greet"},
		{"", "index", "GET", "/api/"},
		{"", "delete_line_items", "DELETE", "/api/line-items/{id}"},
		{"", "files", "GET", "/api/Files/{rest...}"},
		{"", "put", "PUT", "/api/{id}"},
		{"", "post", "POST", "/api/{$}"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Routes() =\n%v\nwant\n%v", got, want)
	}
	other := New(Config{})
	other.Handle("/any/Thing", text("any"))
	if r := other.Routes(); len(r) != 1 || r[0].Name != "any_thing" {
		t.Fatalf("Routes() = %+v", r)
	}
}

func TestNameOverridesAndClashes(t *testing.T) {
	app := New(Config{})
	app.Use(inModule("products", func(app *App) {
		Get(app, "/products", func(context.Context, None) ([]Item, error) { return nil, nil }, Name("list"))
		app.Handle("GET /products/all", text("all"), Name("list"))
		Get(app, "/products/{id}", func(context.Context, None) (Item, error) { return Item{}, nil })
		Get(app, "/products/{id}/{$}", func(context.Context, None) (Item, error) { return Item{}, nil })
		app.Handle("GET /empty", text("x"), Name(""))
	}))
	app.Use(inModule("orders", func(app *App) {
		app.Handle("GET /orders", text("x"), Name("list"))
	}))
	app.Use(inModule("dup", func(app *App) {
		app.Handle("GET /dup", text("1"))
		app.Handle("GET /dup", text("2"))
	}))
	// Outside a module names are not checked.
	app.Handle("GET /a", text("a"))
	app.Handle("GET /a/{id}", text("a"))
	want := strings.Join([]string{
		`warp: module "products" has two routes named "list" (GET /products and GET /products/all); give them different names.`,
		`warp: module "products" has two routes named "products" (GET /products/{id} and GET /products/{id}/{$}); give one a warp.Name.`,
		`warp: Name("") is empty; give the route a name.`,
	}, "\n")
	lines := strings.Split(errText(app.Err()), "\n")
	if len(lines) != 4 || strings.Join(lines[:3], "\n") != want ||
		lines[3] != `warp: route "GET /dup" conflicts with GET /dup, registered before it; give one of them another path.` {
		t.Fatalf("Err() =\n%s\nwant\n%s\nand the mux's conflict", errText(app.Err()), want)
	}
	var got []string
	for _, r := range app.Routes() {
		got = append(got, r.Module+"."+r.Name+" "+r.Path)
	}
	if want := []string{"products.list /api/products", "products.products /api/products/{id}", "orders.list /api/orders", "dup.dup /api/dup", ".a /api/a", ".a /api/a/{id}"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Routes() = %v, want %v", got, want)
	}
	if w := serve(t, app.Handler(), "GET", "/api/products/all"); w.Body.String() == "all" {
		t.Fatal("a route whose name clashed was served")
	}
	if w := serve(t, app.Handler(), "GET", "/api/empty"); w.Code != 404 {
		t.Fatalf("a route named \"\" was served: status %d", w.Code)
	}
}

func TestOptionsApplyInOrder(t *testing.T) {
	var order []string
	mark := func(name string) func(http.Handler) http.Handler {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				order = append(order, name)
				next.ServeHTTP(w, r)
			})
		}
	}
	guard := Options(Permission("products:edit"), With(mark("guard")))
	app := New(Config{})
	app.Handle("POST /products", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		order = append(order, "handler")
	}), With(mark("first")), guard, Options(Name("one"), Name("two")), With(mark("last")))
	serve(t, mustRegister(t, app), "POST", "/api/products")
	if want := []string{"first", "guard", "last", "handler"}; !reflect.DeepEqual(order, want) {
		t.Fatalf("order %v, want %v", order, want)
	}
	r := app.Routes()[0]
	if r.Permission != "products:edit" || r.Name != "two" {
		t.Fatalf("Permission %q, Name %q", r.Permission, r.Name)
	}
}

type testDB struct{ name string }

func TestProvideAndNeed(t *testing.T) {
	t.Run("provided", func(t *testing.T) {
		app := New(Config{})
		var db *testDB
		var n int
		app.Use(
			inModule("data", func(app *App) { Provide(app, &testDB{"main"}); Provide(app, 7) }),
			inModule("products", func(app *App) { db = Need[*testDB](app, "data"); n = Need[int](app, "data") }),
		)
		if err := app.Err(); err != nil {
			t.Fatal(err)
		}
		if db == nil || db.name != "main" || n != 7 {
			t.Fatalf("Need = %v, %d", db, n)
		}
		if got := Need[*testDB](app, "data"); got != db {
			t.Fatalf("Need outside a module = %v", got)
		}
	})
	t.Run("missing", func(t *testing.T) {
		app := New(Config{})
		var db *testDB
		var n int
		app.Use(inModule("products", func(app *App) { db = Need[*testDB](app, "data") }))
		n = Need[int](app, "clock")
		want := strings.Join([]string{
			`warp: module "products" needs the "data" module; list data.Module() before it in app/modules.go.`,
			`warp: the app needs the "clock" module; list clock.Module() in app/modules.go.`,
		}, "\n")
		if got := errText(app.Err()); got != want {
			t.Fatalf("Err() =\n%s\nwant\n%s", got, want)
		}
		if db != nil || n != 0 {
			t.Fatalf("Need = %v, %d, want the zero values", db, n)
		}
	})
	t.Run("twice", func(t *testing.T) {
		app := New(Config{})
		app.Use(
			inModule("a", func(app *App) { Provide(app, &testDB{"a"}) }),
			inModule("b", func(app *App) { Provide(app, &testDB{"b"}) }),
		)
		if want := `warp: two modules provide a *warp.testDB; list only one of them.`; errText(app.Err()) != want {
			t.Fatalf("Err() = %q, want %q", errText(app.Err()), want)
		}
		if got := Need[*testDB](app, "a"); got.name != "a" {
			t.Fatalf("Need = %v, want the first", got)
		}
	})
	t.Run("a miss after start is logged", func(t *testing.T) {
		logs := captureLog(t)
		app := New(Config{})
		if err := app.Start(context.Background()); err != nil {
			t.Fatal(err)
		}
		if db := Need[*testDB](app, "data"); db != nil {
			t.Fatalf("Need = %v", db)
		}
		want := `warp: the app needs the \"data\" module; list data.Module() in app/modules.go.`
		if !strings.Contains(logs.String(), want) || !strings.Contains(errText(app.Err()), "needs the \"data\" module") {
			t.Fatalf("log %q, Err() %q", logs, app.Err())
		}
	})
	t.Run("after start", func(t *testing.T) {
		captureLog(t)
		app := New(Config{})
		if err := app.Start(context.Background()); err != nil {
			t.Fatal(err)
		}
		Provide(app, &testDB{"late"})
		if want := `warp: Provide was called after the app started; provide every value in a module's Register.`; errText(app.Err()) != want {
			t.Fatalf("Err() = %q, want %q", errText(app.Err()), want)
		}
		if err := app.Stop(context.Background()); err != nil {
			t.Fatal(err)
		}
	})
}

func TestMistake(t *testing.T) {
	app := New(Config{})
	app.Use(inModule("products", func(app *App) {
		app.Mistake("products: %s does not embed data.Record; embed it, since a store needs an id and two times.", "Product")
	}))
	app.Handle("x", text("x"))
	want := strings.Join([]string{
		`warp: products: Product does not embed data.Record; embed it, since a store needs an id and two times.`,
		`warp: route "x" has a path that does not start with /; write it as "/x".`,
	}, "\n")
	if got := errText(app.Err()); got != want {
		t.Fatalf("Err() =\n%s\nwant\n%s", got, want)
	}
	if err := app.Start(context.Background()); errText(err) != want {
		t.Fatalf("Start = %v, want the mistakes", err)
	}

	// A mistake refuses a description too, whatever the configuration.
	out := describeTo(t)
	app = New(ConfigFrom(map[string]string{"PORT": "nope"}))
	app.Mistake("a store's table is named %q; use lower-case letters, digits and _, such as order_items.", "Order Items")
	if err, want := app.Serve(nil), `warp: a store's table is named "Order Items"; use lower-case letters, digits and _, such as order_items.`; errText(err) != want {
		t.Fatalf("Serve = %q, want %q", errText(err), want)
	}
	if out.Len() != 0 {
		t.Fatalf("Serve printed %q", out)
	}

	// After start it is logged as well, since nothing reads Err then; Start
	// is the app's, not a describing run's.
	t.Setenv("WARP_DESCRIBE", "")
	logs := captureLog(t)
	app = New(Config{})
	if err := app.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	app.Mistake("late: say what to do.")
	if want := "warp: late: say what to do."; errText(app.Err()) != want || !strings.Contains(logs.String(), want) {
		t.Fatalf("Err() = %q, log %q", errText(app.Err()), logs)
	}
	if err := app.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
}

// A route option may say the route needs a module, as auth.Can does: an app
// that lists none of that name has a registration mistake, which weft gen's
// description reports too.
func TestRequiresAModule(t *testing.T) {
	app := New(Config{})
	Get(app, "/x", func(context.Context, None) (Item, error) { return Item{}, nil }, Requires("auth", "auth.Module()"))
	want := `warp: route "GET /x" needs the "auth" module; list auth.Module() in app/modules.go.`
	if errText(app.Err()) != want || errText(app.registrationErr()) != want {
		t.Fatalf("Err = %q, registrationErr = %q, want %q", errText(app.Err()), errText(app.registrationErr()), want)
	}
	app = New(Config{})
	var log []string
	app.Use(mod("auth", &log))
	Get(app, "/x", func(context.Context, None) (Item, error) { return Item{}, nil }, Requires("auth", "auth.Module()"))
	if err := app.Err(); err != nil {
		t.Fatalf("with the module: %v", err)
	}
}
