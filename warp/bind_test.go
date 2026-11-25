package warp

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"
)

type Address struct {
	Zip string `json:"zip"`
}

type CreateIn struct {
	Name    string   `json:"name" warp:"required"`
	Price   float64  `json:"price"`
	Active  bool     `json:"active"`
	Tags    []string `json:"tags"`
	Address Address  `json:"address"`
}

func TestBodyFailures(t *testing.T) {
	limit := int64(1 << 20)
	for _, c := range []struct {
		name, body string
		status     int
		code, msg  string
		fields     map[string]string
	}{
		{"too large", `{"name":"` + strings.Repeat("a", int(limit)-10) + `"}`, 413, "too_large", "The body is larger than 1 MB.", nil},
		{"not json", `garbage`, 400, "bad_request", "The body is not valid JSON.", nil},
		{"cut short", `{"name":`, 400, "bad_request", "The body is not valid JSON.", nil},
		{"text after the value", `{"name":"a"} x`, 400, "bad_request", "The body is not valid JSON.", nil},
		{"a second object", `{"name":"a"}{"name":"b"}`, 400, "bad_request", "The body is not valid JSON.", nil},
		{"a list", `[1]`, 400, "bad_request", "The body must be a JSON object.", nil},
		{"a string", `"x"`, 400, "bad_request", "The body must be a JSON object.", nil},
		{"null", `null`, 400, "bad_request", "The body must be a JSON object.", nil},
		{"unknown field", `{"nme":"a"}`, 400, "bad_request", `The body has the field "nme", which this route does not take.`, nil},
		{"unknown nested field", `{"name":"a","address":{"zp":"1"}}`, 400, "bad_request", `The body has the field "zp", which this route does not take.`, nil},
		{"unknown field after a wrong type", `{"name":5,"nme":"a"}`, 400, "bad_request", `The body has the field "nme", which this route does not take.`, nil},
		{"text wanted", `{"name":5}`, 400, "bad_request", "1 field has the wrong type.", map[string]string{"name": "Send text."}},
		{"number wanted", `{"name":"a","price":"5"}`, 400, "bad_request", "1 field has the wrong type.", map[string]string{"price": "Send a number."}},
		{"bool wanted", `{"name":"a","active":"yes"}`, 400, "bad_request", "1 field has the wrong type.", map[string]string{"active": "Send true or false."}},
		{"list wanted", `{"name":"a","tags":"red"}`, 400, "bad_request", "1 field has the wrong type.", map[string]string{"tags": "Send a list."}},
		{"text in a list", `{"name":"a","tags":[1]}`, 400, "bad_request", "1 field has the wrong type.", map[string]string{"tags": "Send text."}},
		{"object wanted", `{"name":"a","address":"Main St"}`, 400, "bad_request", "1 field has the wrong type.", map[string]string{"address": "Send an object."}},
		{"nested wrong type", `{"name":"a","address":{"zip":1}}`, 400, "bad_request", "1 field has the wrong type.", map[string]string{"address.zip": "Send text."}},
		{"empty body", ``, 422, "validation", "1 field needs attention.", map[string]string{"name": "This is required."}},
		{"blank body", " \n\t ", 422, "validation", "1 field needs attention.", map[string]string{"name": "This is required."}},
	} {
		t.Run(c.name, func(t *testing.T) {
			ran := false
			app := New(Config{MaxBodyBytes: limit})
			Post(app, "/items", func(ctx context.Context, in CreateIn) (Item, error) {
				ran = true
				return Item{}, nil
			})
			w := send(t, mustRegister(t, app), "POST", "/api/items", c.body)
			code, msg, fields := apiError(t, w)
			if w.Code != c.status || code != c.code || msg != c.msg || !reflect.DeepEqual(fields, c.fields) {
				t.Fatalf("status %d, code %q, message %q, fields %v", w.Code, code, msg, fields)
			}
			if ran {
				t.Fatal("the handler ran")
			}
		})
	}
}

