package warp

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"
)

// The types of the description's tests.

type descRecord struct {
	ID        int64     `json:"id" warp:"readonly"`
	CreatedAt time.Time `json:"created_at" warp:"readonly"`
}

// Gadget is a model: it embeds a record, so a body of it has a twin.
type Gadget struct {
	descRecord
	Name   string       `json:"name" warp:"required,max=120,unique"`
	Status string       `json:"status" warp:"enum=draft|live"`
	Email  string       `json:"email" warp:"email"`
	Site   *string      `json:"site" warp:"url"`
	Price  float64      `json:"price" warp:"min=0"`
	Parts  []GadgetPart `json:"parts"`
}

// GadgetPart has a readonly field and holds itself.
type GadgetPart struct {
	ID     int64       `json:"id" warp:"readonly"`
	SKU    string      `json:"sku" warp:"required"`
	Parent *GadgetPart `json:"parent"`
}

type descList[T any] struct {
	Items []T `json:"items"`
	Total int `json:"total"`
}

type gadgetFind struct {
	ID     int64             `path:"id"`
	Fields []string          `query:"fields"`
	Full   *bool             `query:"full"`
	Filter map[string]string `query:"filter"`
	Page   int               `query:"page" warp:"min=1"`
}

type withShop[T any] struct {
	Shop string `path:"shop"`
	Body T      `body:"json"`
}

type gadgetID struct {
	ID int64 `path:"id"`
}

type gadgetNote struct {
	ID   int64  `path:"id"`
	Text string `json:"text" warp:"required"`
}

func noOut[In any](context.Context, In) (None, error) { return None{}, nil }

func gadgetOut[In any](context.Context, In) (Gadget, error) { return Gadget{}, nil }

// gadgetApp is an app with a module whose routes cover each kind of route.
func gadgetApp() *App {
	app := New(Config{})
	app.Use(inModule("gadgets", func(app *App) {
		Get(app, "/gadgets/{id}", gadgetOut[gadgetFind], Name("show"), Permission("gadgets:read"))
		Post(app, "/shops/{shop}/gadgets", gadgetOut[withShop[Gadget]])
		Patch(app, "/gadgets/{id}", gadgetOut[withID[Gadget]])
		Delete(app, "/gadgets/{id}", noOut[gadgetID])
		Post(app, "/gadgets/{id}/notes", noOut[gadgetNote], Status(202))
		Get(app, "/gadgets", func(context.Context, None) ([]Gadget, error) { return nil, nil })
		app.Handle("GET /gadgets/export.csv", text("csv"), Permission("gadgets:read"))
	}))
	return app
}

