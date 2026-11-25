package warp

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

type Item struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// send serves one request with body, and header given as name, value pairs.
// A body is sent as application/json unless header gives a Content-Type; a
// header given as "" is left out.
func send(t *testing.T, h http.Handler, method, path, body string, header ...string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		r.Header.Set("Content-Type", "application/json")
	}
	for i := 0; i+1 < len(header); i += 2 {
		if header[i+1] == "" {
			r.Header.Del(header[i])
			continue
		}
		r.Header.Set(header[i], header[i+1])
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

// apiError reads an answer in the error shape.
func apiError(t *testing.T, w *httptest.ResponseRecorder) (code, message string, fields map[string]string) {
	t.Helper()
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("Content-Type %q, body %q", ct, w.Body.String())
	}
	e, ok := decode(t, w.Body.String())["error"].(map[string]any)
	if !ok {
		t.Fatalf("body %q is not in the error shape", w.Body.String())
	}
	code, _ = e["code"].(string)
	message, _ = e["message"].(string)
	if f, ok := e["fields"].(map[string]any); ok {
		fields = map[string]string{}
		for k, v := range f {
			fields[k], _ = v.(string)
		}
	}
	return code, message, fields
}

func mustRegister(t *testing.T, app *App) http.Handler {
	t.Helper()
	if err := app.Err(); err != nil {
		t.Fatal(err)
	}
	return app.Handler()
}

func TestGetBindsPathAndQuery(t *testing.T) {
	type In struct {
		ID   int      `path:"id"`
		Page int      `query:"page"`
		Tags []string `query:"tag"`
	}
	var got In
	app := New(Config{})
	Get(app, "/items/{id}", func(ctx context.Context, in In) (Item, error) {
		got = in
		return Item{ID: in.ID, Name: "lamp"}, nil
	})
	w := send(t, mustRegister(t, app), "GET", "/api/items/7?page=2&tag=a&tag=b", "")
	if w.Code != 200 || w.Body.String() != `{"id":7,"name":"lamp"}`+"\n" {
		t.Fatalf("status %d, body %q", w.Code, w.Body.String())
	}
	if want := (In{ID: 7, Page: 2, Tags: []string{"a", "b"}}); !reflect.DeepEqual(got, want) {
		t.Fatalf("input %+v, want %+v", got, want)
	}
}

func TestPostBindsBodyAndAnswers201(t *testing.T) {
	type In struct {
		Name  string `json:"name" warp:"required"`
		Price int    `json:"price"`
	}
	var got In
	app := New(Config{})
	Post(app, "/items", func(ctx context.Context, in In) (Item, error) {
		got = in
		return Item{ID: 1, Name: in.Name}, nil
	})
	w := send(t, mustRegister(t, app), "POST", "/api/items", `{"name":"lamp","price":12}`)
	if w.Code != 201 || w.Body.String() != `{"id":1,"name":"lamp"}`+"\n" {
		t.Fatalf("status %d, body %q", w.Code, w.Body.String())
	}
	if want := (In{Name: "lamp", Price: 12}); got != want {
		t.Fatalf("input %+v, want %+v", got, want)
	}
}

func TestPutPatchDeleteAnswer200(t *testing.T) {
	type Body struct {
		ID   int    `path:"id"`
		Name string `json:"name"`
	}
	type ByID struct {
		ID int `path:"id"`
	}
	var got []string
	app := New(Config{})
	Put(app, "/items/{id}", func(ctx context.Context, in Body) (Item, error) {
		got = append(got, "put "+in.Name)
		return Item{ID: in.ID, Name: in.Name}, nil
	})
	Patch(app, "/items/{id}", func(ctx context.Context, in Body) (Item, error) {
		got = append(got, "patch "+in.Name)
		return Item{ID: in.ID, Name: in.Name}, nil
	})
	Delete(app, "/items/{id}", func(ctx context.Context, in ByID) (Item, error) {
		got = append(got, "delete")
		return Item{ID: in.ID}, nil
	})
	h := mustRegister(t, app)
	for _, method := range []string{"PUT", "PATCH", "DELETE"} {
		if w := send(t, h, method, "/api/items/3", `{"name":"x"}`); w.Code != 200 {
			t.Fatalf("%s: status %d, body %q", method, w.Code, w.Body.String())
		}
	}
	if want := []string{"put x", "patch x", "delete"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("calls %v, want %v", got, want)
	}
}

