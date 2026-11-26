package warp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"
)

// nnLeaf holds one of each kind that encoding/json writes as null when nil.
type nnLeaf struct {
	Tags []string       `json:"tags"`
	Raw  []byte         `json:"raw"`
	Meta map[string]int `json:"meta"`
}

// NNBase is embedded, so its fields are the answer's own.
type NNBase struct {
	Labels []string `json:"labels"`
}

// nnNode reaches nnLeaf every way an answer can, and holds itself.
type nnNode struct {
	NNBase
	Leaf     nnLeaf            `json:"leaf"`
	Ptr      *nnLeaf           `json:"ptr"`
	NilPtr   *nnLeaf           `json:"nil_ptr"`
	List     []nnLeaf          `json:"list"`
	Dict     map[string]nnLeaf `json:"dict"`
	Arr      [1]nnLeaf         `json:"arr"`
	PtrList  *[]string         `json:"ptr_list"`
	Omit     []string          `json:"omit,omitempty"`
	Zero     []string          `json:"zero,omitzero"`
	Any      any               `json:"any"`
	Hidden   []string          `json:"-"`
	Children []nnNode          `json:"children"`
	Next     *nnNode           `json:"next"`
}

func answer(t *testing.T, out any) string {
	t.Helper()
	app := New(Config{})
	typ := reflect.TypeOf(out)
	if typ == nil {
		t.Fatal("answer needs a typed value")
	}
	switch v := out.(type) {
	case nnNode:
		Get(app, "/x", func(context.Context, None) (nnNode, error) { return v, nil })
	case *nnNode:
		Get(app, "/x", func(context.Context, None) (*nnNode, error) { return v, nil })
	case []nnLeaf:
		Get(app, "/x", func(context.Context, None) ([]nnLeaf, error) { return v, nil })
	case map[string][]string:
		Get(app, "/x", func(context.Context, None) (map[string][]string, error) { return v, nil })
	case []byte:
		Get(app, "/x", func(context.Context, None) ([]byte, error) { return v, nil })
	default:
		t.Fatalf("answer cannot register a %T", out)
	}
	w := send(t, mustRegister(t, app), "GET", "/api/x", "")
	if w.Code != http.StatusOK {
		t.Fatalf("status %d, body %s", w.Code, w.Body)
	}
	return w.Body.String()
}

// A typed route never answers null for a list, a map or bytes, however deep;
// a nil pointer stays null, as the description says (T | null).
func TestAnswerHasNoNullLists(t *testing.T) {
	var nilList []string
	shared := &nnLeaf{}
	node := nnNode{
		Ptr:     shared,
		List:    []nnLeaf{{}},
		Dict:    map[string]nnLeaf{"a": {}},
		PtrList: &nilList,
		Children: []nnNode{{
			Next: &nnNode{},
		}},
	}
	leaf := `{"tags":[],"raw":"","meta":{}}`
	inner := `{"labels":[],"leaf":` + leaf + `,"ptr":null,"nil_ptr":null,"list":[],"dict":{},"arr":[` + leaf + `],"ptr_list":null,"any":null,"children":[],"next":null}`
	child := `{"labels":[],"leaf":` + leaf + `,"ptr":null,"nil_ptr":null,"list":[],"dict":{},"arr":[` + leaf + `],"ptr_list":null,"any":null,"children":[],"next":` + inner + `}`
	want := `{"labels":[],"leaf":` + leaf + `,"ptr":` + leaf + `,"nil_ptr":null,"list":[` + leaf + `],"dict":{"a":` + leaf + `},"arr":[` + leaf + `],"ptr_list":[],"any":null,"children":[` + child + `],"next":null}` + "\n"
	if got := answer(t, node); got != want {
		t.Fatalf("body\n%s\nwant\n%s", got, want)
	}
	// What the handler returned is not changed: the fix-up is made on copies.
	if shared.Tags != nil || nilList != nil || node.List[0].Tags != nil || node.Dict["a"].Meta != nil || node.Children[0].Next.Labels != nil {
		t.Fatalf("the handler's values were changed: %+v", node)
	}

	for _, c := range []struct {
		name string
		out  any
		want string
	}{
		{"a nil pointer to a struct", (*nnNode)(nil), "null\n"},
		{"a pointer to a struct", &nnNode{}, `{"labels":[],"leaf":` + leaf + `,"ptr":null,"nil_ptr":null,"list":[],"dict":{},"arr":[` + leaf + `],"ptr_list":null,"any":null,"children":[],"next":null}` + "\n"},
		{"a nil list", []nnLeaf(nil), "[]\n"},
		{"a list", []nnLeaf{{}}, "[" + leaf + "]\n"},
		{"a nil map", map[string][]string(nil), "{}\n"},
		{"a map of nil lists", map[string][]string{"a": nil}, `{"a":[]}` + "\n"},
		{"nil bytes", []byte(nil), `""` + "\n"},
	} {
		if got := answer(t, c.out); got != c.want {
			t.Errorf("%s: body %s, want %s", c.name, got, c.want)
		}
	}
}

// WriteJSON called by a plain handler writes what encoding/json writes, but
// for a nil list at the top.
func TestWriteJSONKeepsNullInside(t *testing.T) {
	w := send(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		WriteJSON(w, 200, nnLeaf{})
	}), "GET", "/", "")
	if got := w.Body.String(); got != `{"tags":null,"raw":null,"meta":null}`+"\n" {
		t.Fatalf("body %s", got)
	}
}