func TestBodyAtTheLimit(t *testing.T) {
	limit := int64(2 << 20)
	app := New(Config{MaxBodyBytes: limit})
	Post(app, "/items", func(ctx context.Context, in CreateIn) (Item, error) { return Item{Name: in.Name[:1]}, nil })
	h := mustRegister(t, app)
	fits := `{"name":"` + strings.Repeat("a", int(limit)-11) + `"}`
	if len(fits) != int(limit) {
		t.Fatalf("body of %d bytes", len(fits))
	}
	if w := send(t, h, "POST", "/api/items", fits); w.Code != 201 {
		t.Fatalf("a body of the limit: status %d, body %.200q", w.Code, w.Body.String())
	}
	w := send(t, h, "POST", "/api/items", fits+" ")
	if code, msg, _ := apiError(t, w); w.Code != 413 || code != "too_large" || msg != "The body is larger than 2 MB." {
		t.Fatalf("one byte over: status %d, code %q, message %q", w.Code, code, msg)
	}
}

func TestBodyOnlyForPostPutPatch(t *testing.T) {
	type Find struct {
		Q string `query:"q"`
	}
	app := New(Config{})
	Get(app, "/find", func(ctx context.Context, in Find) (Item, error) { return Item{Name: in.Q}, nil })
	Delete(app, "/find", func(ctx context.Context, in Find) (Item, error) { return Item{Name: in.Q}, nil })
	Post(app, "/find", func(ctx context.Context, in Find) (Item, error) { return Item{Name: in.Q}, nil })
	h := mustRegister(t, app)
	for _, method := range []string{"GET", "DELETE"} {
		if w := send(t, h, method, "/api/find?q=a", "garbage", "Content-Type", "text/plain"); w.Code != 200 || !strings.Contains(w.Body.String(), `"name":"a"`) {
			t.Errorf("%s: status %d, body %q", method, w.Code, w.Body.String())
		}
	}
	// A POST reads its body even when no field is tagged json, and an empty one is {}.
	if w := send(t, h, "POST", "/api/find?q=a", ""); w.Code != 201 {
		t.Errorf("empty POST: status %d, body %q", w.Code, w.Body.String())
	}
	if w := send(t, h, "POST", "/api/find?q=a", `{"q":"b"}`); w.Code != 400 {
		t.Errorf("POST naming a query field: status %d, body %q", w.Code, w.Body.String())
	}
}

func TestQueryConversion(t *testing.T) {
	type In struct {
		Page   int      `query:"page"`
		Small  int8     `query:"small"`
		Count  uint     `query:"count"`
		Price  float32  `query:"price"`
		Open   bool     `query:"open"`
		Q      string   `query:"q"`
		Limit  *int     `query:"limit"`
		Sort   *string  `query:"sort"`
		Tags   []string `query:"tag"`
		Ignore string   `json:"-" query:"ignore"`
	}
	var got In
	app := New(Config{})
	Get(app, "/items", func(ctx context.Context, in In) (Item, error) {
		got = in
		return Item{}, nil
	})
	h := mustRegister(t, app)

	w := send(t, h, "GET", "/api/items?page=two", "")
	if code, msg, fields := apiError(t, w); w.Code != 400 || code != "bad_request" || msg != "1 field has the wrong type." ||
		!reflect.DeepEqual(fields, map[string]string{"page": "Send a whole number."}) {
		t.Fatalf("page=two: status %d, code %q, message %q, fields %v", w.Code, code, msg, fields)
	}

	w = send(t, h, "GET", "/api/items?small=300&count=-1&price=cheap&open=maybe&page=1.5&limit=x", "")
	want := map[string]string{
		"small": "Send a whole number.", "count": "Send a whole number.", "price": "Send a number.",
		"open": "Send true or false.", "page": "Send a whole number.", "limit": "Send a whole number.",
	}
	if _, msg, fields := apiError(t, w); w.Code != 400 || msg != "6 fields have the wrong type." || !reflect.DeepEqual(fields, want) {
		t.Fatalf("several: status %d, message %q, fields %v", w.Code, msg, fields)
	}

	w = send(t, h, "GET", "/api/items?price=NaN", "")
	if _, _, fields := apiError(t, w); w.Code != 400 || fields["price"] != "Send a number." {
		t.Fatalf("price=NaN: status %d, fields %v", w.Code, fields)
	}

	// Absent keys leave the zero value, and an empty value is absent but for text.
	got = In{}
	if w := send(t, h, "GET", "/api/items?page=&open=&limit=&q=&sort=", ""); w.Code != 200 {
		t.Fatalf("empty values: status %d, body %q", w.Code, w.Body.String())
	}
	if got.Page != 0 || got.Open || got.Limit != nil || got.Q != "" || got.Sort == nil || *got.Sort != "" {
		t.Fatalf("empty values: %+v", got)
	}

	got = In{}
	send(t, h, "GET", "/api/items", "")
	if !reflect.DeepEqual(got, In{}) {
		t.Fatalf("no query: %+v", got)
	}

	got = In{}
	send(t, h, "GET", "/api/items?page=3&page=4&small=-128&count=9&price=1.5&open=true&q=lamp&limit=0&sort=name&tag=a&tag=&tag=b&ignore=x", "")
	if got.Page != 3 || got.Small != -128 || got.Count != 9 || got.Price != 1.5 || !got.Open || got.Q != "lamp" ||
		got.Limit == nil || *got.Limit != 0 || got.Sort == nil || *got.Sort != "name" ||
		!reflect.DeepEqual(got.Tags, []string{"a", "", "b"}) || got.Ignore != "x" {
		t.Fatalf("every kind: %+v", got)
	}
}