func TestStatusOption(t *testing.T) {
	app := New(Config{})
	Post(app, "/search", func(ctx context.Context, in None) ([]Item, error) {
		return []Item{{ID: 1}}, nil
	}, Status(200))
	Get(app, "/jobs", func(ctx context.Context, in None) (Item, error) {
		return Item{}, nil
	}, Status(202))
	h := mustRegister(t, app)
	if w := send(t, h, "POST", "/api/search", ""); w.Code != 200 {
		t.Fatalf("POST: status %d", w.Code)
	}
	if w := send(t, h, "GET", "/api/jobs", ""); w.Code != 202 {
		t.Fatalf("GET: status %d", w.Code)
	}
}

func TestNoneOutIs204(t *testing.T) {
	type In struct {
		ID int `path:"id"`
	}
	app := New(Config{})
	Delete(app, "/items/{id}", func(ctx context.Context, in In) (None, error) {
		return None{}, nil
	})
	// None answers 204 whatever Status says.
	Post(app, "/items/{id}/archive", func(ctx context.Context, in In) (None, error) {
		return None{}, nil
	}, Status(200))
	h := mustRegister(t, app)
	for _, c := range []struct{ method, path string }{{"DELETE", "/api/items/1"}, {"POST", "/api/items/1/archive"}} {
		w := send(t, h, c.method, c.path, "")
		if w.Code != 204 || w.Body.Len() != 0 || w.Header().Get("Content-Type") != "" {
			t.Fatalf("%s: status %d, body %q, Content-Type %q", c.method, w.Code, w.Body.String(), w.Header().Get("Content-Type"))
		}
	}
}

func TestNoneInReadsNothing(t *testing.T) {
	ran := false
	app := New(Config{})
	Post(app, "/ping", func(ctx context.Context, in None) (Item, error) {
		ran = true
		return Item{Name: "pong"}, nil
	})
	w := send(t, mustRegister(t, app), "POST", "/api/ping?x=1", "garbage", "Content-Type", "text/plain")
	if w.Code != 201 || !ran {
		t.Fatalf("status %d, body %q, ran %v", w.Code, w.Body.String(), ran)
	}
}

func TestHandlerErrorShapes(t *testing.T) {
	type In struct {
		ID int `path:"id"`
	}
	logs := captureLog(t)
	app := New(Config{})
	Get(app, "/items/{id}", func(ctx context.Context, in In) (Item, error) {
		if in.ID == 7 {
			return Item{ID: 7}, NotFound("No item has id 7.")
		}
		return Item{ID: 8}, errors.New("db password hunter2")
	})
	h := mustRegister(t, app)
	w := send(t, h, "GET", "/api/items/7", "")
	if code, msg, _ := apiError(t, w); w.Code != 404 || code != "not_found" || msg != "No item has id 7." {
		t.Fatalf("status %d, code %q, message %q", w.Code, code, msg)
	}
	w = send(t, h, "GET", "/api/items/8", "")
	if code, _, _ := apiError(t, w); w.Code != 500 || code != "internal" || strings.Contains(w.Body.String(), "hunter2") {
		t.Fatalf("status %d, body %q", w.Code, w.Body.String())
	}
	if !strings.Contains(logs.String(), "db password hunter2") {
		t.Fatalf("the error was not logged: %q", logs.String())
	}
}