func describe(t *testing.T, app *App) Description {
	t.Helper()
	if err := app.Err(); err != nil {
		t.Fatal(err)
	}
	d, err := app.Describe()
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func r(name, arg string) RuleInfo { return RuleInfo{Name: name, Arg: arg} }

func TestDescribeRoutes(t *testing.T) {
	none := []RuleInfo{}
	idParam := []ParamInfo{{Name: "id", TS: "number", Style: "value", Rules: none}}
	got := describe(t, gadgetApp()).Routes
	want := []RouteInfo{
		{
			Module: "gadgets", Name: "show", Method: "GET", Path: "/api/gadgets/{id}", Status: 200, Permission: "gadgets:read", Typed: true,
			PathParams: idParam,
			Query: []ParamInfo{
				{Name: "fields", TS: "string[]", Optional: true, Style: "list", Rules: none},
				{Name: "full", TS: "boolean", Optional: true, Style: "value", Rules: none},
				{Name: "filter", TS: "Record<string, string>", Optional: true, Style: "filter", Rules: none},
				{Name: "page", TS: "number", Optional: true, Style: "value", Rules: []RuleInfo{r("min", "1")}},
			},
			Output: &OutputInfo{TS: "Gadget"},
		},
		{
			Module: "gadgets", Name: "post_shops_gadgets", Method: "POST", Path: "/api/shops/{shop}/gadgets", Status: 201, Typed: true,
			PathParams: []ParamInfo{{Name: "shop", TS: "string", Style: "value", Rules: none}},
			Query:      []ParamInfo{},
			Body:       "GadgetInput",
			Output:     &OutputInfo{TS: "Gadget"},
		},
		{
			Module: "gadgets", Name: "patch_gadgets", Method: "PATCH", Path: "/api/gadgets/{id}", Status: 200, Typed: true,
			PathParams: idParam, Query: []ParamInfo{},
			Body: "GadgetInput", Partial: true,
			Output: &OutputInfo{TS: "Gadget"},
		},
		{
			Module: "gadgets", Name: "delete_gadgets", Method: "DELETE", Path: "/api/gadgets/{id}", Status: 204, Typed: true,
			PathParams: idParam, Query: []ParamInfo{},
		},
		{
			Module: "gadgets", Name: "post_gadgets_notes", Method: "POST", Path: "/api/gadgets/{id}/notes", Status: 204, Typed: true,
			PathParams: idParam, Query: []ParamInfo{},
			Body: "gadgetNote",
		},
		{
			Module: "gadgets", Name: "gadgets", Method: "GET", Path: "/api/gadgets", Status: 200, Typed: true,
			PathParams: []ParamInfo{}, Query: []ParamInfo{},
			Output: &OutputInfo{TS: "Gadget", List: true},
		},
		{
			Module: "gadgets", Name: "gadgets_export.csv", Method: "GET", Path: "/api/gadgets/export.csv", Permission: "gadgets:read",
			PathParams: []ParamInfo{}, Query: []ParamInfo{},
		},
	}
	if len(got) != len(want) {
		t.Fatalf("%d routes, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if !reflect.DeepEqual(got[i], want[i]) {
			g, _ := json.Marshal(got[i])
			w, _ := json.Marshal(want[i])
			t.Errorf("route %d:\n got %s\nwant %s", i, g, w)
		}
	}
	// The note's body is the input itself, its json fields alone.
	for _, ti := range describe(t, gadgetApp()).Types {
		if ti.Name == "gadgetNote" {
			want := []FieldInfo{{Name: "text", TS: "string", Rules: []RuleInfo{r("required", "")}}}
			if !reflect.DeepEqual(ti.Fields, want) {
				t.Errorf("gadgetNote fields %+v, want %+v", ti.Fields, want)
			}
		}
	}
}

type descStatus string

type descLeaf struct {
	N int `json:"n"`
}

// descKinds has a field of each kind the TypeScript table names.
type descKinds struct {
	S          string              `json:"s"`
	I          int                 `json:"i"`
	I8         int8                `json:"i8"`
	U16        uint16              `json:"u16"`
	F32        float32             `json:"f32"`
	B          bool                `json:"b"`
	T          time.Time           `json:"t"`
	Raw        []byte              `json:"raw"`
	Ints       []int               `json:"ints"`
	Pair       [2]int              `json:"pair"`
	Flags      map[string]bool     `json:"flags"`
	Leaves     map[string]descLeaf `json:"leaves"`
	Ptr        *string             `json:"ptr"`
	Leaf       descLeaf            `json:"leaf"`
	MaybeLeafs []*descLeaf         `json:"maybe_leafs"`
	Any        any                 `json:"any"`
	Self       *descKinds          `json:"self"`
	Status     descStatus          `json:"status"`
	Untagged   int
	Skipped    int `json:"-"`
	hidden     int
}

func TestDescribeTypes(t *testing.T) {
	app := New(Config{})
	Get(app, "/kinds", func(context.Context, None) (descKinds, error) { return descKinds{}, nil })
	Post(app, "/gadgets", func(context.Context, Gadget) (Gadget, error) { return Gadget{}, nil })
	Get(app, "/lists", func(context.Context, None) (descList[Gadget], error) { return descList[Gadget]{}, nil })
	d := describe(t, app)

	none := []RuleInfo{}
	f := func(name, ts string) FieldInfo { return FieldInfo{Name: name, TS: ts, Rules: none} }
	opt := func(fi FieldInfo) FieldInfo { fi.Optional = true; return fi }
	ro := func(fi FieldInfo) FieldInfo { fi.Readonly = true; return fi }
	rules := func(fi FieldInfo, rs ...RuleInfo) FieldInfo { fi.Rules = rs; return fi }
	const pkg = "github.com/flakerimi/weft/warp."
	gadgetFields := []FieldInfo{
		rules(f("name", "string"), r("required", ""), r("max", "120"), r("unique", "")),
		rules(f("status", "string"), r("enum", "draft|live")),
		rules(f("email", "string"), r("email", "")),
		rules(opt(f("site", "string | null")), r("url", "")),
		rules(f("price", "number"), r("min", "0")),
	}
	want := []TypeInfo{
		{Name: "descKinds", Go: pkg + "descKinds", Fields: []FieldInfo{
			f("s", "string"), f("i", "number"), f("i8", "number"), f("u16", "number"), f("f32", "number"),
			f("b", "boolean"), f("t", "string"), f("raw", "string"), f("ints", "number[]"), f("pair", "number[]"),
			f("flags", "Record<string, boolean>"), f("leaves", "Record<string, descLeaf>"),
			opt(f("ptr", "string | null")), f("leaf", "descLeaf"), f("maybe_leafs", "(descLeaf | null)[]"),
			f("any", "unknown"), opt(f("self", "descKinds | null")), f("status", "string"), f("Untagged", "number"),
		}},
		{Name: "descLeaf", Go: pkg + "descLeaf", Fields: []FieldInfo{f("n", "number")}},
		{Name: "Gadget", Go: pkg + "Gadget", Fields: append([]FieldInfo{
			ro(f("id", "number")), ro(f("created_at", "string")),
		}, append(gadgetFields, f("parts", "GadgetPart[]"))...)},
		{Name: "GadgetInput", Go: pkg + "Gadget", Fields: append(gadgetFields[:len(gadgetFields):len(gadgetFields)], f("parts", "GadgetPartInput[]"))},
		{Name: "GadgetPart", Go: pkg + "GadgetPart", Fields: []FieldInfo{
			ro(f("id", "number")), rules(f("sku", "string"), r("required", "")), opt(f("parent", "GadgetPart | null")),
		}},
		{Name: "GadgetPartInput", Go: pkg + "GadgetPart", Fields: []FieldInfo{
			rules(f("sku", "string"), r("required", "")), opt(f("parent", "GadgetPartInput | null")),
		}},
		{Name: "descListGadget", Go: pkg + "descList[" + pkg + "Gadget]", Fields: []FieldInfo{
			f("items", "Gadget[]"), f("total", "number"),
		}},
	}
	if len(d.Types) != len(want) {
		var names []string
		for _, ti := range d.Types {
			names = append(names, ti.Name)
		}
		t.Fatalf("types %v, want %d", names, len(want))
	}
	for i := range want {
		if !reflect.DeepEqual(d.Types[i], want[i]) {
			g, _ := json.Marshal(d.Types[i])
			w, _ := json.Marshal(want[i])
			t.Errorf("type %d:\n got %s\nwant %s", i, g, w)
		}
	}
	if b := d.Routes[1].Body; b != "GadgetInput" {
		t.Errorf("POST /gadgets body %q, want GadgetInput", b)
	}
}

// A body type with no readonly field anywhere has no twin.
func TestDescribeNoTwin(t *testing.T) {
	app := New(Config{})
	Post(app, "/shops/{shop}/leaves", func(context.Context, withShop[descLeaf]) (descLeaf, error) { return descLeaf{}, nil })
	d := describe(t, app)
	if d.Routes[0].Body != "descLeaf" || len(d.Types) != 1 || d.Types[0].Name != "descLeaf" {
		t.Fatalf("body %q, types %+v", d.Routes[0].Body, d.Types)
	}
	// Gadget answered alone gets no twin either.
	app = New(Config{})
	Get(app, "/gadgets", func(context.Context, None) (Gadget, error) { return Gadget{}, nil })
	for _, ti := range describe(t, app).Types {
		if strings.HasSuffix(ti.Name, "Input") {
			t.Fatalf("%s described for a type no route takes", ti.Name)
		}
	}
}

// URL clashes with net/url.URL.
type URL struct {
	Link url.URL `json:"link"`
}

type descNote struct {
	ID int64 `json:"id" warp:"readonly"`
}

// descNoteInput clashes with descNote's twin.
type descNoteInput struct {
	Text string `json:"text"`
}

func TestDescribeNameClash(t *testing.T) {
	app := New(Config{})
	Get(app, "/links", func(context.Context, None) (URL, error) { return URL{}, nil })
	_, err := app.Describe()
	want := `warp: two types are named "URL" (github.com/flakerimi/weft/warp.URL and net/url.URL); rename one, since the generated TypeScript has one name for each.`
	if errText(err) != want {
		t.Fatalf("Describe = %q, want %q", errText(err), want)
	}

	app = New(Config{})
	Post(app, "/notes", func(context.Context, descNote) (descNoteInput, error) { return descNoteInput{}, nil })
	_, err = app.Describe()
	want = `warp: two types are named "descNoteInput" (the input of github.com/flakerimi/weft/warp.descNote and github.com/flakerimi/weft/warp.descNoteInput); rename one, since the generated TypeScript has one name for each.`
	if errText(err) != want {
		t.Fatalf("Describe = %q, want %q", errText(err), want)
	}
}

type descHolder struct {
	Meta struct {
		N int `json:"n"`
	} `json:"meta"`
	Ch chan int `json:"ch"`
}

func TestDescribeUnnamed(t *testing.T) {
	type wantErr struct {
		register func(*App)
		want     string
	}
	for _, c := range []wantErr{
		{func(app *App) {
			Get(app, "/products", func(context.Context, None) (struct{ N int }, error) { return struct{ N int }{}, nil })
		}, "warp: GET /products answers an unnamed struct; give the type a name, since the generated TypeScript needs one."},
		{func(app *App) {
			Get(app, "/products", func(context.Context, None) ([]struct{ N int }, error) { return nil, nil })
		}, "warp: GET /products answers an unnamed struct; give the type a name, since the generated TypeScript needs one."},
		{func(app *App) {
			Post(app, "/products", func(context.Context, struct {
				N int `json:"n"`
			}) (None, error) {
				return None{}, nil
			})
		}, "warp: POST /products takes an unnamed struct; give the type a name, since the generated TypeScript needs one."},
		{func(app *App) {
			Get(app, "/holders", func(context.Context, None) (descHolder, error) { return descHolder{}, nil })
		}, "warp: descHolder.Meta is an unnamed struct; give the type a name, since the generated TypeScript needs one.\n" +
			"warp: descHolder.Ch is a chan int, which JSON cannot carry; give it another type, or tag it json:\"-\"."},
	} {
		app := New(Config{})
		c.register(app)
		if err := app.Err(); err != nil {
			t.Fatal(err)
		}
		if _, err := app.Describe(); errText(err) != c.want {
			t.Errorf("Describe = %q, want %q", errText(err), c.want)
		}
	}
	// An input whose fields are all query and path fields needs no name.
	app := New(Config{})
	Get(app, "/products/{id}", func(context.Context, struct {
		ID int `path:"id"`
	}) (None, error) {
		return None{}, nil
	})
	describe(t, app)
}

func TestDescribeIsStable(t *testing.T) {
	app := gadgetApp()
	Get(app, "/kinds", func(context.Context, None) (descKinds, error) { return descKinds{}, nil })
	encode := func() []byte {
		b, err := json.MarshalIndent(describe(t, app), "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	first := encode()
	for range 20 {
		if again := encode(); !bytes.Equal(first, again) {
			t.Fatalf("two descriptions differ:\n%s\n%s", first, again)
		}
	}
	// A second app made the same way describes itself the same way, and
	// starting changes nothing.
	other := gadgetApp()
	Get(other, "/kinds", func(context.Context, None) (descKinds, error) { return descKinds{}, nil })
	if err := other.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer other.Stop(context.Background())
	b, _ := json.MarshalIndent(describe(t, other), "", "  ")
	if !bytes.Equal(first, b) {
		t.Fatalf("another app describes itself otherwise:\n%s\n%s", first, b)
	}
	// Empty lists are written as [], never null.
	if bytes.Contains(first, []byte(`"rules": null`)) || bytes.Contains(first, []byte(`"query": null`)) {
		t.Fatalf("a list is written as null:\n%s", first)
	}
}

// describeTo replaces where Serve writes a description for the rest of the
// test.
func describeTo(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	old := describeOut
	describeOut = &buf
	t.Cleanup(func() { describeOut = old })
	t.Setenv("WARP_DESCRIBE", "1")
	return &buf
}

func TestServeDescribes(t *testing.T) {
	out := describeTo(t)
	// A port that is taken: listening would fail.
	ln := listenLocal(t)
	defer ln.Close()
	_, port, _ := net.SplitHostPort(ln.Addr().String())
	var log []string
	app := New(ConfigFrom(map[string]string{"HOST": "127.0.0.1", "PORT": port}))
	m := mod("gadgets", &log)
	m.register = func(app *App) {
		Get(app, "/gadgets", func(context.Context, None) ([]Gadget, error) { return nil, nil })
		Get(app, "/gadgets/{id}", gadgetOut[gadgetFind], Name("show"))
	}
	app.Use(m)
	if err := app.Serve(nil); err != nil {
		t.Fatalf("Serve = %v", err)
	}
	if got := notes(&log); !reflect.DeepEqual(got, []string{"register:gadgets"}) {
		t.Fatalf("steps %v: Serve migrated or started", got)
	}
	var d Description
	if err := json.Unmarshal(out.Bytes(), &d); err != nil {
		t.Fatalf("Serve printed %q: %v", out, err)
	}
	if len(d.Routes) != 2 || d.Routes[0].Path != "/api/gadgets" || len(d.Types) != 2 {
		t.Fatalf("description %+v", d)
	}
	if !strings.Contains(out.String(), `"ts": "Record<string, string>"`) {
		t.Fatalf("Serve printed %s, want < and > as they are", out)
	}
	if !strings.HasPrefix(out.String(), "{\n  \"routes\"") || !strings.HasSuffix(out.String(), "}\n") {
		t.Fatalf("Serve printed %q, want indented JSON and a newline", out)
	}
	// The app was never started, so it still can be.
	if err := app.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	app.Stop(context.Background())
}

// The configuration's problems do not stop a description: it needs routes
// and types, not a working environment.
func TestServeDescribesDespiteConfig(t *testing.T) {
	out := describeTo(t)
	app := New(ConfigFrom(map[string]string{"PORT": "nope", "TRUSTED_ORIGINS": "not an origin"}))
	Get(app, "/gadgets", gadgetOut[None])
	if app.Err() == nil || !strings.Contains(app.Err().Error(), `PORT is "nope"`) {
		t.Fatalf("Err = %v, want the bad PORT", app.Err())
	}
	if err := app.Serve(nil); err != nil {
		t.Fatalf("Serve = %v", err)
	}
	if !json.Valid(out.Bytes()) || out.Len() == 0 {
		t.Fatalf("Serve printed %q", out)
	}
	// A bad origin given in a Config by hand is the configuration's too.
	out.Reset()
	app = New(Config{TrustedOrigins: []string{"not an origin"}})
	if app.Err() == nil {
		t.Fatal("Err = nil, want the bad origin")
	}
	if err := app.Serve(nil); err != nil || !json.Valid(out.Bytes()) {
		t.Fatalf("Serve = %v, printed %q", err, out)
	}
}

func TestServeDescribeReportsMistakes(t *testing.T) {
	out := describeTo(t)
	app := New(ConfigFrom(map[string]string{"PORT": "nope"}))
	Get(app, "gadgets", gadgetOut[None])
	err := app.Serve(nil)
	if want := `warp: route "GET gadgets" has a path that does not start with /; write it as "GET /gadgets".`; errText(err) != want {
		t.Fatalf("Serve = %q, want %q", errText(err), want)
	}
	if out.Len() != 0 {
		t.Fatalf("Serve printed %q", out)
	}
	// Describe's own mistakes are returned too.
	app = New(Config{})
	Get(app, "/products", func(context.Context, None) (struct{ N int }, error) { return struct{ N int }{}, nil })
	if err := app.Serve(nil); err == nil || !strings.Contains(err.Error(), "unnamed struct") {
		t.Fatalf("Serve = %v, want the unnamed struct", err)
	}
	if out.Len() != 0 {
		t.Fatalf("Serve printed %q", out)
	}
}

func TestDescribeGenericNames(t *testing.T) {
	for in, want := range map[string]string{
		"Product":                             "Product",
		"List[main.Product]":                  "ListProduct",
		"List[github.com/shop/app.Item]":      "ListItem",
		"List[gopkg.in/yaml.v3.Node]":         "ListNode",
		"List[[]*main.Product]":               "ListProductList",
		"List[[3]int]":                        "ListIntList",
		"List[interface {}]":                  "ListAny",
		"Page[map[string]int,main.Order]":     "PageMapStringIntOrder",
		"Pair[main.List[net/url.URL],string]": "PairListURLString",
	} {
		if got := genericName(in); got != want {
			t.Errorf("genericName(%q) = %q, want %q", in, got, want)
		}
	}
}

type descBase struct {
	Name  string `json:"name"`
	Shade string `json:"shade"`
}

// descOwn writes itself as text, and descRaw as JSON.
type descOwn struct{ v string }

func (o descOwn) MarshalText() ([]byte, error) { return []byte(o.v), nil }

type descRaw struct{ v int }

func (r descRaw) MarshalJSON() ([]byte, error) { return []byte("1"), nil }

// descEncodings is answered as encoding/json writes it.
type descEncodings struct {
	*descBase                 // flattened, as a value would be
	Shade     int             `json:"shade"` // shallower than descBase's, so it wins
	Own       descOwn         `json:"own"`
	Raw       descRaw         `json:"raw"`
	Money     Money           `json:"money"`
	Code      Code            `json:"code"`
	Number    json.Number     `json:"number"`
	Message   json.RawMessage `json:"message"`
	Page      int             `query:"page"` // a route's, not the JSON's
	descLeaf  `json:"leaf"`   // named by its tag, so a field of its own
}

func TestDescribeEncodings(t *testing.T) {
	app := New(Config{})
	Get(app, "/encodings", func(context.Context, None) (descEncodings, error) { return descEncodings{}, nil })
	d := describe(t, app)
	var got []string
	for _, f := range d.Types[0].Fields {
		got = append(got, f.Name+": "+f.TS)
	}
	want := []string{"name: string", "shade: number", "own: string", "raw: unknown", "money: unknown", "code: string", "number: number", "message: unknown", "leaf: descLeaf"}
	if !reflect.DeepEqual(got, want) || len(d.Types) != 2 {
		t.Fatalf("fields %q, want %q; types %+v", got, want, d.Types)
	}
}

// descOrder has no readonly field of its own, and holds parts that do.
type descOrder struct {
	Note  string       `json:"note"`
	Parts []GadgetPart `json:"parts"`
}

func TestDescribeNestedTwin(t *testing.T) {
	app := New(Config{})
	Post(app, "/orders", func(context.Context, descOrder) (None, error) { return None{}, nil })
	d := describe(t, app)
	var got []string
	for _, ti := range d.Types {
		got = append(got, ti.Name)
	}
	if want := []string{"descOrder", "descOrderInput", "GadgetPart", "GadgetPartInput"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("types %v, want %v", got, want)
	}
	if d.Routes[0].Body != "descOrderInput" || d.Types[1].Fields[1].TS != "GadgetPartInput[]" {
		t.Fatalf("body %q, twin %+v", d.Routes[0].Body, d.Types[1])
	}
}