func TestPathConversion(t *testing.T) {
	type In struct {
		ID   int8    `path:"id"`
		On   bool    `path:"on"`
		At   float64 `path:"at"`
		Rest string  `path:"rest"`
	}
	var got In
	app := New(Config{})
	Get(app, "/items/{id}/{on}/{at}/{rest...}", func(ctx context.Context, in In) (Item, error) {
		got = in
		return Item{}, nil
	})
	type Show struct {
		ID int `path:"id"`
	}
	Get(app, "/items/{id}", func(ctx context.Context, in Show) (Item, error) { return Item{ID: in.ID}, nil })
	Get(app, "/items/{$}", func(ctx context.Context, in None) ([]Item, error) { return nil, nil })
	h := mustRegister(t, app)

	w := send(t, h, "GET", "/api/items/abc", "")
	if code, msg, _ := apiError(t, w); w.Code != 404 || code != "not_found" || msg != "Nothing is at /api/items/abc." {
		t.Fatalf("abc: status %d, code %q, message %q", w.Code, code, msg)
	}
	for _, path := range []string{"/api/items/300/true/1/x", "/api/items/1/yes/1/x", "/api/items/1/true/one/x"} {
		if w := send(t, h, "GET", path, ""); w.Code != 404 {
			t.Errorf("%s: status %d", path, w.Code)
		}
	}
	if w := send(t, h, "GET", "/api/items/-5/false/2.5/a/b/c", ""); w.Code != 200 {
		t.Fatalf("status %d, body %q", w.Code, w.Body.String())
	}
	if want := (In{ID: -5, On: false, At: 2.5, Rest: "a/b/c"}); got != want {
		t.Fatalf("input %+v, want %+v", got, want)
	}
	if w := send(t, h, "GET", "/api/items/", ""); w.Code != 200 || w.Body.String() != "[]\n" {
		t.Fatalf("{$}: status %d, body %q", w.Code, w.Body.String())
	}
}

func TestFailureOrder(t *testing.T) {
	type In struct {
		ID   int    `path:"id"`
		Page int    `query:"page"`
		Name string `json:"name" warp:"required"`
	}
	ran := false
	app := New(Config{})
	Put(app, "/items/{id}", func(ctx context.Context, in In) (Item, error) {
		ran = true
		return Item{}, nil
	})
	h := mustRegister(t, app)
	for _, c := range []struct {
		path, body string
		status     int
		msg        string
	}{
		{"/api/items/x?page=x", `{"name":5}`, 400, "1 field has the wrong type."},
		{"/api/items/x?page=x", `{"nme":5}`, 400, `The body has the field "nme", which this route does not take.`},
		{"/api/items/x?page=x", `{}`, 400, "1 field has the wrong type."},
		{"/api/items/x?page=1", `{}`, 404, "Nothing is at /api/items/x."},
		{"/api/items/1?page=1", `{}`, 422, "1 field needs attention."},
	} {
		w := send(t, h, "PUT", c.path, c.body)
		if _, msg, fields := apiError(t, w); w.Code != c.status || msg != c.msg {
			t.Errorf("%s %s: status %d, message %q, fields %v", c.path, c.body, w.Code, msg, fields)
		}
	}
	w := send(t, h, "PUT", "/api/items/x?page=x", `{"name":5}`)
	if _, _, fields := apiError(t, w); fields["name"] != "Send text." || len(fields) != 1 {
		t.Errorf("a body problem comes before the query's: fields %v", fields)
	}
	if ran {
		t.Fatal("the handler ran")
	}
}