func TestValidationRunsBeforeHandler(t *testing.T) {
	type In struct {
		Name  string `json:"name" warp:"required"`
		Email string `json:"email" warp:"email"`
		Page  int    `query:"page" warp:"min=1"`
	}
	ran := false
	app := New(Config{})
	Post(app, "/people", func(ctx context.Context, in In) (Item, error) {
		ran = true
		return Item{}, nil
	})
	w := send(t, mustRegister(t, app), "POST", "/api/people?page=0", `{"email":"nope"}`)
	code, msg, fields := apiError(t, w)
	want := map[string]string{"name": "This is required.", "email": "Enter an email address."}
	if w.Code != 422 || code != "validation" || msg != "2 fields need attention." || !reflect.DeepEqual(fields, want) || ran {
		t.Fatalf("status %d, code %q, message %q, fields %v, ran %v", w.Code, code, msg, fields, ran)
	}
	w = send(t, mustRegister(t, app), "POST", "/api/people?page=-1", `{"name":"Ada"}`)
	if _, _, fields := apiError(t, w); w.Code != 422 || fields["page"] != "Enter 1 or more." || ran {
		t.Fatalf("status %d, fields %v, ran %v", w.Code, fields, ran)
	}
}

func TestBodyCannotSetPathField(t *testing.T) {
	type In struct {
		ID   int    `path:"id"`
		Page int    `query:"page"`
		Name string `json:"name"`
	}
	app := New(Config{})
	Put(app, "/items/{id}", func(ctx context.Context, in In) (Item, error) {
		return Item{ID: in.ID, Name: in.Name}, nil
	})
	h := mustRegister(t, app)
	for _, c := range []struct{ body, field string }{
		{`{"id":9}`, "id"}, {`{"ID":9}`, "ID"}, {`{"page":2}`, "page"}, {`{"Page":2}`, "Page"},
	} {
		w := send(t, h, "PUT", "/api/items/3", c.body)
		code, msg, _ := apiError(t, w)
		want := `The body has the field "` + c.field + `", which this route does not take.`
		if w.Code != 400 || code != "bad_request" || msg != want {
			t.Errorf("%s: status %d, code %q, message %q", c.body, w.Code, code, msg)
		}
	}
	if w := send(t, h, "PUT", "/api/items/3", `{"name":"x"}`); w.Body.String() != `{"id":3,"name":"x"}`+"\n" {
		t.Fatalf("body %q", w.Body.String())
	}
}

func TestRequestAndResponseHeader(t *testing.T) {
	type In struct {
		ID int `path:"id"`
	}
	app := New(Config{})
	Get(app, "/items/{id}", func(ctx context.Context, in In) (Item, error) {
		r := Request(ctx)
		if r == nil || r.Context() != ctx {
			return Item{}, errors.New("Request(ctx) is not the request with ctx")
		}
		ResponseHeader(ctx).Set("Set-Cookie", "session=abc")
		if r.Header.Get("X-Who") == "nobody" {
			return Item{}, Forbidden("Sign in first.")
		}
		return Item{ID: in.ID, Name: r.Header.Get("X-Who")}, nil
	})
	h := mustRegister(t, app)
	w := send(t, h, "GET", "/api/items/4", "", "X-Who", "ada")
	if w.Code != 200 || w.Body.String() != `{"id":4,"name":"ada"}`+"\n" || w.Header().Get("Set-Cookie") != "session=abc" {
		t.Fatalf("status %d, body %q, Set-Cookie %q", w.Code, w.Body.String(), w.Header().Get("Set-Cookie"))
	}
	// Headers go with an error answer too.
	w = send(t, h, "GET", "/api/items/4", "", "X-Who", "nobody")
	if w.Code != 403 || w.Header().Get("Set-Cookie") != "session=abc" {
		t.Fatalf("status %d, Set-Cookie %q", w.Code, w.Header().Get("Set-Cookie"))
	}
}

func TestRequestOutsideTypedRoute(t *testing.T) {
	if r := Request(context.Background()); r != nil {
		t.Fatalf("Request = %v, want nil", r)
	}
	h := ResponseHeader(context.Background())
	if h == nil || len(h) != 0 {
		t.Fatalf("ResponseHeader = %v, want an empty header", h)
	}
	h.Set("X", "y") // must not panic
}

