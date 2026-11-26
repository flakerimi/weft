package dbtest

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/flakerimi/weft/warp"
	"github.com/flakerimi/weft/warp/data"
	"github.com/flakerimi/weft/warp/warptest"
)

// resourceModule is the module products: a store of T on the table products
// and its resource at /products.
type resourceModule[T any] struct {
	store **data.Store[T]
	opts  []data.ResourceOption
}

func (m resourceModule[T]) Name() string { return "products" }

func (m resourceModule[T]) Register(app *warp.App) error {
	*m.store = data.NewStore[T](data.For(app), "products")
	data.Resource(app, "/products", *m.store, m.opts...)
	return nil
}

// withResource starts an app with data and the products module over a
// database of the test's own.
func withResource[T any](t *testing.T, opts ...data.ResourceOption) (*warptest.Client, *data.Store[T]) {
	t.Helper()
	var s *data.Store[T]
	c := warptest.New(t, env(dbFile(t)), data.Module(), resourceModule[T]{&s, opts})
	return c, s
}

// moduleFunc is a module named name whose Register runs fn.
type moduleFunc struct {
	name string
	fn   func(app *warp.App)
}

func (m moduleFunc) Name() string { return m.name }

func (m moduleFunc) Register(app *warp.App) error {
	m.fn(app)
	return nil
}

func TestResourceRoutes(t *testing.T) {
	c, _ := withResource[Product](t)
	type route struct {
		Module, Name, Method, Path string
		Out                        reflect.Type
		Status                     int
	}
	var got []route
	for _, r := range c.App.Routes() {
		got = append(got, route{r.Module, r.Name, r.Method, r.Path, r.Out, r.Status})
	}
	product := reflect.TypeFor[Product]()
	want := []route{
		{"products", "list", "GET", "/api/products", reflect.TypeFor[data.List[Product]](), 200},
		{"products", "show", "GET", "/api/products/{id}", product, 200},
		{"products", "create", "POST", "/api/products", product, 201},
		{"products", "update", "PATCH", "/api/products/{id}", product, 200},
		{"products", "remove", "DELETE", "/api/products/{id}", reflect.TypeFor[warp.None](), 204},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("routes\n%+v\nwant\n%+v", got, want)
	}
	routes := c.App.Routes()
	in := routes[3].In
	if routes[0].In != reflect.TypeFor[data.Query]() || routes[1].In != reflect.TypeFor[data.ID]() ||
		routes[2].In != product || routes[4].In != reflect.TypeFor[data.ID]() ||
		in.NumField() != 2 || in.Field(0).Tag.Get("path") != "id" || in.Field(1).Type != product || in.Field(1).Tag.Get("body") != "json" {
		t.Fatalf("inputs %v %v %v %v %v", routes[0].In, routes[1].In, routes[2].In, in, routes[4].In)
	}
	// The generated client reads them from the description: create and
	// update take the model as the body, update in part.
	desc, err := c.App.Describe()
	if err != nil {
		t.Fatalf("Describe = %v", err)
	}
	type described struct {
		Name, Body string
		Partial    bool
		Output     string
		Path       []string
		Query      []string
	}
	var gotDesc []described
	for _, r := range desc.Routes {
		d := described{Name: r.Name, Body: r.Body, Partial: r.Partial}
		if r.Output != nil {
			d.Output = r.Output.TS
		}
		for _, p := range r.PathParams {
			d.Path = append(d.Path, p.Name)
		}
		for _, q := range r.Query {
			d.Query = append(d.Query, q.Name)
		}
		gotDesc = append(gotDesc, d)
	}
	wantDesc := []described{
		{Name: "list", Output: "ListProduct", Query: []string{"page", "per_page", "sort", "q", "filter"}},
		{Name: "show", Output: "Product", Path: []string{"id"}},
		{Name: "create", Body: "ProductInput", Output: "Product"},
		{Name: "update", Body: "ProductInput", Partial: true, Output: "Product", Path: []string{"id"}},
		{Name: "remove", Path: []string{"id"}},
	}
	if !reflect.DeepEqual(gotDesc, wantDesc) {
		t.Fatalf("described\n%+v\nwant\n%+v", gotDesc, wantDesc)
	}

	// Each one, over HTTP.
	var made Product
	c.Post("/api/products", map[string]any{"name": "Lamp", "price": 12.5, "status": "live"}).Want(201).JSON(&made)
	if made.ID != 1 || made.Name != "Lamp" || made.Price != 12.5 || made.Status != "live" || made.CreatedAt.IsZero() {
		t.Fatalf("create answered %+v", made)
	}
	var shown Product
	c.Get("/api/products/1").Want(200).JSON(&shown)
	if !reflect.DeepEqual(shown, made) {
		t.Fatalf("show answered %+v, want %+v", shown, made)
	}
	if e := c.Get("/api/products/999").Want(404).Error(); e.Message != "No product has id 999." {
		t.Fatalf("show of a missing id: %+v", e)
	}
	c.Get("/api/products/lamp").Want(404)
	var l data.List[Product]
	c.Get("/api/products").Want(200).JSON(&l)
	if l.Total != 1 || len(l.Items) != 1 || !reflect.DeepEqual(l.Items[0], made) {
		t.Fatalf("list answered %+v", l)
	}
	var updated Product
	c.Patch("/api/products/1", map[string]any{"status": "draft"}).Want(200).JSON(&updated)
	if updated.Status != "draft" || updated.Name != "Lamp" {
		t.Fatalf("update answered %+v", updated)
	}
	if res := c.Delete("/api/products/1").Want(204); len(res.Body) != 0 {
		t.Fatalf("remove answered a body: %s", res.Body)
	}
	c.Get("/api/products/1").Want(404)
	c.Delete("/api/products/1").Want(404)
	c.Patch("/api/products/1", map[string]any{"price": 1}).Want(404)
	c.Patch("/api/products/1", map[string]any{}).Want(404)
}

