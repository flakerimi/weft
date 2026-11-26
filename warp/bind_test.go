package warp

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/http/httptest"
	"reflect"
	"slices"
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

type Stock struct {
	Count int  `json:"count"`
	Max   *int `json:"max"`
}

func TestEmbeddedFieldsAreTheInputs(t *testing.T) {
	type In struct {
		Paging
		Owner
		Stock
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
	if got.Page != 2 || got.OwnerID != 4 || got.Name != "lamp" || got.Count != 0 || got.Max != nil {
		t.Fatalf("input %+v", got)
	}
	got = In{Stock: Stock{Count: 5}}
	if w := send(t, h, "POST", "/api/owners/4/items?page=2", `{"count":0,"max":0}`); w.Code != 201 {
		t.Fatalf("explicit zeros: status %d, body %q", w.Code, w.Body.String())
	}
	if got.Count != 0 || got.Max == nil || *got.Max != 0 {
		t.Fatalf("explicit zeros: input %+v", got)
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
		}, []string{"ShowIn.Name has no json, query, path or body tag; add one."}},
		{"json:\"-\" is no tag", func(app *App) {
			type ShowIn struct {
				ID   int    `path:"id"`
				Name string `json:"-"`
			}
			Get(app, "/products/{id}", func(context.Context, ShowIn) (Item, error) { return Item{}, nil })
		}, []string{"ShowIn.Name has no json, query, path or body tag; add one."}},
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
				When time.Time      `query:"when"`
				Nums []int          `query:"num"`
				Opts *int           `path:"id"`
				By   map[string]int `query:"by"`
			}
			Get(app, "/products/{id}", func(context.Context, ShowIn) (Item, error) { return Item{}, nil })
		}, []string{
			"ShowIn.When is a time.Time, which a query or path value cannot fill; use a string, a number or a bool.",
			"ShowIn.Nums is a []int, which a query or path value cannot fill; use a string, a number or a bool.",
			"ShowIn.Opts is a *int, which a query or path value cannot fill; use a string, a number or a bool.",
			"ShowIn.By is a map[string]int; a filter map is a map[string]string.",
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
		}, []string{`ShowIn.Name has the rule "emal", which warp does not know; use required, email, url, min=, max=, enum=, unique or readonly.`}},
		{"every mistake", func(app *App) {
			type ShowIn struct {
				Name string
				Body string `json:"body"`
				Tag  string `query:"tag" warp:"min=x"`
			}
			Get(app, "/products/{id}", func(context.Context, ShowIn) (Item, error) { return Item{}, nil })
		}, []string{
			"ShowIn.Name has no json, query, path or body tag; add one.",
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
		}, `MakeIn.Name has the json name "it's", which JSON tags cannot carry; use letters, digits, _, - and .`},
		{"a space", func(app *App) {
			type MakeIn struct {
				Name string `json:"first name"`
			}
			Post(app, "/products", func(context.Context, MakeIn) (Item, error) { return Item{}, nil })
		}, `MakeIn.Name has the json name "first name", which JSON tags cannot carry; use letters, digits, _, - and .`},
		{"a dash", func(app *App) {
			type MakeIn struct {
				Name string `json:"-,"`
			}
			Post(app, "/products", func(context.Context, MakeIn) (Item, error) { return Item{}, nil })
		}, `MakeIn.Name has the json name "-", which JSON tags cannot carry; use letters, digits, _, - and .`},
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
	logs := captureLog(t)
	app := New(Config{})
	app.mu.Lock()
	app.started = true
	app.mu.Unlock()
	Get(app, "/products/{id}", func(context.Context, ShowIn) (Item, error) { return Item{}, nil }, Status(0))
	want := `warp: route "GET /products/{id}" was registered after the app started; register every route before Serve or Start.`
	if got := errText(app.Err()); got != want {
		t.Fatalf("Err() = %q\nwant %q", got, want)
	}
	if !strings.Contains(logs.String(), "registered after the app started") {
		t.Fatalf("the mistake was not logged: %q", logs)
	}
}

// Money reads itself from JSON, so registration does not look inside it.
type Money struct{ Cents int }

func (m *Money) UnmarshalJSON(b []byte) error {
	m.Cents = len(b)
	return nil
}

// Code reads itself from text, so registration does not look inside it.
type Code struct{ V string }

func (c *Code) UnmarshalText(b []byte) error {
	c.V = string(b)
	return nil
}

func TestNestedBodyMistakes(t *testing.T) {
	const at = "warp: POST /orders: "
	const inBody = "; only the route's input itself takes query and path tags."
	for _, c := range []struct {
		name     string
		register func(app *App)
		want     []string
	}{
		{"a nested struct", func(app *App) {
			type Address struct {
				Street string `json:"street"`
				City   string
				Secret string `json:"-"`
			}
			type OrderIn struct {
				Address Address `json:"address"`
			}
			Post(app, "/orders", func(context.Context, OrderIn) (Item, error) { return Item{}, nil })
		}, []string{"Address.City has no json tag; add one."}},
		{"a slice of structs", func(app *App) {
			type Line struct {
				SKU string `query:"sku"`
			}
			type OrderIn struct {
				Lines []Line `json:"lines"`
			}
			Post(app, "/orders", func(context.Context, OrderIn) (Item, error) { return Item{}, nil })
		}, []string{"Line.SKU is tagged query inside a body field" + inBody}},
		{"a pointer to a struct", func(app *App) {
			type Ship struct {
				ID int `json:"id" path:"id"`
			}
			type OrderIn struct {
				Ship *Ship `json:"ship"`
			}
			Post(app, "/orders", func(context.Context, OrderIn) (Item, error) { return Item{}, nil })
		}, []string{"Ship.ID is tagged path inside a body field" + inBody}},
		{"a map of structs", func(app *App) {
			type Note struct {
				Text string `json:",omitempty"`
			}
			type OrderIn struct {
				Notes map[string]Note `json:"notes"`
			}
			Post(app, "/orders", func(context.Context, OrderIn) (Item, error) { return Item{}, nil })
		}, []string{`Note.Text has a json tag with no name; name it, as in json:"text".`}},
		{"an array of pointers to structs", func(app *App) {
			type Part struct {
				Name string `json:"it's"`
			}
			type OrderIn struct {
				Pair [2]*Part `json:"pair"`
			}
			Post(app, "/orders", func(context.Context, OrderIn) (Item, error) { return Item{}, nil })
		}, []string{`Part.Name has the json name "it's", which JSON tags cannot carry; use letters, digits, _, - and .`}},
		{"one key twice, through an embedded struct", func(app *App) {
			type Inner struct {
				A string `json:"a"`
			}
			type Box struct {
				Inner
				B string `json:"a"`
			}
			type OrderIn struct {
				Boxes map[string][]Box `json:"boxes"`
			}
			Post(app, "/orders", func(context.Context, OrderIn) (Item, error) { return Item{}, nil })
		}, []string{`Box.A and Box.B both have the json key "a"; rename one.`}},
		{"deeper, and in field order", func(app *App) {
			type Geo struct {
				Lat float64
			}
			type Address struct {
				Geo  Geo `json:"geo"`
				Zone string
			}
			type OrderIn struct {
				Address Address `json:"address"`
				Note    string
			}
			Post(app, "/orders", func(context.Context, OrderIn) (Item, error) { return Item{}, nil })
		}, []string{
			"Geo.Lat has no json tag; add one.",
			"Address.Zone has no json tag; add one.",
			"OrderIn.Note has no json, query, path or body tag; add one.",
		}},
		{"a self-referencing type", func(app *App) {
			type Node struct {
				City     string
				Children []Node `json:"children"`
				Parent   *Node  `json:"parent"`
			}
			type OrderIn struct {
				Root  Node   `json:"root"`
				Other []Node `json:"other"`
			}
			Post(app, "/orders", func(context.Context, OrderIn) (Item, error) { return Item{}, nil })
		}, []string{"Node.City has no json tag; add one."}},
		{"an embedded pointer, nested", func(app *App) {
			type Meta struct {
				Tag string `json:"tag"`
			}
			type Address struct {
				*Meta
				City string `json:"city"`
			}
			type OrderIn struct {
				Address Address `json:"address"`
			}
			Post(app, "/orders", func(context.Context, OrderIn) (Item, error) { return Item{}, nil })
		}, []string{"Address embeds *Meta, a pointer; embed Meta by value."}},
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
		})
	}
}