type Paging struct {
	Page int `query:"page" warp:"min=1"`
}

type Owner struct {
	OwnerID int `path:"owner"`
}

func TestEmbeddedFieldsAreTheInputs(t *testing.T) {
	type In struct {
		Paging
		*Owner
		Name   string `json:"name"`
		hidden string
	}
	var got In
	app := New(Config{})
	Post(app, "/owners/{owner}/items", func(ctx context.Context, in In) (Item, error) {
		got = in
		return Item{}, nil
	})
	h := mustRegister(t, app)
	if w := send(t, h, "POST", "/api/owners/4/items?page=2", `{"name":"lamp"}`); w.Code != 201 {
		t.Fatalf("status %d, body %q", w.Code, w.Body.String())
	}
	if got.Page != 2 || got.Owner == nil || got.OwnerID != 4 || got.Name != "lamp" {
		t.Fatalf("input %+v", got)
	}
	if w := send(t, h, "POST", "/api/owners/4/items?page=0", `{"hidden":"x"}`); w.Code != 400 {
		t.Fatalf("an unexported field is not in the body: status %d, body %q", w.Code, w.Body.String())
	}
	w := send(t, h, "POST", "/api/owners/4/items?page=-1", `{}`)
	if _, _, fields := apiError(t, w); w.Code != 422 || fields["page"] != "Enter 1 or more." {
		t.Fatalf("an embedded rule: status %d, fields %v", w.Code, fields)
	}
}