// Review Focus 2: an update that sends one field.
func TestResourcePatchWritesSentFields(t *testing.T) {
	c, s := withResource[Product](t)
	clk := &clock{time.Date(2026, 11, 25, 9, 0, 0, 0, time.UTC)}
	data.For(c.App).SetClock(clk.Now)
	var lamp, desk Product
	c.Post("/api/products", map[string]any{"name": "Lamp", "price": 12.5, "status": "live"}).Want(201).JSON(&lamp)
	c.Post("/api/products", map[string]any{"name": "Desk", "price": 80, "status": "draft"}).Want(201).JSON(&desk)
	unchanged := func(want Product) {
		t.Helper()
		if got, err := s.Get(ctx, want.ID); err != nil || !reflect.DeepEqual(got, want) {
			t.Fatalf("stored %+v, %v; want %+v", got, err, want)
		}
	}

	// 1. One field sent: that field changes, and updated_at, and nothing else.
	clk.now = clk.now.Add(time.Hour)
	var got Product
	c.Patch("/api/products/1", map[string]any{"price": 5}).Want(200).JSON(&got)
	want := lamp
	want.Price, want.UpdatedAt = 5, clk.now
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("PATCH price answered\n%+v\nwant\n%+v", got, want)
	}
	unchanged(want)
	lamp = want

	// 2. A field sent empty is checked whole: required fails.
	if e := c.Patch("/api/products/1", map[string]any{"name": ""}).Want(422).Error(); !reflect.DeepEqual(e.Fields, map[string]string{"name": "This is required."}) {
		t.Fatalf("PATCH name empty: %+v", e)
	}
	for body, field := range map[string]string{`{"price": -1}`: "price", `{"status": "gone"}`: "status", `{"name": "` + strings.Repeat("n", 121) + `"}`: "name"} {
		if e := c.Patch("/api/products/1", body).Want(422).Error(); len(e.Fields) != 1 || e.Fields[field] == "" {
			t.Errorf("PATCH %s: %+v", body, e)
		}
	}
	unchanged(lamp)

	// 3. What the store sets is never read.
	for _, key := range []string{"id", "created_at", "updated_at"} {
		body := map[string]any{key: 9, "price": 1}
		if key != "id" {
			body[key] = "2000-01-01T00:00:00Z"
		}
		e := c.Patch("/api/products/1", body).Want(400).Error()
		if e.Message != fmt.Sprintf("The body has the field %q, which this route does not take.", key) {
			t.Errorf("PATCH %s: %+v", key, e)
		}
	}
	unchanged(lamp)

	// 4. A name another record has is a 422 on name.
	if e := c.Patch("/api/products/1", map[string]any{"name": "Desk"}).Want(422).Error(); !reflect.DeepEqual(e.Fields, map[string]string{"name": "This is already taken."}) {
		t.Fatalf("PATCH a taken name: %+v", e)
	}
	unchanged(lamp)
	unchanged(desk)

	// Nothing sent: the record as it is, updated_at untouched.
	clk.now = clk.now.Add(time.Hour)
	c.Patch("/api/products/1", map[string]any{}).Want(200).JSON(&got)
	if !reflect.DeepEqual(got, lamp) {
		t.Fatalf("PATCH {} answered %+v, want %+v", got, lamp)
	}
	unchanged(lamp)
	// Its own name again is no violation; two fields write two.
	c.Patch("/api/products/1", map[string]any{"name": "Lamp", "status": "draft"}).Want(200).JSON(&got)
	if got.Name != "Lamp" || got.Status != "draft" || got.Price != 5 || got.UpdatedAt != clk.now {
		t.Fatalf("PATCH two fields answered %+v", got)
	}
}

