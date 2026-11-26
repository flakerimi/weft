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
		{Name: "items", Method: "GET", Path: "/api/items/{id}", In: in, Out: item, Status: 200},
		{Name: "post_items", Method: "POST", Path: "/api/items", In: none, Out: item, Status: 201},
		{Name: "put_items", Method: "PUT", Path: "/api/items/{id}", In: in, Out: item, Status: 202},
		{Name: "delete_items", Method: "DELETE", Path: "/api/items/{id}", In: in, Out: none, Status: 204},
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
	Get(app, "/e", func(context.Context, None) (Item, error) { return Item{}, nil }, Status(200))
	Get(app, "/f", func(context.Context, None) (Item, error) { return Item{}, nil }, Status(599))
	app.Handle("GET /g", text("g"))
	Get(app, "/h", func(context.Context, None) (Item, error) { return Item{}, nil }, Status(100))
	app.Handle("GET /i", text("i"), Status(199))
	const tail = " is not a status a route can answer with; use one from 200 to 599."
	want := strings.Join([]string{
		"warp: GET /a: Status(0)" + tail,
		"warp: GET /b: Status(600)" + tail,
		"warp: POST /c: Status(99)" + tail,
		"warp: GET /d: Status(-1)" + tail,
		"warp: GET /d: ShowIn.Name has no json, query, path or body tag; add one.",
		"warp: GET /h: Status(100)" + tail,
		"warp: GET /i: Status(199)" + tail,
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

func TestPatchChecksOnlySentFields(t *testing.T) {
	type Address struct {
		City string `json:"city" warp:"required"`
		Zip  string `json:"zip"`
	}
	type In struct {
		ID      int     `path:"id" warp:"min=1"`
		Name    string  `json:"name" warp:"required"`
		Price   float64 `json:"price" warp:"min=0"`
		Address Address `json:"address"`
	}
	app := New(Config{})
	answer := func(ctx context.Context, in In) (Item, error) { return Item{ID: in.ID}, nil }
	Patch(app, "/items/{id}", answer)
	Post(app, "/items/{id}", answer)
	Put(app, "/items/{id}", answer)
	Patch(app, "/generic/{id}", func(ctx context.Context, in withID[bodyFields]) (Item, error) { return Item{}, nil })
	h := mustRegister(t, app)
	for _, c := range []struct {
		method, path, body string
		status             int
		fields             map[string]string
	}{
		{"PATCH", "/api/items/1", `{"price":5}`, 200, nil},
		{"PATCH", "/api/items/1", `{}`, 200, nil},
		{"PATCH", "/api/items/1", ``, 200, nil},
		{"POST", "/api/items/1", `{"price":5}`, 422, map[string]string{"name": "This is required.", "address.city": "This is required."}},
		{"PUT", "/api/items/1", `{"price":5}`, 422, map[string]string{"name": "This is required.", "address.city": "This is required."}},
		{"PATCH", "/api/items/1", `{"name":""}`, 422, map[string]string{"name": "This is required."}},
		{"PATCH", "/api/items/1", `{"NAME":" ","price":-1}`, 422, map[string]string{"name": "This is required.", "price": "Enter 0 or more."}},
		// A nested field that was sent is checked whole.
		{"PATCH", "/api/items/1", `{"address":{"zip":"1"}}`, 422, map[string]string{"address.city": "This is required."}},
		// The path is checked on every route.
		{"PATCH", "/api/items/-1", `{"price":5}`, 422, map[string]string{"id": "Enter 1 or more."}},
		{"PATCH", "/api/generic/1", `{"price":5}`, 200, nil},
		{"PATCH", "/api/generic/1", `{"name":"","price":5}`, 422, map[string]string{"name": "This is required."}},
	} {
		w := send(t, h, c.method, c.path, c.body)
		if c.status == 200 {
			if w.Code != 200 {
				t.Errorf("%s %s %s: status %d, body %q", c.method, c.path, c.body, w.Code, w.Body.String())
			}
			continue
		}
		if _, _, fields := apiError(t, w); w.Code != c.status || !reflect.DeepEqual(fields, c.fields) {
			t.Errorf("%s %s %s: status %d, fields %v", c.method, c.path, c.body, w.Code, fields)
		}
	}
	// Validate called directly checks every field.
	if got := fieldsOf(t, In{ID: 1, Price: 5, Address: Address{City: "Oslo"}}); !reflect.DeepEqual(got, map[string]string{"name": "This is required."}) {
		t.Fatalf("Validate: fields %v", got)
	}
}

// rangeIn checks itself across fields, with a value receiver.
type rangeIn struct {
	From int `json:"from" warp:"min=0"`
	To   int `json:"to"`
}

func (in rangeIn) Validate(ctx context.Context) error {
	ResponseHeader(ctx).Add("X-Checked", "input")
	switch {
	case in.To == 99:
		return errors.New("the calendar is down")
	case in.To == 42:
		return Conflict("That range is booked.")
	case Sent(ctx, "to") && in.To < in.From:
		return Invalid(map[string]string{"to": "End after the start."})
	}
	return nil
}

// spanBody checks itself with a pointer receiver.
type spanBody struct {
	From int `json:"from"`
	To   int `json:"to"`
}

func (b *spanBody) Validate(ctx context.Context) error {
	ResponseHeader(ctx).Add("X-Checked", "body")
	if b.To < b.From {
		return Invalid(map[string]string{"to": "End after the start.", "from": "Start before the end."})
	}
	return nil
}

type spanIn struct {
	ID   int      `path:"id"`
	Body spanBody `body:"json"`
}

func (in *spanIn) Validate(ctx context.Context) error {
	ResponseHeader(ctx).Add("X-Checked", "input")
	if in.ID == 7 {
		return Invalid(map[string]string{"id": "This one is closed.", "to": "Not this end."})
	}
	return nil
}

// spanOnly has no method of its own; its body type has one.
type spanOnly struct {
	Body spanBody `body:"json"`
}

func TestInputValidateMethod(t *testing.T) {
	logs := captureLog(t)
	ran := false
	app := New(Config{})
	Post(app, "/ranges", func(ctx context.Context, in rangeIn) (Item, error) { ran = true; return Item{}, nil })
	Patch(app, "/ranges", func(ctx context.Context, in rangeIn) (Item, error) { ran = true; return Item{}, nil })
	Post(app, "/spans/{id}", func(ctx context.Context, in spanIn) (Item, error) { ran = true; return Item{}, nil })
	Post(app, "/only", func(ctx context.Context, in spanOnly) (Item, error) { ran = true; return Item{}, nil })
	Patch(app, "/spans/{id}", func(ctx context.Context, in spanIn) (Item, error) { ran = true; return Item{}, nil })
	Patch(app, "/only", func(ctx context.Context, in spanOnly) (Item, error) { ran = true; return Item{}, nil })
	h := mustRegister(t, app)
	for _, c := range []struct {
		method, path, body string
		status             int
		code, msg          string
		fields             map[string]string
		checked            []string
	}{
		{"POST", "/api/ranges", `{"from":1,"to":2}`, 201, "", "", nil, []string{"input"}},
		{"POST", "/api/ranges", `{"from":3,"to":2}`, 422, "validation", "1 field needs attention.", map[string]string{"to": "End after the start."}, []string{"input"}},
		{"POST", "/api/ranges", `{"from":-1,"to":-5}`, 422, "validation", "1 field needs attention.", map[string]string{"from": "Enter 0 or more."}, nil},
		{"POST", "/api/ranges", `{"to":99}`, 500, "internal", "Something went wrong on the server; its log has the details.", nil, []string{"input"}},
		{"POST", "/api/ranges", `{"to":42}`, 409, "conflict", "That range is booked.", nil, []string{"input"}},
		// On PATCH it runs too, and can ask what was sent.
		{"PATCH", "/api/ranges", `{"from":3}`, 200, "", "", nil, []string{"input"}},
		{"PATCH", "/api/ranges", `{"from":3,"to":2}`, 422, "validation", "1 field needs attention.", map[string]string{"to": "End after the start."}, []string{"input"}},
		{"POST", "/api/only", `{"from":3,"to":2}`, 422, "validation", "2 fields need attention.", map[string]string{"to": "End after the start.", "from": "Start before the end."}, []string{"body"}},
		{"POST", "/api/spans/1", `{"from":1,"to":2}`, 201, "", "", nil, []string{"input", "body"}},
		// The input's comes first, and the two are merged.
		{"POST", "/api/spans/7", `{"from":3,"to":2}`, 422, "validation", "3 fields need attention.",
			map[string]string{"id": "This one is closed.", "to": "Not this end.", "from": "Start before the end."}, []string{"input", "body"}},
		// On PATCH the body holds only what was sent, so its type's method is
		// not called; the input's own still is.
		{"PATCH", "/api/only", `{"from":3,"to":2}`, 200, "", "", nil, nil},
		{"PATCH", "/api/only", `{"to":2}`, 200, "", "", nil, nil},
		{"PATCH", "/api/spans/1", `{"from":3,"to":2}`, 200, "", "", nil, []string{"input"}},
		{"PATCH", "/api/spans/7", `{"from":3,"to":2}`, 422, "validation", "2 fields need attention.",
			map[string]string{"id": "This one is closed.", "to": "Not this end."}, []string{"input"}},
	} {
		ran = false
		w := send(t, h, c.method, c.path, c.body)
		if got := w.Header().Values("X-Checked"); !reflect.DeepEqual(got, c.checked) {
			t.Errorf("%s %s %s: checked %q, want %q", c.method, c.path, c.body, got, c.checked)
		}
		if c.status < 300 {
			if w.Code != c.status || !ran {
				t.Errorf("%s %s %s: status %d, ran %v, body %q", c.method, c.path, c.body, w.Code, ran, w.Body.String())
			}
			continue
		}
		if code, msg, fields := apiError(t, w); w.Code != c.status || code != c.code || msg != c.msg || !reflect.DeepEqual(fields, c.fields) || ran {
			t.Errorf("%s %s %s: status %d, code %q, message %q, fields %v, ran %v", c.method, c.path, c.body, w.Code, code, msg, fields, ran)
		}
	}
	if !strings.Contains(logs.String(), "the calendar is down") {
		t.Fatalf("the error was not logged: %q", logs.String())
	}
}