func TestEmbeddedPointerInInput(t *testing.T) {
	type Meta struct {
		Tag string `query:"tag"`
	}
	type ShowIn struct {
		*Meta
		ID int `path:"id"`
	}
	app := New(Config{})
	Get(app, "/products/{id}", func(context.Context, ShowIn) (Item, error) { return Item{}, nil })
	want := "warp: GET /products/{id}: ShowIn embeds *Meta, a pointer; embed Meta by value."
	if got := errText(app.Err()); got != want {
		t.Fatalf("Err() = %q, want %q", got, want)
	}
}

func TestNestedBodyBindsAndValidates(t *testing.T) {
	type Address struct {
		City string `json:"city" warp:"required"`
		Zip  string `json:"zip"`
	}
	type Line struct {
		SKU string `json:"sku" warp:"required"`
	}
	type OrderIn struct {
		Address Address         `json:"address"`
		Lines   []Line          `json:"lines"`
		ByName  map[string]Line `json:"by_name"`
		When    time.Time       `json:"when"`
		Money   Money           `json:"money"`
		Code    Code            `json:"code"`
	}
	var got OrderIn
	app := New(Config{})
	Post(app, "/orders", func(ctx context.Context, in OrderIn) (Item, error) {
		got = in
		return Item{}, nil
	})
	h := mustRegister(t, app)
	w := send(t, h, "POST", "/api/orders", `{"address":{"zip":"1"},"lines":[{"sku":""}]}`)
	want := map[string]string{"address.city": "This is required.", "lines.0.sku": "This is required."}
	if _, _, fields := apiError(t, w); w.Code != 422 || !reflect.DeepEqual(fields, want) {
		t.Fatalf("status %d, fields %v", w.Code, fields)
	}
	body := `{"address":{"city":"Oslo"},"lines":[{"sku":"a"}],"by_name":{"x":{"sku":"b"}},` +
		`"when":"2026-11-25T10:00:00Z","money":"1.50","code":"ab"}`
	if w := send(t, h, "POST", "/api/orders", body); w.Code != 201 {
		t.Fatalf("status %d, body %q", w.Code, w.Body.String())
	}
	if got.Address.City != "Oslo" || got.Lines[0].SKU != "a" || got.ByName["x"].SKU != "b" ||
		got.When.Year() != 2026 || got.Money.Cents != 6 || got.Code.V != "ab" {
		t.Fatalf("input %+v", got)
	}
}

func TestMapValuesAreValidated(t *testing.T) {
	type Line struct {
		SKU string `json:"sku" warp:"required"`
	}
	type OrderIn struct {
		ByName map[string]Line `json:"by_name"`
	}
	ran := false
	app := New(Config{})
	Post(app, "/orders", func(ctx context.Context, in OrderIn) (Item, error) {
		ran = true
		return Item{}, nil
	})
	w := send(t, mustRegister(t, app), "POST", "/api/orders", `{"by_name":{"x":{"sku":""}}}`)
	if _, _, fields := apiError(t, w); w.Code != 422 || !reflect.DeepEqual(fields, map[string]string{"by_name.x.sku": "This is required."}) || ran {
		t.Fatalf("status %d, fields %v, ran %v", w.Code, fields, ran)
	}
}

func TestRuleMistakeInMapValue(t *testing.T) {
	type Line struct {
		SKU string `json:"sku" warp:"min=x"`
	}
	type OrderIn struct {
		ByName map[string]Line `json:"by_name"`
	}
	app := New(Config{})
	Post(app, "/orders", func(context.Context, OrderIn) (Item, error) { return Item{}, nil })
	if got, want := errText(app.Err()), `warp: POST /orders: Line.SKU has "min=x"; write a number after min=.`; got != want {
		t.Fatalf("Err() = %q, want %q", got, want)
	}
}

func TestTaggedEmbedIsRefused(t *testing.T) {
	const tail = `; give the field a name, or drop the tag to make %s's fields the input's own.`
	t.Run("top level", func(t *testing.T) {
		type Base struct {
			Note string `json:"note"`
		}
		type ShowIn struct {
			Base `json:"base"`
			Name string `json:"name"`
		}
		app := New(Config{})
		Post(app, "/products", func(context.Context, ShowIn) (Item, error) { return Item{}, nil })
		want := `warp: POST /products: ShowIn embeds Base with the json name "base"` + strings.Replace(tail, "%s", "Base", 1)
		if got := errText(app.Err()); got != want {
			t.Fatalf("Err() = %q\nwant %q", got, want)
		}
	})
	t.Run("nested", func(t *testing.T) {
		type Geo struct {
			Lat float64 `json:"lat"`
		}
		type Address struct {
			Geo  `json:"geo,omitempty"`
			City string `json:"city"`
		}
		type ShowIn struct {
			Address Address `json:"address"`
		}
		app := New(Config{})
		Post(app, "/products", func(context.Context, ShowIn) (Item, error) { return Item{}, nil })
		want := `warp: POST /products: Address embeds Geo with the json name "geo"` + strings.Replace(tail, "%s", "Geo", 1)
		if got := errText(app.Err()); got != want {
			t.Fatalf("Err() = %q\nwant %q", got, want)
		}
	})
	t.Run(`json:"-"`, func(t *testing.T) {
		type Base struct {
			Note string `json:"note"`
		}
		type ShowIn struct {
			Base `json:"-"`
			Name string `json:"name"`
		}
		app := New(Config{})
		Post(app, "/products", func(context.Context, ShowIn) (Item, error) { return Item{}, nil })
		want := `warp: POST /products: ShowIn embeds Base with json:"-"; give the field a name to leave it out, or drop the tag to make Base's fields the input's own.`
		if got := errText(app.Err()); got != want {
			t.Fatalf("Err() = %q\nwant %q", got, want)
		}
		if len(app.Routes()) != 0 {
			t.Fatalf("Routes() = %v", app.Routes())
		}
	})
	t.Run("an embed tagged with no name is flattened", func(t *testing.T) {
		type Base struct {
			Note string `json:"note"`
		}
		type ShowIn struct {
			Base `json:",omitempty"`
		}
		var got ShowIn
		app := New(Config{})
		Post(app, "/products", func(ctx context.Context, in ShowIn) (Item, error) {
			got = in
			return Item{}, nil
		})
		if w := send(t, mustRegister(t, app), "POST", "/api/products", `{"note":"n"}`); w.Code != 201 || got.Note != "n" {
			t.Fatalf("status %d, body %q, input %+v", w.Code, w.Body.String(), got)
		}
	})
}