// A value that holds itself through a pointer is not followed for ever: the
// fix-up stops, and encoding/json answers its cycle as the 500 it always was.
func TestAnswerThatHoldsItselfEnds(t *testing.T) {
	logs := captureLog(t)
	loop := &nnNode{}
	loop.Next = loop
	app := New(Config{})
	Get(app, "/loop", func(context.Context, None) (*nnNode, error) { return loop, nil })
	w := send(t, mustRegister(t, app), "GET", "/api/loop", "")
	if w.Code != http.StatusInternalServerError || !strings.Contains(logs.String(), "cycle") {
		t.Fatalf("status %d, log %q", w.Code, logs)
	}
}

// flatOut holds nothing that can be a null list: it has no plan, so its
// answer costs what it did before the fix-up existed.
type flatOut struct {
	ID    int64     `json:"id"`
	Name  string    `json:"name"`
	At    time.Time `json:"at"`
	Owner *flatOut  `json:"owner"`
	Any   any       `json:"any"`
}

// selfWritten writes itself, so what is inside it is its own affair.
type selfWritten []string

func (s selfWritten) MarshalJSON() ([]byte, error) { return []byte(`"x"`), nil }

func (s *selfWritten) UnmarshalJSON([]byte) error { return nil }

func TestEmptyPlanCostsNothingWhenNotNeeded(t *testing.T) {
	for _, typ := range []reflect.Type{
		reflect.TypeFor[flatOut](), reflect.TypeFor[[]flatOut](), reflect.TypeFor[*flatOut](),
		reflect.TypeFor[selfWritten](), reflect.TypeFor[struct {
			S selfWritten `json:"s"`
			H []string    `json:"-"`
			h []string
		}](),
		reflect.TypeFor[None](), reflect.TypeFor[int](),
	} {
		if p := emptyPlanFor(typ); p != nil {
			t.Errorf("%s has a plan", typ)
		}
	}
	// A type that has lists but answers none nil is not copied: the fix-up
	// looks and allocates nothing but the one value a map's entries are read
	// into, for each map whose values it looks inside.
	full := nnLeaf{Tags: []string{"a"}, Raw: []byte("b"), Meta: map[string]int{"c": 1}}
	node := nnNode{
		NNBase: NNBase{Labels: []string{}}, Leaf: full, Ptr: &full, List: []nnLeaf{full}, Dict: map[string]nnLeaf{"a": full},
		Arr: [1]nnLeaf{full}, Children: []nnNode{}, Omit: nil,
	}
	p := emptyPlanFor(reflect.TypeFor[nnNode]())
	rv := reflect.ValueOf(&node).Elem()
	if allocs := testing.AllocsPerRun(20, func() { p.apply(rv, true, 0) }); allocs != 1 {
		t.Errorf("a full answer cost %v allocations to look at, want 1 (for its one map)", allocs)
	}
	node.Dict = map[string]nnLeaf{}
	if allocs := testing.AllocsPerRun(20, func() { p.apply(rv, true, 0) }); allocs != 1 {
		t.Errorf("a full answer with an empty map cost %v allocations to look at, want 1", allocs)
	}
	node.Dict = nil
	node.Dict = map[string]nnLeaf{"a": full}
	node.List = nil
	if v, changed := p.apply(rv, true, 0); !changed || v.Interface().(nnNode).List == nil {
		t.Errorf("the nil list was not made empty")
	}
	// One that has nil lists costs one allocation for each it makes empty,
	// and a copy of each shared thing that holds one.
	leaf := emptyPlanFor(reflect.TypeFor[nnLeaf]())
	var nilLeaf nnLeaf
	lv := reflect.ValueOf(&nilLeaf).Elem()
	if allocs := testing.AllocsPerRun(20, func() { nilLeaf = nnLeaf{}; leaf.apply(lv, true, 0) }); allocs > 3 {
		t.Errorf("three nil fields cost %v allocations, want at most 3", allocs)
	}
}

// A typed route whose answer has no plan costs what the plain WriteJSON of
// that answer costs, plus the route's own work, the same as on a route that
// answers nothing: the fix-up adds nothing to it, not even the answer's
// address.
func TestAnswerWithoutPlanCostsNothingExtra(t *testing.T) {
	flat := flatOut{ID: 1, Name: "a", At: time.Unix(0, 0).UTC()}
	app := New(Config{})
	Get(app, "/typed", func(context.Context, None) (flatOut, error) { return flat, nil })
	Get(app, "/typed-none", func(context.Context, None) (None, error) { return None{}, nil })
	app.Handle("GET /plain", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { WriteJSON(w, 200, flat) }))
	app.Handle("GET /plain-none", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) }))
	h := mustRegister(t, app)
	cost := func(path string) float64 {
		r := httptest.NewRequest("GET", path, nil)
		w := &discard{h: http.Header{}}
		return testing.AllocsPerRun(100, func() { h.ServeHTTP(w, r) })
	}
	answered := cost("/api/typed") - cost("/api/plain")
	none := cost("/api/typed-none") - cost("/api/plain-none")
	if answered != none {
		t.Fatalf("a typed route costs %v allocations over WriteJSON when it answers flatOut, and %v when it answers nothing", answered, none)
	}
}