func TestRegistrationMistakes(t *testing.T) {
	const at = "warp: GET /products/{id}: "
	for _, c := range []struct {
		name     string
		register func(app *App)
		want     []string
	}{
		{"not a struct", func(app *App) {
			type ShowIn string
			Get(app, "/products/{id}", func(context.Context, ShowIn) (Item, error) { return Item{}, nil })
		}, []string{"the input ShowIn must be a struct or warp.None."}},
		{"a pointer", func(app *App) {
			type ShowIn struct{}
			Get(app, "/products/{id}", func(context.Context, *ShowIn) (Item, error) { return Item{}, nil })
		}, []string{"the input *warp.ShowIn must be a struct or warp.None."}},
		{"no tag", func(app *App) {
			type ShowIn struct {
				ID   int `path:"id"`
				Name string
			}
			Get(app, "/products/{id}", func(context.Context, ShowIn) (Item, error) { return Item{}, nil })
		}, []string{"ShowIn.Name has no json, query or path tag; add one."}},
		{"json:\"-\" is no tag", func(app *App) {
			type ShowIn struct {
				ID   int    `path:"id"`
				Name string `json:"-"`
			}
			Get(app, "/products/{id}", func(context.Context, ShowIn) (Item, error) { return Item{}, nil })
		}, []string{"ShowIn.Name has no json, query or path tag; add one."}},
		{"path tag without wildcard", func(app *App) {
			type ShowIn struct {
				ID  int `path:"id"`
				Rev int `path:"rev"`
			}
			Get(app, "/products/{id}", func(context.Context, ShowIn) (Item, error) { return Item{}, nil })
		}, []string{`ShowIn.Rev is tagged path:"rev", and the path has no {rev}.`}},
		{"wildcard without field", func(app *App) {
			type ShowIn struct {
				Page int `query:"page"`
			}
			Get(app, "/products/{id}", func(context.Context, ShowIn) (Item, error) { return Item{}, nil })
		}, []string{`the path has {id}, and ShowIn has no field tagged path:"id".`}},
		{"json on GET", func(app *App) {
			type ShowIn struct {
				ID   int    `path:"id"`
				Body string `json:"body"`
			}
			Get(app, "/products/{id}", func(context.Context, ShowIn) (Item, error) { return Item{}, nil })
		}, []string{"ShowIn.Body is tagged json, and GET reads no body; tag it query or path."}},
		{"a type no value fills", func(app *App) {
			type ShowIn struct {
				When time.Time         `query:"when"`
				Nums []int             `query:"num"`
				Opts *int              `path:"id"`
				By   map[string]string `query:"by"`
			}
			Get(app, "/products/{id}", func(context.Context, ShowIn) (Item, error) { return Item{}, nil })
		}, []string{
			"ShowIn.When is a time.Time, which a query or path value cannot fill; use a string, a number or a bool.",
			"ShowIn.Nums is a []int, which a query or path value cannot fill; use a string, a number or a bool.",
			"ShowIn.Opts is a *int, which a query or path value cannot fill; use a string, a number or a bool.",
			"ShowIn.By is a map[string]string, which a query or path value cannot fill; use a string, a number or a bool.",
		}},
		{"two tags", func(app *App) {
			type ShowIn struct {
				ID   int    `path:"id" query:"id"`
				Name string `json:"name" query:"name"`
			}
			Get(app, "/products/{id}", func(context.Context, ShowIn) (Item, error) { return Item{}, nil })
		}, []string{
			"ShowIn.ID has both a path and a query tag; keep one.",
			"ShowIn.Name has both a json and a query tag; keep one.",
		}},
		{"two query fields with one key", func(app *App) {
			type ShowIn struct {
				ID   int `path:"id"`
				Page int `query:"page"`
				P    int `query:"page"`
			}
			Get(app, "/products/{id}", func(context.Context, ShowIn) (Item, error) { return Item{}, nil })
		}, []string{`ShowIn.Page and ShowIn.P both have the query key "page"; rename one.`}},
		{"two path fields with one key", func(app *App) {
			type ShowIn struct {
				ID  int    `path:"id"`
				Raw string `path:"id"`
			}
			Get(app, "/products/{id}", func(context.Context, ShowIn) (Item, error) { return Item{}, nil })
		}, []string{`ShowIn.ID and ShowIn.Raw both have the path key "id"; rename one.`}},
		{"a rule mistake", func(app *App) {
			type ShowIn struct {
				ID   int    `path:"id"`
				Name string `query:"name" warp:"emal"`
			}
			Get(app, "/products/{id}", func(context.Context, ShowIn) (Item, error) { return Item{}, nil })
		}, []string{`ShowIn.Name has the rule "emal", which warp does not know; use required, email, url, min=, max=, enum= or unique.`}},
		{"every mistake", func(app *App) {
			type ShowIn struct {
				Name string
				Body string `json:"body"`
				Tag  string `query:"tag" warp:"min=x"`
			}
			Get(app, "/products/{id}", func(context.Context, ShowIn) (Item, error) { return Item{}, nil })
		}, []string{
			"ShowIn.Name has no json, query or path tag; add one.",
			"ShowIn.Body is tagged json, and GET reads no body; tag it query or path.",
			`the path has {id}, and ShowIn has no field tagged path:"id".`,
			`ShowIn.Tag has "min=x"; write a number after min=.`,
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			app := New(Config{})
			c.register(app)
			var want []string
			for _, w := range c.want {
				want = append(want, at+w)
			}
			if got := strings.Split(errText(app.Err()), "\n"); !reflect.DeepEqual(got, want) {
				t.Fatalf("Err() =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
			}
			if routes := app.Routes(); len(routes) != 0 {
				t.Fatalf("Routes() = %v, want none", routes)
			}
			w := send(t, app.Handler(), "GET", "/api/products/1", "")
			if code, _, _ := apiError(t, w); w.Code != 404 || code != "not_found" {
				t.Fatalf("status %d, code %q", w.Code, code)
			}
		})
	}
}

func TestDeleteReadsNoBody(t *testing.T) {
	type DropIn struct {
		ID   int    `path:"id"`
		Note string `json:"note"`
	}
	app := New(Config{})
	Delete(app, "/products/{id}", func(context.Context, DropIn) (Item, error) { return Item{}, nil })
	want := "warp: DELETE /products/{id}: DropIn.Note is tagged json, and DELETE reads no body; tag it query or path."
	if got := errText(app.Err()); got != want {
		t.Fatalf("Err() = %q, want %q", got, want)
	}
}

func TestDuplicateJSONKey(t *testing.T) {
	type Base struct {
		Name string `json:"name"`
	}
	type MakeIn struct {
		Base
		Title string `json:"name"`
	}
	app := New(Config{})
	Post(app, "/products", func(context.Context, MakeIn) (Item, error) { return Item{}, nil })
	want := `warp: POST /products: MakeIn.Name and MakeIn.Title both have the json key "name"; rename one.`
	if got := errText(app.Err()); got != want {
		t.Fatalf("Err() = %q, want %q", got, want)
	}
}

func TestTypedRouteKeepsRouteChecks(t *testing.T) {
	app := New(Config{})
	Get(app, "products", func(context.Context, None) (Item, error) { return Item{}, nil })
	Get(app, "/api/products", func(context.Context, None) (Item, error) { return Item{}, nil })
	Get(app, "/items", func(context.Context, None) (Item, error) { return Item{}, nil })
	Get(app, "/items", func(context.Context, None) (Item, error) { return Item{}, nil })
	got := strings.Split(errText(app.Err()), "\n")
	if len(got) != 3 || !strings.Contains(got[0], "does not start with /") || !strings.Contains(got[1], "already gets /api") ||
		!strings.Contains(got[2], "cannot be registered") {
		t.Fatalf("Err() = %q", got)
	}
	if routes := app.Routes(); len(routes) != 1 {
		t.Fatalf("Routes() = %v", routes)
	}
}

func TestJSONNameMistakes(t *testing.T) {
	for _, c := range []struct {
		name     string
		register func(app *App)
		want     string
	}{
		{"no name", func(app *App) {
			type MakeIn struct {
				Name string `json:",omitempty"`
			}
			Post(app, "/products", func(context.Context, MakeIn) (Item, error) { return Item{}, nil })
		}, `MakeIn.Name has a json tag with no name; name it, as in json:"name".`},
		{"an empty tag", func(app *App) {
			type MakeIn struct {
				UserID int `json:""`
			}
			Post(app, "/products", func(context.Context, MakeIn) (Item, error) { return Item{}, nil })
		}, `MakeIn.UserID has a json tag with no name; name it, as in json:"user_id".`},
		{"a quote", func(app *App) {
			type MakeIn struct {
				Name string `json:"it's"`
			}
			Post(app, "/products", func(context.Context, MakeIn) (Item, error) { return Item{}, nil })
		}, `MakeIn.Name has the json name "it's", which JSON tags cannot carry; use letters, digits and underscores.`},
		{"a space", func(app *App) {
			type MakeIn struct {
				Name string `json:"first name"`
			}
			Post(app, "/products", func(context.Context, MakeIn) (Item, error) { return Item{}, nil })
		}, `MakeIn.Name has the json name "first name", which JSON tags cannot carry; use letters, digits and underscores.`},
		{"a dash", func(app *App) {
			type MakeIn struct {
				Name string `json:"-,"`
			}
			Post(app, "/products", func(context.Context, MakeIn) (Item, error) { return Item{}, nil })
		}, `MakeIn.Name has the json name "-", which JSON tags cannot carry; use letters, digits and underscores.`},
	} {
		t.Run(c.name, func(t *testing.T) {
			app := New(Config{})
			c.register(app)
			if got, want := errText(app.Err()), "warp: POST /products: "+c.want; got != want {
				t.Fatalf("Err() = %q\nwant %q", got, want)
			}
			if routes := app.Routes(); len(routes) != 0 {
				t.Fatalf("Routes() = %v, want none", routes)
			}
		})
	}
}

func TestJSONNamesThatWork(t *testing.T) {
	type MakeIn struct {
		First string `json:"first-name"`
		Dot   string `json:"a.b"`
		Under string `json:"_x,omitempty"`
		Word  string `json:"größe"`
	}
	var got MakeIn
	app := New(Config{})
	Post(app, "/products", func(ctx context.Context, in MakeIn) (Item, error) {
		got = in
		return Item{}, nil
	})
	h := mustRegister(t, app)
	if w := send(t, h, "POST", "/api/products", `{"first-name":"a","a.b":"b","_x":"c","größe":"d"}`); w.Code != 201 {
		t.Fatalf("status %d, body %q", w.Code, w.Body.String())
	}
	if want := (MakeIn{"a", "b", "c", "d"}); got != want {
		t.Fatalf("input %+v, want %+v", got, want)
	}
}

func TestJSONDashFieldIsValidated(t *testing.T) {
	type In struct {
		X int `json:"-" query:"x" warp:"min=5"`
	}
	app := New(Config{})
	Get(app, "/items", func(ctx context.Context, in In) (Item, error) { return Item{}, nil })
	w := send(t, mustRegister(t, app), "GET", "/api/items?x=2", "")
	if code, _, fields := apiError(t, w); w.Code != 422 || code != "validation" ||
		!reflect.DeepEqual(fields, map[string]string{"x": "Enter 5 or more."}) {
		t.Fatalf("status %d, code %q, fields %v", w.Code, code, fields)
	}
}

func TestBodyMediaType(t *testing.T) {
	const object = `{"name":"a"}`
	for _, c := range []struct {
		contentType, body string
		status            int
		msg               string
	}{
		{"text/plain", object, 415, "Send the body as application/json."},
		{"", object, 415, "Send the body as application/json."},
		{"application/json; charset", object, 415, "Send the body as application/json."},
		{"application/x-www-form-urlencoded", "name=a", 415, "Send the body as application/json."},
		{"multipart/form-data; boundary=x", object, 415, "Send the body as application/json."},
		{"+json", object, 415, "Send the body as application/json."},
		{"text/plain", "garbage", 415, "Send the body as application/json."},
		{"text/plain", "", 422, "1 field needs attention."},
		{"", " \n\t", 422, "1 field needs attention."},
		{"application/json", object, 201, ""},
		{"application/json; charset=utf-8", object, 201, ""},
		{"Application/JSON", object, 201, ""},
		{"application/merge-patch+json", object, 201, ""},
		{"application/json", "garbage", 400, "The body is not valid JSON."},
	} {
		app := New(Config{})
		Post(app, "/items", func(ctx context.Context, in CreateIn) (Item, error) { return Item{}, nil })
		w := send(t, mustRegister(t, app), "POST", "/api/items", c.body, "Content-Type", c.contentType)
		if w.Code != c.status {
			t.Errorf("%q %q: status %d, body %q", c.contentType, c.body, w.Code, w.Body.String())
			continue
		}
		if c.msg == "" {
			continue
		}
		if code, msg, _ := apiError(t, w); msg != c.msg || (c.status == 415 && code != "unsupported_media_type") {
			t.Errorf("%q %q: code %q, message %q", c.contentType, c.body, code, msg)
		}
	}
	// A body too large is a 413 whatever its type.
	app := New(Config{MaxBodyBytes: 1 << 20})
	Post(app, "/items", func(ctx context.Context, in CreateIn) (Item, error) { return Item{}, nil })
	w := send(t, mustRegister(t, app), "POST", "/api/items", strings.Repeat("a", 1<<20+1), "Content-Type", "text/plain")
	if w.Code != 413 {
		t.Fatalf("too large as text/plain: status %d", w.Code)
	}
}

func TestTypedRouteAfterStart(t *testing.T) {
	type ShowIn struct {
		Name string
	}
	app := New(Config{})
	app.mu.Lock()
	app.started = true
	app.mu.Unlock()
	Get(app, "/products/{id}", func(context.Context, ShowIn) (Item, error) { return Item{}, nil }, Status(0))
	want := `warp: route "GET /products/{id}" was registered after the app started serving; register every route before Serve.`
	if got := errText(app.Err()); got != want {
		t.Fatalf("Err() = %q\nwant %q", got, want)
	}
}