func TestRoutesDescribeThemselves(t *testing.T) {
	type ShowIn struct {
		ID int `path:"id"`
	}
	app := New(Config{})
	Get(app, "/items/{id}", func(ctx context.Context, in ShowIn) (Item, error) { return Item{}, nil })
	Post(app, "/items", func(ctx context.Context, in None) (Item, error) { return Item{}, nil })
	Put(app, "/items/{id}", func(ctx context.Context, in ShowIn) (Item, error) { return Item{}, nil }, Status(202))
	Delete(app, "/items/{id}", func(ctx context.Context, in ShowIn) (None, error) { return None{}, nil }, Status(200))
	mustRegister(t, app)
	in, item, none := reflect.TypeFor[ShowIn](), reflect.TypeFor[Item](), reflect.TypeFor[None]()
	want := []Route{
		{Method: "GET", Path: "/api/items/{id}", In: in, Out: item, Status: 200},
		{Method: "POST", Path: "/api/items", In: none, Out: item, Status: 201},
		{Method: "PUT", Path: "/api/items/{id}", In: in, Out: item, Status: 202},
		{Method: "DELETE", Path: "/api/items/{id}", In: in, Out: none, Status: 204},
	}
	if got := app.Routes(); !reflect.DeepEqual(got, want) {
		t.Fatalf("Routes() = %v\nwant %v", got, want)
	}
}

func TestNilSliceOut(t *testing.T) {
	app := New(Config{})
	Get(app, "/items", func(ctx context.Context, in None) ([]Item, error) { return nil, nil })
	w := send(t, mustRegister(t, app), "GET", "/api/items", "")
	if w.Code != 200 || w.Body.String() != "[]\n" {
		t.Fatalf("status %d, body %q", w.Code, w.Body.String())
	}
}

func TestTypedRouteGoesThroughWith(t *testing.T) {
	var log []string
	app := New(Config{})
	mw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			note(&log, "with")
			next.ServeHTTP(w, r)
		})
	}
	Get(app, "/items", func(ctx context.Context, in None) ([]Item, error) {
		note(&log, "handler")
		return nil, nil
	}, With(mw))
	send(t, mustRegister(t, app), "GET", "/api/items", "")
	if want := []string{"with", "handler"}; !reflect.DeepEqual(notes(&log), want) {
		t.Fatalf("log %v, want %v", notes(&log), want)
	}
}

func TestStatusOutOfRange(t *testing.T) {
	type ShowIn struct {
		Name string
	}
	app := New(Config{})
	app.Handle("GET /a", text("a"), Status(0))
	Get(app, "/b", func(context.Context, None) (Item, error) { return Item{}, nil }, Status(600))
	Post(app, "/c", func(context.Context, None) (None, error) { return None{}, nil }, Status(99))
	Get(app, "/d", func(context.Context, ShowIn) (Item, error) { return Item{}, nil }, Status(-1))
	Get(app, "/e", func(context.Context, None) (Item, error) { return Item{}, nil }, Status(100))
	Get(app, "/f", func(context.Context, None) (Item, error) { return Item{}, nil }, Status(599))
	app.Handle("GET /g", text("g"))
	want := strings.Join([]string{
		"warp: GET /a: Status(0) is not an HTTP status; use one from 100 to 599.",
		"warp: GET /b: Status(600) is not an HTTP status; use one from 100 to 599.",
		"warp: POST /c: Status(99) is not an HTTP status; use one from 100 to 599.",
		"warp: GET /d: Status(-1) is not an HTTP status; use one from 100 to 599.",
		"warp: GET /d: ShowIn.Name has no json, query or path tag; add one.",
	}, "\n")
	if got := errText(app.Err()); got != want {
		t.Fatalf("Err() =\n%s\nwant\n%s", got, want)
	}
	var paths []string
	for _, r := range app.Routes() {
		paths = append(paths, r.Path)
	}
	if want := []string{"/api/e", "/api/f", "/api/g"}; !reflect.DeepEqual(paths, want) {
		t.Fatalf("Routes() paths %v, want %v", paths, want)
	}
}