func TestResourceOptions(t *testing.T) {
	mark := func(tag string) warp.RouteOption {
		return warp.With(func(h http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Add("X-Mark", tag)
				h.ServeHTTP(w, r)
			})
		})
	}
	refuse := warp.With(func(http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			warp.WriteError(w, r, warp.Forbidden("Not you."))
		})
	})
	var calls []string
	c, _ := withResource[Product](t,
		data.Each(warp.Permission("products:any"), mark("each")),
		data.PerAction(func(action string) []warp.RouteOption {
			calls = append(calls, action)
			switch action {
			case "remove":
				return []warp.RouteOption{warp.Permission("products:delete"), mark("remove"), refuse}
			case "list":
				return []warp.RouteOption{warp.Name("everything")} // the names are the resource's
			}
			return nil
		}),
		data.Each(mark("each2")),
		data.PerAction(func(action string) []warp.RouteOption {
			if action == "create" {
				return []warp.RouteOption{warp.Permission("products:create")}
			}
			return nil
		}),
		data.PerAction(nil),
	)
	if want := []string{"list", "show", "create", "update", "remove"}; !reflect.DeepEqual(calls, want) {
		t.Fatalf("PerAction was called with %v, want %v", calls, want)
	}
	perms := map[string]string{}
	for _, r := range c.App.Routes() {
		perms[r.Name] = r.Permission
	}
	if want := map[string]string{"list": "products:any", "show": "products:any", "create": "products:create", "update": "products:any", "remove": "products:delete"}; !reflect.DeepEqual(perms, want) {
		t.Fatalf("permissions %v, want %v", perms, want)
	}
	res := c.Post("/api/products", map[string]any{"name": "Lamp"}).Want(201)
	if got := res.Header.Values("X-Mark"); !reflect.DeepEqual(got, []string{"each", "each2"}) {
		t.Fatalf("create's marks %v", got)
	}
	for _, res := range []*warptest.Response{c.Get("/api/products"), c.Get("/api/products/1"), c.Patch("/api/products/1", map[string]any{})} {
		if got := res.Want(200).Header.Values("X-Mark"); !reflect.DeepEqual(got, []string{"each", "each2"}) {
			t.Fatalf("marks %v", got)
		}
	}
	res = c.Delete("/api/products/1").Want(403)
	if got := res.Header.Values("X-Mark"); !reflect.DeepEqual(got, []string{"each", "each2", "remove"}) {
		t.Fatalf("remove's marks %v", got)
	}
	c.Get("/api/products/1").Want(200)
}

// Priced checks itself across fields.
type Priced struct {
	data.Record
	Name  string  `json:"name" warp:"required"`
	Price float64 `json:"price" warp:"min=0"`
	Sale  float64 `json:"sale"`
}

func (p Priced) Validate(ctx context.Context) error {
	if p.Sale > p.Price {
		return warp.Invalid(map[string]string{"sale": "Make the sale price no more than the price."})
	}
	return nil
}