// A pointer query field sent empty stays nil, so its rules are skipped; sent
// as 0 it points at 0, and its rules apply.
func TestEmptyQueryPointerIsNotSent(t *testing.T) {
	type ListIn struct {
		PerPage *int `query:"per_page" warp:"min=1,max=100"`
	}
	var got ListIn
	app := New(Config{})
	Get(app, "/items", func(ctx context.Context, in ListIn) ([]Item, error) {
		got = in
		return nil, nil
	})
	h := mustRegister(t, app)
	if w := send(t, h, "GET", "/api/items?per_page=", ""); w.Code != 200 || got.PerPage != nil {
		t.Fatalf("per_page=: status %d, body %q, input %+v", w.Code, w.Body.String(), got)
	}
	w := send(t, h, "GET", "/api/items?per_page=0", "")
	if code, _, fields := apiError(t, w); w.Code != 422 || code != "validation" || !reflect.DeepEqual(fields, map[string]string{"per_page": "Enter 1 or more."}) {
		t.Fatalf("per_page=0: status %d, fields %v", w.Code, fields)
	}
}

func TestBodyNestingLimit(t *testing.T) {
	type DocIn struct {
		Data any    `json:"data"`
		Note string `json:"note"`
	}
	app := New(Config{})
	Post(app, "/docs", func(ctx context.Context, in DocIn) (Item, error) { return Item{}, nil })
	h := mustRegister(t, app)
	// The body's own object is the first level.
	nested := func(levels int) string {
		return `{"data":` + strings.Repeat("[", levels-1) + strings.Repeat("]", levels-1) + `}`
	}
	if w := send(t, h, "POST", "/api/docs", nested(64)); w.Code != 201 {
		t.Fatalf("64 levels: status %d, body %q", w.Code, w.Body.String())
	}
	mixed := `{"data":` + strings.Repeat(`[{"a":`, 32) + "1" + strings.Repeat("}]", 32) + `}`
	for _, body := range []string{nested(65), nested(1000), mixed} {
		w := send(t, h, "POST", "/api/docs", body)
		if code, msg, _ := apiError(t, w); w.Code != 400 || code != "bad_request" || msg != "The body is nested more than 64 levels deep." {
			t.Fatalf("%.40s…: status %d, code %q, message %q", body, w.Code, code, msg)
		}
	}
	// Brackets inside a string do not count, escaped quotes included.
	inString := `{"note":"` + strings.Repeat(`[{\"`, 100) + `\\","data":` + strings.Repeat("[", 62) + strings.Repeat("]", 62) + `}`
	if w := send(t, h, "POST", "/api/docs", inString); w.Code != 201 {
		t.Fatalf("brackets in a string: status %d, body %q", w.Code, w.Body.String())
	}
	if allocs := testing.AllocsPerRun(5, func() { tooDeep([]byte(inString), maxNesting) }); allocs != 0 {
		t.Fatalf("tooDeep made %.0f allocations", allocs)
	}
	// It comes after the media type is checked.
	w := send(t, h, "POST", "/api/docs", nested(65), "Content-Type", "text/plain")
	if w.Code != 415 {
		t.Fatalf("too deep as text/plain: status %d", w.Code)
	}
}

// A value that is valid JSON but that its field's own decoder refuses is not
// "not valid JSON".
func TestBodyValueItsFieldCannotTake(t *testing.T) {
	type EventIn struct {
		When time.Time `json:"when"`
		Data []byte    `json:"data"`
	}
	app := New(Config{})
	Post(app, "/events", func(ctx context.Context, in EventIn) (Item, error) { return Item{}, nil })
	h := mustRegister(t, app)
	for _, body := range []string{`{"when":"tomorrow"}`, `{"data":"!!!"}`} {
		w := send(t, h, "POST", "/api/events", body)
		if code, msg, fields := apiError(t, w); w.Code != 400 || code != "bad_request" || msg != "The body has a value its field cannot take." || fields != nil {
			t.Errorf("%s: status %d, code %q, message %q, fields %v", body, w.Code, code, msg, fields)
		}
	}
	if w := send(t, h, "POST", "/api/events", `{"when":"2026-11-25T10:00:00Z","data":"YWI="}`); w.Code != 201 {
		t.Fatalf("status %d, body %q", w.Code, w.Body.String())
	}
}

// bodyFields is the type of a body field in the tests below.
type bodyFields struct {
	Name  string  `json:"name" warp:"required"`
	Price float64 `json:"price" warp:"min=0"`
}

func TestBodyField(t *testing.T) {
	type UpdateIn struct {
		ID   int        `path:"id"`
		Page int        `query:"page"`
		Body bodyFields `body:"json"`
	}
	var got UpdateIn
	var sent []string
	var hasName, hasBody bool
	app := New(Config{})
	Put(app, "/items/{id}", func(ctx context.Context, in UpdateIn) (Item, error) {
		got, sent = in, SentKeys(ctx)
		hasName, hasBody = Sent(ctx, "name"), Sent(ctx, "body")
		return Item{ID: in.ID, Name: in.Body.Name}, nil
	})
	h := mustRegister(t, app)
	w := send(t, h, "PUT", "/api/items/3?page=2", `{"price":5,"name":"lamp"}`)
	if w.Code != 200 || w.Body.String() != `{"id":3,"name":"lamp"}`+"\n" {
		t.Fatalf("status %d, body %q", w.Code, w.Body.String())
	}
	if want := (UpdateIn{ID: 3, Page: 2, Body: bodyFields{Name: "lamp", Price: 5}}); got != want {
		t.Fatalf("input %+v, want %+v", got, want)
	}
	if !reflect.DeepEqual(sent, []string{"price", "name"}) || !hasName || hasBody {
		t.Fatalf("SentKeys %q, Sent(name) %v, Sent(body) %v", sent, hasName, hasBody)
	}
	for _, c := range []struct {
		body   string
		status int
		msg    string
		fields map[string]string
	}{
		{`{"price":-1}`, 422, "2 fields need attention.", map[string]string{"name": "This is required.", "price": "Enter 0 or more."}},
		{`{"name":"a","price":"x"}`, 400, "1 field has the wrong type.", map[string]string{"price": "Send a number."}},
		{`{"body":{"name":"a"}}`, 400, `The body has the field "body", which this route does not take.`, nil},
		{`{"name":"a","id":9}`, 400, `The body has the field "id", which this route does not take.`, nil},
		{`{"name":"a","page":9}`, 400, `The body has the field "page", which this route does not take.`, nil},
	} {
		w := send(t, h, "PUT", "/api/items/3", c.body)
		if _, msg, fields := apiError(t, w); w.Code != c.status || msg != c.msg || !reflect.DeepEqual(fields, c.fields) {
			t.Errorf("%s: status %d, message %q, fields %v", c.body, w.Code, msg, fields)
		}
	}
	// Validate called directly keys the body's fields without a prefix too.
	if got := fieldsOf(t, UpdateIn{}); !reflect.DeepEqual(got, map[string]string{"name": "This is required."}) {
		t.Fatalf("Validate: fields %v", got)
	}
}

// withID is the input a generic package writes: a struct may hold a field of
// a type parameter where it may not embed one.
type withID[T any] struct {
	ID   int64 `path:"id"`
	Body T     `body:"json"`
}

func TestBodyFieldGeneric(t *testing.T) {
	var got withID[bodyFields]
	app := New(Config{})
	Post(app, "/items/{id}", func(ctx context.Context, in withID[bodyFields]) (Item, error) {
		got = in
		return Item{}, nil
	})
	h := mustRegister(t, app)
	if w := send(t, h, "POST", "/api/items/8", `{"name":"lamp","price":2}`); w.Code != 201 {
		t.Fatalf("status %d, body %q", w.Code, w.Body.String())
	}
	if want := (withID[bodyFields]{ID: 8, Body: bodyFields{Name: "lamp", Price: 2}}); got != want {
		t.Fatalf("input %+v, want %+v", got, want)
	}
	w := send(t, h, "POST", "/api/items/8", `{}`)
	if _, _, fields := apiError(t, w); w.Code != 422 || !reflect.DeepEqual(fields, map[string]string{"name": "This is required."}) {
		t.Fatalf("status %d, fields %v", w.Code, fields)
	}
}