func TestResourceValidates(t *testing.T) {
	c, s := withResource[Priced](t)
	if e := c.Post("/api/products", map[string]any{"price": 3}).Want(422).Error(); !reflect.DeepEqual(e.Fields, map[string]string{"name": "This is required."}) {
		t.Fatalf("create without a name: %+v", e)
	}
	if e := c.Post("/api/products", map[string]any{"name": "Lamp", "price": 3, "sale": 5}).Want(422).Error(); !reflect.DeepEqual(e.Fields, map[string]string{"sale": "Make the sale price no more than the price."}) {
		t.Fatalf("create with a sale above the price: %+v", e)
	}
	if e := c.Post("/api/products", map[string]any{"name": "Lamp", "id": 4}).Want(400).Error(); e.Message != `The body has the field "id", which this route does not take.` {
		t.Fatalf("create with an id: %+v", e)
	}
	var lamp Priced
	c.Post("/api/products", map[string]any{"name": "Lamp", "price": 5, "sale": 3}).Want(201).JSON(&lamp)

	if e := c.Patch("/api/products/1", map[string]any{"price": 10, "sale": 20}).Want(422).Error(); !reflect.DeepEqual(e.Fields, map[string]string{"sale": "Make the sale price no more than the price."}) {
		t.Fatalf("update with a sale above the price: %+v", e)
	}
	if got, _ := s.Get(ctx, 1); !reflect.DeepEqual(got, lamp) {
		t.Fatalf("a refused update wrote %+v", got)
	}
	var got Priced
	c.Patch("/api/products/1", map[string]any{"price": 10, "sale": 4}).Want(200).JSON(&got)
	if got.Price != 10 || got.Sale != 4 || got.Name != "Lamp" {
		t.Fatalf("update answered %+v", got)
	}
}

func TestResourceMistakes(t *testing.T) {
	// mistakes registers data and a module running fn, and returns the app's
	// mistakes as lines and its routes.
	mistakes := func(fn func(app *warp.App)) ([]string, []warp.Route) {
		app := warp.New(warp.ConfigFrom(nil))
		app.Use(data.Module(), moduleFunc{"products", fn})
		var lines []string
		if err := app.Err(); err != nil {
			lines = strings.Split(err.Error(), "\n")
		}
		return lines, app.Routes()
	}
	for _, path := range []string{"/products/", "products", "", "/", "/products/{id}", "/products/{$}", "/a//b", "/a b", "GET /products", "/a/../b", "/a/./b"} {
		got, routes := mistakes(func(app *warp.App) {
			data.Resource(app, path, data.NewStore[Product](data.For(app), "products"))
		})
		want := []string{fmt.Sprintf("warp: data.Resource was given the path %q; write it like /products.", path)}
		if !reflect.DeepEqual(got, want) || len(routes) != 0 {
			t.Errorf("path %q: %q, %d routes; want %q", path, got, len(routes), want)
		}
	}
	for _, path := range []string{"/products", "/shop/order-items", "/v2/things_1"} {
		got, routes := mistakes(func(app *warp.App) {
			data.Resource(app, path, data.NewStore[Product](data.For(app), "products"))
		})
		if len(got) != 0 || len(routes) != 5 || routes[0].Path != "/api"+path || routes[1].Path != "/api"+path+"/{id}" {
			t.Errorf("path %q: %q, %v", path, got, routes)
		}
	}

	got, _ := mistakes(func(app *warp.App) { data.Resource(app, "/products", (*data.Store[Product])(nil)) })
	if want := []string{`warp: data.Resource was given a nil store for /products; make one with data.NewStore in the module's Register.`}; !reflect.DeepEqual(got, want) {
		t.Errorf("nil store: %q", got)
	}

	// Twice on one path: the core's own sentences, no panic, none of data's.
	got, routes := mistakes(func(app *warp.App) {
		s := data.NewStore[Product](data.For(app), "products")
		data.Resource(app, "/products", s)
		data.Resource(app, "/products", s)
	})
	if len(got) == 0 || len(routes) != 5 || slices.ContainsFunc(got, func(l string) bool { return strings.Contains(l, "data.") }) {
		t.Errorf("twice: %q, %d routes", got, len(routes))
	}

	// A model the store refuses: the store's sentence alone, and no panic.
	type Plain struct {
		Name string `json:"name"`
	}
	got, _ = mistakes(func(app *warp.App) {
		data.Resource(app, "/plain", data.NewStore[Plain](data.For(app), "plain"))
	})
	if want := []string{`warp: plain: Plain does not embed data.Record; embed it, since a store needs an id and two times.`}; !reflect.DeepEqual(got, want) {
		t.Errorf("a refused model: %q", got)
	}
}