func TestBodyFieldMistakes(t *testing.T) {
	for _, c := range []struct {
		name     string
		register func(app *App)
		want     []string
	}{
		{"not a struct", func(app *App) {
			type UpdateIn struct {
				ID   int    `path:"id"`
				Body string `body:"json"`
			}
			Patch(app, "/products/{id}", func(context.Context, UpdateIn) (Item, error) { return Item{}, nil })
		}, []string{`warp: PATCH /products/{id}: UpdateIn.Body is tagged body:"json" and is a string; make it a struct.`}},
		{"a pointer, time and a type that decodes itself", func(app *App) {
			type UpdateIn struct {
				ID   int         `path:"id"`
				Body *bodyFields `body:"json"`
				When time.Time   `body:"json"`
				Cash Money       `body:"json"`
			}
			Patch(app, "/products/{id}", func(context.Context, UpdateIn) (Item, error) { return Item{}, nil })
		}, []string{
			`warp: PATCH /products/{id}: UpdateIn.Body is tagged body:"json" and is a *warp.bodyFields; make it a struct.`,
			`warp: PATCH /products/{id}: UpdateIn.When is tagged body:"json" and is a time.Time; make it a struct.`,
			`warp: PATCH /products/{id}: UpdateIn.Cash is tagged body:"json" and is a warp.Money; make it a struct.`,
		}},
		{"a json field beside it", func(app *App) {
			type UpdateIn struct {
				ID   int        `path:"id"`
				Name string     `json:"name"`
				Body bodyFields `body:"json"`
			}
			Patch(app, "/products/{id}", func(context.Context, UpdateIn) (Item, error) { return Item{}, nil })
		}, []string{`warp: PATCH /products/{id}: UpdateIn has a body field and the json field Name; move Name into the body's type.`}},
		{"two body fields", func(app *App) {
			type UpdateIn struct {
				ID    int        `path:"id"`
				Body  bodyFields `body:"json"`
				Other bodyFields `body:"json"`
			}
			Patch(app, "/products/{id}", func(context.Context, UpdateIn) (Item, error) { return Item{}, nil })
		}, []string{`warp: PATCH /products/{id}: UpdateIn has two body fields, Body and Other; keep one.`}},
		{"on GET", func(app *App) {
			type ShowIn struct {
				ID   int        `path:"id"`
				Body bodyFields `body:"json"`
			}
			Get(app, "/products/{id}", func(context.Context, ShowIn) (Item, error) { return Item{}, nil })
		}, []string{`warp: GET /products/{id}: ShowIn.Body is tagged body, and GET reads no body; remove it.`}},
		{"another kind", func(app *App) {
			type UpdateIn struct {
				ID   int        `path:"id"`
				Body bodyFields `body:"xml"`
			}
			Patch(app, "/products/{id}", func(context.Context, UpdateIn) (Item, error) { return Item{}, nil })
		}, []string{`warp: PATCH /products/{id}: UpdateIn.Body is tagged body:"xml"; the only body kind is json.`}},
		{"a body and a json tag", func(app *App) {
			type UpdateIn struct {
				ID   int        `path:"id"`
				Body bodyFields `body:"json" json:"body"`
			}
			Patch(app, "/products/{id}", func(context.Context, UpdateIn) (Item, error) { return Item{}, nil })
		}, []string{`warp: PATCH /products/{id}: UpdateIn.Body has both a body and a json tag; keep one.`}},
		{"inside the body", func(app *App) {
			type Address struct {
				City string     `json:"city"`
				Body bodyFields `body:"json"`
			}
			type Fields struct {
				Name    string  `json:"name"`
				Address Address `json:"address"`
			}
			type UpdateIn struct {
				ID   int    `path:"id"`
				Body Fields `body:"json"`
			}
			Patch(app, "/products/{id}", func(context.Context, UpdateIn) (Item, error) { return Item{}, nil })
		}, []string{`warp: PATCH /products/{id}: Address.Body is tagged body inside a body field; only the route's input itself has a body field.`}},
		{"not exported", func(app *App) {
			type UpdateIn struct {
				ID   int        `path:"id"`
				body bodyFields `body:"json"`
			}
			Patch(app, "/products/{id}", func(context.Context, UpdateIn) (Item, error) { return Item{}, nil })
		}, []string{`warp: PATCH /products/{id}: UpdateIn.body is tagged body and is not exported; export it.`}},
		{"on an embedded struct", func(app *App) {
			type Base struct {
				Name string `json:"name"`
			}
			type UpdateIn struct {
				ID   int `path:"id"`
				Base `body:"json"`
			}
			Patch(app, "/products/{id}", func(context.Context, UpdateIn) (Item, error) { return Item{}, nil })
		}, []string{`warp: PATCH /products/{id}: UpdateIn embeds Base tagged body; give the body field a name.`}},
		{"readonly", func(app *App) {
			type UpdateIn struct {
				ID   int        `path:"id"`
				Body bodyFields `body:"json" warp:"readonly"`
			}
			Patch(app, "/products/{id}", func(context.Context, UpdateIn) (Item, error) { return Item{}, nil })
		}, []string{`warp: PATCH /products/{id}: UpdateIn.Body is tagged body and readonly; a body is always read, so remove readonly.`}},
		{"the body's own fields are checked", func(app *App) {
			type Base struct {
				Name string `json:"name"`
			}
			type Fields struct {
				Base
				Title string `json:"name"`
				Bare  string
				Page  int `query:"page"`
			}
			type UpdateIn struct {
				ID   int    `path:"id"`
				Body Fields `body:"json"`
			}
			Patch(app, "/products/{id}", func(context.Context, UpdateIn) (Item, error) { return Item{}, nil })
		}, []string{
			`warp: PATCH /products/{id}: Fields.Name and Fields.Title both have the json key "name"; rename one.`,
			`warp: PATCH /products/{id}: Fields.Bare has no json tag; add one.`,
			`warp: PATCH /products/{id}: Fields.Page is tagged query inside a body field; only the route's input itself takes query and path tags.`,
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			app := New(Config{})
			c.register(app)
			if got := strings.Split(errText(app.Err()), "\n"); !reflect.DeepEqual(got, c.want) {
				t.Fatalf("Err() =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(c.want, "\n"))
			}
			if routes := app.Routes(); len(routes) != 0 {
				t.Fatalf("Routes() = %v, want none", routes)
			}
		})
	}
}

// The types of TestReadonly: a record's fields are answered and never read.
type roRecord struct {
	ID        int64     `json:"id" warp:"readonly"`
	CreatedAt time.Time `json:"created_at" warp:"readonly"`
}

type roPerson struct {
	Name string `json:"name" warp:"required"`
	Bare string
}

type roLine struct {
	ID  int    `json:"id" warp:"readonly"`
	SKU string `json:"sku" warp:"required"`
}

type roAddress struct {
	ID   int64  `json:"id" warp:"readonly"`
	City string `json:"city"`
}

type roProduct struct {
	roRecord
	Name    string             `json:"name" warp:"required"`
	Owner   roPerson           `json:"owner" warp:"readonly"`
	Address roAddress          `json:"address"`
	Lines   []roLine           `json:"lines"`
	ByName  map[string]*roLine `json:"by_name"`
	Any     any                `json:"any"`
	Where   *roAddress         `json:"where"`
}

func TestReadonly(t *testing.T) {
	var got roProduct
	app := New(Config{})
	Post(app, "/products", func(ctx context.Context, in struct {
		Body roProduct `body:"json"`
	}) (roProduct, error) {
		got = in.Body
		out := in.Body
		out.ID, out.Owner.Name = 7, "Ada"
		return out, nil
	})
	type PlainIn struct {
		ID   int    `json:"id" warp:"readonly"`
		Name string `json:"name"`
	}
	Post(app, "/plain", func(ctx context.Context, in PlainIn) (Item, error) { return Item{}, nil })
	h := mustRegister(t, app)

	// Answered: the output carries it. Not validated: Owner's required Name
	// is empty, and Owner's untagged field is no registration mistake.
	body := `{"name":"lamp","address":{"city":"Oslo"},"lines":[{"sku":"a"}],"by_name":{"x":{"sku":"b"},"y":null},"any":{"id":1}}`
	w := send(t, h, "POST", "/api/products", body)
	if w.Code != 201 || !strings.Contains(w.Body.String(), `"id":7,"created_at":"0001-01-01T00:00:00Z","name":"lamp","owner":{"name":"Ada","Bare":""}`) {
		t.Fatalf("status %d, body %q", w.Code, w.Body.String())
	}
	if got.Name != "lamp" || got.Address.City != "Oslo" || got.Lines[0].SKU != "a" || got.ByName["x"].SKU != "b" || got.ID != 0 {
		t.Fatalf("input %+v", got)
	}
	if err := Validate(roProduct{Name: "lamp"}); err != nil {
		t.Fatalf("Validate: %v", err)
	}

	// Refused in the body, at every depth, and whatever its case.
	for _, c := range []struct{ path, body, field string }{
		{"/api/products", `{"name":"a","id":9}`, "id"},
		{"/api/products", `{"ID":9}`, "ID"},
		{"/api/products", `{"created_at":"2026-11-25T10:00:00Z"}`, "created_at"},
		{"/api/products", `{"owner":{"name":"x"}}`, "owner"},
		{"/api/products", `{"name":"a","address":{"city":"Oslo","id":3}}`, "id"},
		{"/api/products", `{"address":{"Id":3}}`, "Id"},
		{"/api/products", `{"address":{"id":3}}`, "id"},
		{"/api/products", `{"address":{"ID":3}}`, "ID"},
		{"/api/products", `{"lines":[{"sku":"a"},{"sku":"b","id":1}]}`, "id"},
		{"/api/products", `{"by_name":{"x":{"id":1,"sku":"b"}}}`, "id"},
		{"/api/plain", `{"id":1}`, "id"},
		// Through a pointer to a struct.
		{"/api/products", `{"where":{"id":1}}`, "id"},
		{"/api/products", `{"where":{"city":"x","ID":1}}`, "ID"},
		// Sent with a type it does not take: still the field it does not take,
		// not the type it would need.
		{"/api/products", `{"address":{"id":"x"}}`, "id"},
		{"/api/products", `{"lines":[{"id":"x"}]}`, "id"},
		{"/api/products", `{"where":{"id":[1]}}`, "id"},
		{"/api/products", `{"by_name":{"x":{"ID":{}}}}`, "ID"},
		{"/api/products", `{"name":5,"address":{"id":"x"}}`, "id"},
	} {
		w := send(t, h, "POST", c.path, c.body)
		want := `The body has the field "` + c.field + `", which this route does not take.`
		if code, msg, _ := apiError(t, w); w.Code != 400 || code != "bad_request" || msg != want {
			t.Errorf("%s: status %d, code %q, message %q", c.body, w.Code, code, msg)
		}
	}
}

func TestReadonlyMistakes(t *testing.T) {
	for _, c := range []struct {
		name     string
		register func(app *App)
		want     string
	}{
		{"on a query field", func(app *App) {
			type ShowIn struct {
				ID   int `path:"id"`
				Page int `query:"page" warp:"readonly"`
			}
			Get(app, "/products/{id}", func(context.Context, ShowIn) (Item, error) { return Item{}, nil })
		}, `warp: GET /products/{id}: ShowIn.Page is tagged query and readonly; readonly is for json fields.`},
		{"on a path field", func(app *App) {
			type ShowIn struct {
				ID int `path:"id" warp:"readonly"`
			}
			Get(app, "/products/{id}", func(context.Context, ShowIn) (Item, error) { return Item{}, nil })
		}, `warp: GET /products/{id}: ShowIn.ID is tagged path and readonly; readonly is for json fields.`},
		{"beside other rules", func(app *App) {
			type Product struct {
				ID   int64  `json:"id" warp:"readonly,min=1"`
				Name string `json:"name"`
			}
			Post(app, "/products", func(context.Context, struct {
				Body Product `body:"json"`
			}) (Item, error) {
				return Item{}, nil
			})
		}, `warp: POST /products: Product.ID has the rule "readonly" beside other rules; a readonly field is never read, so remove them.`},
	} {
		t.Run(c.name, func(t *testing.T) {
			app := New(Config{})
			c.register(app)
			if got := errText(app.Err()); got != c.want {
				t.Fatalf("Err() = %q\nwant %q", got, c.want)
			}
		})
	}
}

func TestSentKeys(t *testing.T) {
	type Address struct {
		Zip string `json:"zip"`
	}
	type In struct {
		A       int     `json:"a"`
		B       int     `json:"b"`
		C       *int    `json:"c"`
		Address Address `json:"address"`
	}
	var sent []string
	var sentB, sentZip bool
	ran := false
	app := New(Config{})
	Post(app, "/things", func(ctx context.Context, in In) (Item, error) {
		ran = true
		sent, sentB, sentZip = SentKeys(ctx), Sent(ctx, "b"), Sent(ctx, "zip")
		return Item{}, nil
	})
	Get(app, "/things", func(ctx context.Context, in None) (Item, error) {
		ran = true
		sent, sentB = SentKeys(ctx), Sent(ctx, "b")
		return Item{}, nil
	})
	h := mustRegister(t, app)
	for _, c := range []struct {
		method, body string
		want         []string
		b            bool
	}{
		{"POST", `{"b":1,"address":{"zip":"x"},"a":2,"B":3,"c":null}`, []string{"b", "address", "a", "c"}, true},
		{"POST", `{"a":1,"a":2,"A":3}`, []string{"a"}, false},
		{"POST", `{"a":1, "b" : 2 }`, []string{"a", "b"}, true},
		{"POST", ``, nil, false},
		{"POST", `{}`, nil, false},
		{"GET", ``, nil, false},
	} {
		sent, sentB, sentZip, ran = nil, false, false, false
		if w := send(t, h, c.method, "/api/things", c.body); w.Code >= 300 || !ran {
			t.Fatalf("%s %s: status %d, body %q", c.method, c.body, w.Code, w.Body.String())
		}
		if !reflect.DeepEqual(sent, c.want) || sentB != c.b || sentZip {
			t.Errorf("%s %s: SentKeys %q, Sent(b) %v, Sent(zip) %v", c.method, c.body, sent, sentB, sentZip)
		}
	}
	// A key the route does not take is refused before the handler runs.
	ran = false
	if w := send(t, h, "POST", "/api/things", `{"a":1,"zz":2}`); w.Code != 400 || ran {
		t.Fatalf("an unknown key: status %d, ran %v", w.Code, ran)
	}
	if Sent(context.Background(), "a") || SentKeys(context.Background()) != nil {
		t.Fatal("outside a typed route, nothing was sent")
	}
}

// Finding the sent keys costs a few allocations, not some for each key.
func TestSentKeysCostLittle(t *testing.T) {
	type In struct {
		A int   `json:"a"`
		B []any `json:"b"`
		C any   `json:"c"`
	}
	p, mistakes := planFor(reflect.TypeFor[In](), "POST", "/x")
	if mistakes != nil {
		t.Fatal(mistakes)
	}
	body := "{" + strings.TrimSuffix(strings.Repeat(`"a":1,"b":[1,{"x":"y\"z"}],"c":{"k":[true,null]},`, 3000), ",") + "}"
	keys, err := p.walk([]byte(body))
	if err != nil || !reflect.DeepEqual(keys, []string{"a", "b", "c"}) {
		t.Fatalf("keys %q, err %v", keys, err)
	}
	if allocs := testing.AllocsPerRun(5, func() { p.walk([]byte(body)) }); allocs > 3 {
		t.Fatalf("finding the keys of 9000: %.0f allocations, want at most 3", allocs)
	}
}

func TestFilterMap(t *testing.T) {
	type Filters map[string]string
	type ListIn struct {
		Filter map[string]string `query:"filter"`
		Where  Filters           `query:"where"`
		Page   int               `query:"page"`
	}
	var got ListIn
	app := New(Config{})
	Get(app, "/items", func(ctx context.Context, in ListIn) (Item, error) {
		got = in
		return Item{}, nil
	})
	h := mustRegister(t, app)
	get := func(query string) *httptest.ResponseRecorder {
		t.Helper()
		got = ListIn{}
		return send(t, h, "GET", "/api/items?"+query, "")
	}
	if w := get("filter[status]=live&filter%5Bcat%5D=3&filter[status]=draft&where[size]=&page=2"); w.Code != 200 {
		t.Fatalf("status %d, body %q", w.Code, w.Body.String())
	}
	if want := (ListIn{Filter: map[string]string{"status": "live", "cat": "3"}, Where: Filters{"size": ""}, Page: 2}); !reflect.DeepEqual(got, want) {
		t.Fatalf("input %+v, want %+v", got, want)
	}
	// Keys that are not filter[name] are ignored, as other keys are.
	if w := get("filter[]=a&filter[a=b&filter=c&filter[a]b=d&filter[a][b]=e&filters[x]=f&page=1"); w.Code != 200 || got.Filter != nil {
		t.Fatalf("status %d, input %+v", w.Code, got)
	}
	keys := func(prefix string, n int) string {
		var parts []string
		for i := range n {
			parts = append(parts, fmt.Sprintf("%s[k%d]=v", prefix, i))
		}
		return strings.Join(parts, "&")
	}
	if w := get(keys("filter", 32) + "&" + keys("other", 40)); w.Code != 200 || len(got.Filter) != 32 {
		t.Fatalf("32 keys: status %d, %d keys", w.Code, len(got.Filter))
	}
	for _, c := range []struct{ query, msg string }{
		{keys("filter", 33), "The query has more filter[...] keys than a route takes (32)."},
		{keys("where", 33), "The query has more where[...] keys than a route takes (32)."},
		{"filter[" + strings.Repeat("k", 65) + "]=v", "A filter[...] key is longer than 64 bytes."},
		{"where[" + strings.Repeat("k", 65) + "]=v", "A where[...] key is longer than 64 bytes."},
	} {
		w := get(c.query)
		if code, msg, _ := apiError(t, w); w.Code != 400 || code != "bad_request" || msg != c.msg {
			t.Errorf("%.30s…: status %d, code %q, message %q", c.query, w.Code, code, msg)
		}
	}
	long := strings.Repeat("k", 64)
	if w := get("filter[" + long + "]=v"); w.Code != 200 || got.Filter[long] != "v" {
		t.Fatalf("a 64-byte key: status %d, input %+v", w.Code, got)
	}
}

func TestFilterMapMistake(t *testing.T) {
	type ListIn struct {
		Filter map[string]int    `query:"filter"`
		By     map[int]string    `query:"by"`
		ID     map[string]string `path:"id"`
	}
	app := New(Config{})
	Get(app, "/items/{id}", func(context.Context, ListIn) (Item, error) { return Item{}, nil })
	want := strings.Join([]string{
		"warp: GET /items/{id}: ListIn.Filter is a map[string]int; a filter map is a map[string]string.",
		"warp: GET /items/{id}: ListIn.By is a map[int]string; a filter map is a map[string]string.",
		"warp: GET /items/{id}: ListIn.ID is a map[string]string, which a query or path value cannot fill; use a string, a number or a bool.",
	}, "\n")
	if got := errText(app.Err()); got != want {
		t.Fatalf("Err() =\n%s\nwant\n%s", got, want)
	}
}

// Warp does not take ,string: a number is sent and read as a number, as the
// description says.
func TestJSONStringOption(t *testing.T) {
	const fix = ` has json:",string"; remove ,string, since Warp sends and reads a number as a number.`
	type Inner struct {
		N int `json:"n,omitempty,string"`
	}
	type ShowIn struct {
		Count int   `json:"count,string"`
		ID    int64 `json:"id,string" warp:"readonly"`
		Inner Inner `json:"inner"`
	}
	type BodyOf struct {
		Flag bool `json:"flag,string"`
	}
	type WithBody struct {
		Body BodyOf `body:"json"`
	}
	app := New(Config{})
	Post(app, "/show", func(context.Context, ShowIn) (None, error) { return None{}, nil })
	Put(app, "/body", func(context.Context, WithBody) (None, error) { return None{}, nil })
	want := strings.Join([]string{
		"warp: POST /show: ShowIn.Count" + fix,
		"warp: POST /show: ShowIn.ID" + fix,
		"warp: POST /show: Inner.N" + fix,
		"warp: PUT /body: BodyOf.Flag" + fix,
	}, "\n")
	if errText(app.Err()) != want {
		t.Fatalf("Err =\n%s\nwant\n%s", errText(app.Err()), want)
	}
	if len(app.Routes()) != 0 {
		t.Fatalf("routes %v registered", app.Routes())
	}
}

// zFan's two keys differ only in case, and the type holds itself through
// both: a body could make the scan read a value once for each, at every
// level. Two keys of one object that fold the same are a registration
// mistake.
type zFan struct {
	ID int   `json:"id" warp:"readonly"`
	A  *zFan `json:"ab"`
	AA *zFan `json:"AB"`
}

type FoldBase struct {
	Kind string `json:"kind"`
}

// zSelf decodes itself, with the usual alias, so Warp cannot see what its
// decoder sets.
type zSelf struct {
	ID   int    `json:"id" warp:"readonly"`
	Name string `json:"name"`
}

func (z *zSelf) UnmarshalJSON(b []byte) error {
	type alias zSelf
	return json.Unmarshal(b, (*alias)(z))
}

type zText struct {
	ID int `json:"id" warp:"readonly"`
}

func (z *zText) UnmarshalText(b []byte) error { return nil }

func TestBodyKeyMistakes(t *testing.T) {
	for _, c := range []struct {
		name     string
		register func(app *App)
		want     []string
	}{
		{"two keys that differ only in case, at the top", func(app *App) {
			type In struct {
				Ab int `json:"ab"`
				AB int `json:"AB"`
			}
			Post(app, "/x", func(context.Context, In) (None, error) { return None{}, nil })
		}, []string{`warp: POST /x: In has the fields Ab and AB, whose keys differ only in case; rename one.`}},
		{"in a body field's type, a readonly one among them", func(app *App) {
			type Top struct {
				ID    int `json:"id" warp:"readonly"`
				Ident int `json:"ID"`
			}
			Post(app, "/x", func(context.Context, struct {
				Body Top `body:"json"`
			}) (None, error) {
				return None{}, nil
			})
		}, []string{`warp: POST /x: Top has the fields ID and Ident, whose keys differ only in case; rename one.`}},
		{"nested, in a type that holds itself", func(app *App) {
			type In struct {
				Fan zFan `json:"fan"`
			}
			Post(app, "/x", func(context.Context, In) (None, error) { return None{}, nil })
		}, []string{`warp: POST /x: zFan has the fields A and AA, whose keys differ only in case; rename one.`}},
		{"promoted from an embedded struct", func(app *App) {
			type Line struct {
				FoldBase
				KIND string `json:"KIND"`
			}
			type In struct {
				Lines []Line `json:"lines"`
			}
			Post(app, "/x", func(context.Context, In) (None, error) { return None{}, nil })
		}, []string{`warp: POST /x: Line has the fields Kind and KIND, whose keys differ only in case; rename one.`}},
		{"by Unicode folding, k and the Kelvin sign", func(app *App) {
			type In struct {
				Kb int `json:"kb"`
				KB int `json:"Kb"`
			}
			Put(app, "/x", func(context.Context, In) (None, error) { return None{}, nil })
		}, []string{"warp: PUT /x: In has the fields Kb and KB, whose keys differ only in case; rename one."}},
		{"a readonly field inside a type that decodes itself", func(app *App) {
			type In struct {
				Self  zSelf  `json:"self"`
				Ptr   *zSelf `json:"ptr"`
				Text  zText  `json:"text"`
				Fine  zSelf  `json:"fine" warp:"readonly"` // never read: nothing to see
				Lines []struct {
					Deep map[string]zSelf `json:"deep"`
				} `json:"lines"`
			}
			Post(app, "/x", func(context.Context, In) (None, error) { return None{}, nil })
		}, []string{
			`warp: POST /x: zSelf decodes itself, so Warp cannot see what it sets, and zSelf.ID is tagged readonly; drop the tag or the type's own decoding.`,
			`warp: POST /x: zText decodes itself, so Warp cannot see what it sets, and zText.ID is tagged readonly; drop the tag or the type's own decoding.`,
		}},
		{"readonly on an embedded struct", func(app *App) {
			type In struct {
				FoldBase `warp:"readonly"`
				Name     string `json:"name"`
			}
			Post(app, "/x", func(context.Context, In) (None, error) { return None{}, nil })
		}, []string{`warp: POST /x: In embeds FoldBase tagged readonly; tag the fields inside it, or give the field a name.`}},
	} {
		t.Run(c.name, func(t *testing.T) {
			app := New(Config{})
			c.register(app)
			if got := strings.Split(errText(app.Err()), "\n"); !reflect.DeepEqual(got, c.want) {
				t.Fatalf("Err() =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(c.want, "\n"))
			}
		})
	}
}

// The types of TestBodyChecksIgnoreFieldOrder. rT has a readonly field and
// is used both plainly and inside types that decode themselves; rBad has
// ordinary mistakes and is used inside one too.
type rT struct {
	ID   int    `json:"id" warp:"readonly"`
	Name string `json:"name"`
}

type rBad struct {
	X     int `json:"x"`
	Y     int `json:"X"`
	NoTag int
}

// rNode holds itself and a readonly field.
type rNode struct {
	Mark string `json:"mark" warp:"readonly"`
	Next *rNode `json:"next"`
}

type rD struct {
	Inner rT `json:"inner"`
}

func (d *rD) UnmarshalJSON(b []byte) error {
	type alias rD
	return json.Unmarshal(b, (*alias)(d))
}

// rD2 is another type that decodes itself, holding the same rT.
type rD2 struct {
	Inner []rT `json:"inner"`
}

func (d *rD2) UnmarshalJSON(b []byte) error {
	type alias rD2
	return json.Unmarshal(b, (*alias)(d))
}

type rDBad struct {
	Inner rBad `json:"inner"`
}

func (d *rDBad) UnmarshalJSON(b []byte) error {
	type alias rDBad
	return json.Unmarshal(b, (*alias)(d))
}

type rDNode struct {
	Head rNode `json:"head"`
}

func (d *rDNode) UnmarshalJSON(b []byte) error {
	type alias rDNode
	return json.Unmarshal(b, (*alias)(d))
}

// What an input's fields find, whatever order they are in: each check runs
// on every type it is for, never skipped because another check met the type
// first. A seeded shuffle tries many orders.
func TestBodyChecksIgnoreFieldOrder(t *testing.T) {
	const ro = " decodes itself, so Warp cannot see what it sets, and "
	fields := []reflect.StructField{
		{Name: "A", Type: reflect.TypeFor[rT](), Tag: `json:"a"`},
		{Name: "B", Type: reflect.TypeFor[rD](), Tag: `json:"b"`},
		{Name: "C", Type: reflect.TypeFor[rD2](), Tag: `json:"c"`},
		{Name: "D", Type: reflect.TypeFor[rBad](), Tag: `json:"d"`},
		{Name: "E", Type: reflect.TypeFor[rDBad](), Tag: `json:"e"`},
		{Name: "F", Type: reflect.TypeFor[*rD](), Tag: `json:"f"`},
		{Name: "G", Type: reflect.TypeFor[[]rD2](), Tag: `json:"g"`},
		{Name: "H", Type: reflect.TypeFor[map[string]rDNode](), Tag: `json:"h"`},
	}
	want := []string{
		"warp: POST /x: rBad has the fields X and Y, whose keys differ only in case; rename one.",
		"warp: POST /x: rBad.NoTag has no json tag; add one.",
		"warp: POST /x: rD" + ro + "rT.ID is tagged readonly; drop the tag or the type's own decoding.",
		"warp: POST /x: rD2" + ro + "rT.ID is tagged readonly; drop the tag or the type's own decoding.",
		"warp: POST /x: rDNode" + ro + "rNode.Mark is tagged readonly; drop the tag or the type's own decoding.",
	}
	slices.Sort(want)
	mistakesOf := func(order []reflect.StructField) []string {
		in := reflect.StructOf(order)
		_, mistakes := planFor(in, "POST", "/x")
		var got []string
		for _, m := range mistakes {
			got = append(got, "warp: POST /x: "+m)
		}
		slices.Sort(got)
		return got
	}
	r := rand.New(rand.NewPCG(20261125, 3))
	for i := range 200 {
		order := slices.Clone(fields)
		if i > 0 {
			r.Shuffle(len(order), func(a, b int) { order[a], order[b] = order[b], order[a] })
		}
		var names []string
		for _, f := range order {
			names = append(names, f.Name)
		}
		if got := mistakesOf(order); !slices.Equal(got, want) {
			t.Fatalf("fields in the order %v:\n%s\nwant\n%s", names, strings.Join(got, "\n"), strings.Join(want, "\n"))
		}
	}
}

// A readonly field inside a type that decodes itself is refused when a
// plain use of its type comes first, so it never reaches a handler.
func TestReadonlyInsideADecoderAfterAPlainUse(t *testing.T) {
	for _, in := range []reflect.Type{
		reflect.TypeFor[struct {
			A rT `json:"a"`
			B rD `json:"b"`
		}](),
		reflect.TypeFor[struct {
			B rD `json:"b"`
			A rT `json:"a"`
		}](),
	} {
		_, mistakes := planFor(in, "POST", "/x")
		if want := "rD decodes itself, so Warp cannot see what it sets, and rT.ID is tagged readonly; drop the tag or the type's own decoding."; !slices.Equal(mistakes, []string{want}) {
			t.Errorf("%s: %q", in, mistakes)
		}
	}
}

// A type error under a map, whose key may hold dots, is named by the map's
// field, on both builds.
func TestTypeErrorUnderAMap(t *testing.T) {
	type Line struct {
		SKU string `json:"sku"`
	}
	type In struct {
		ByName map[string]Line `json:"by_name"`
	}
	app := New(Config{})
	Post(app, "/x", func(context.Context, In) (None, error) { return None{}, nil })
	h := mustRegister(t, app)
	for _, body := range []string{`{"by_name":{"a.b.c":{"sku":123}}}`, `{"by_name":{"a":{"sku":123}}}`} {
		w := send(t, h, "POST", "/api/x", body)
		if _, _, fields := apiError(t, w); w.Code != 400 || !reflect.DeepEqual(fields, map[string]string{"by_name": "Send text."}) {
			t.Errorf("%s: %d %v", body, w.Code, fields)
		}
	}
}

// null for a field that cannot be null (anything but a pointer, an interface
// or a type that reads its own JSON) is a field error, at any depth, on Post,
// Put and Patch alike: never sent, never written. A pointer takes null as
// before.
func TestBodyNullForAValue(t *testing.T) {
	type Geo struct {
		Lat  float64  `json:"lat"`
		Note *string  `json:"note"`
		Pin  *float64 `json:"pin"`
	}
	type Line struct {
		SKU string `json:"sku"`
	}
	type Body struct {
		Name  string            `json:"name"`
		Price float64           `json:"price"`
		Live  bool              `json:"live"`
		When  time.Time         `json:"when"`
		Tags  []string          `json:"tags"`
		Lines []Line            `json:"lines"`
		Ptrs  []*Line           `json:"ptrs"`
		By    map[string]Line   `json:"by"`
		ByPtr map[string]*Line  `json:"by_ptr"`
		Geo   Geo               `json:"geo"`
		Opt   *Geo              `json:"opt"`
		Any   any               `json:"any"`
		Raw   json.RawMessage   `json:"raw"`
		Bytes []byte            `json:"bytes"`
		Nums  map[string]int    `json:"nums"`
		Grid  [][]int           `json:"grid"`
		Deep  map[string][]Line `json:"deep"`
	}
	var ran bool
	var got Body
	var sent []string
	app := New(Config{})
	handle := func(ctx context.Context, in struct {
		ID   int  `path:"id"`
		Body Body `body:"json"`
	}) (None, error) {
		ran, got, sent = true, in.Body, SentKeys(ctx)
		return None{}, nil
	}
	Post(app, "/things/{id}", handle)
	Put(app, "/things/{id}", handle)
	Patch(app, "/things/{id}", handle)
	h := mustRegister(t, app)
	const notNull = "Send a value, not null."
	for _, method := range []string{"POST", "PUT", "PATCH"} {
		for _, c := range []struct {
			body   string
			fields map[string]string
		}{
			{`{"price":null}`, map[string]string{"price": notNull}},
			{`{"name":null,"live":null}`, map[string]string{"name": notNull, "live": notNull}},
			{`{"when":null}`, map[string]string{"when": notNull}},
			{`{"tags":null}`, map[string]string{"tags": notNull}},
			{`{"tags":["a",null]}`, map[string]string{"tags": notNull}},
			{`{"lines":[{"sku":"a"},null]}`, map[string]string{"lines": notNull}},
			{`{"lines":[{"sku":null}]}`, map[string]string{"lines.sku": notNull}},
			{`{"by":{"a.b":null}}`, map[string]string{"by": notNull}},
			{`{"by":{"x":{"sku":null}}}`, map[string]string{"by": notNull}},
			{`{"geo":null}`, map[string]string{"geo": notNull}},
			{`{"geo":{"lat":null}}`, map[string]string{"geo.lat": notNull}},
			{`{"opt":{"lat":null}}`, map[string]string{"opt.lat": notNull}},
			{`{"bytes":null}`, map[string]string{"bytes": notNull}},
			{`{"nums":{"a":null}}`, map[string]string{"nums": notNull}},
			{`{"grid":[[1,null]]}`, map[string]string{"grid": notNull}},
			{`{"deep":{"k":[null]}}`, map[string]string{"deep": notNull}},
			{`{"PRICE":null}`, map[string]string{"price": notNull}},
			{`{"price":1,"price":null}`, map[string]string{"price": notNull}},
		} {
			ran = false
			w := send(t, h, method, "/api/things/1", c.body)
			_, msg, fields := apiError(t, w)
			if w.Code != 400 || ran || !reflect.DeepEqual(fields, c.fields) || !strings.HasSuffix(msg, "the wrong type.") {
				t.Errorf("%s %s: %d %q %v, ran %v; want 400 with %v", method, c.body, w.Code, msg, fields, ran, c.fields)
			}
		}
		// What may be null takes it, and is sent.
		ran = false
		body := `{"opt":null,"ptrs":[null],"by_ptr":{"x":null},"any":null,"raw":null,"geo":{"lat":1,"note":null,"pin":null}}`
		if w := send(t, h, method, "/api/things/1", body); w.Code >= 300 || !ran {
			t.Fatalf("%s %s: %d %s", method, body, w.Code, w.Body.String())
		}
		if got.Opt != nil || got.Ptrs[0] != nil || got.ByPtr["x"] != nil || got.Any != nil || got.Geo.Note != nil || !slices.Equal(sent, []string{"opt", "ptrs", "by_ptr", "any", "raw", "geo"}) {
			t.Errorf("%s: input %+v, sent %q", method, got, sent)
		}
	}
	// A readonly key is still refused first.
	type RO struct {
		ID    int     `json:"id" warp:"readonly"`
		Price float64 `json:"price"`
	}
	app = New(Config{})
	Post(app, "/ro", func(ctx context.Context, in struct {
		Body RO `body:"json"`
	}) (None, error) {
		return None{}, nil
	})
	h = mustRegister(t, app)
	w := send(t, h, "POST", "/api/ro", `{"price":null,"id":1}`)
	if _, msg, _ := apiError(t, w); w.Code != 400 || msg != `The body has the field "id", which this route does not take.` {
		t.Errorf("readonly and null: %d %q", w.Code, msg)
	}
}

// A fixed-size array takes a list of exactly its size: a longer or shorter
// one is refused on its field, never cut or padded.
func TestBodyFixedArray(t *testing.T) {
	type In struct {
		RGB  [3]int    `json:"rgb"`
		Grid [2][2]int `json:"grid"`
		One  [1]string `json:"one"`
	}
	var got In
	app := New(Config{})
	Post(app, "/x", func(_ context.Context, in In) (None, error) { got = in; return None{}, nil })
	h := mustRegister(t, app)
	for body, fields := range map[string]map[string]string{
		`{"rgb":[1,2,3,4,5]}`:  {"rgb": "Send a list of 3 values."},
		`{"rgb":[1,2]}`:        {"rgb": "Send a list of 3 values."},
		`{"rgb":[]}`:           {"rgb": "Send a list of 3 values."},
		`{"grid":[[1,2],[3]]}`: {"grid": "Send a list of 2 values."},
		`{"one":["a","b"]}`:    {"one": "Send a list of 1 value."},
	} {
		w := send(t, h, "POST", "/api/x", body)
		if _, _, f := apiError(t, w); w.Code != 400 || !reflect.DeepEqual(f, fields) {
			t.Errorf("%s: %d %v, want %v", body, w.Code, f, fields)
		}
	}
	if w := send(t, h, "POST", "/api/x", `{"rgb":[1,2,3],"grid":[[1,2],[3,4]],"one":["a"]}`); w.Code != 204 || got.RGB != [3]int{1, 2, 3} || got.Grid[1][1] != 4 {
		t.Fatalf("%d %s, %+v", w.Code, w.Body, got)
	}
}
