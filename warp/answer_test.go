package warp

import (
	"context"
	"encoding/json"
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
		if p, _ := emptyPlanFor(typ); p != nil {
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
	p, _ := emptyPlanFor(reflect.TypeFor[nnNode]())
	rv := reflect.ValueOf(&node).Elem()
	look := func() { p.apply(rv, true, &emptyWalk{}) }
	if allocs := testing.AllocsPerRun(20, look); allocs != 1 {
		t.Errorf("a full answer cost %v allocations to look at, want 1 (for its one map)", allocs)
	}
	node.Dict = map[string]nnLeaf{}
	if allocs := testing.AllocsPerRun(20, look); allocs != 0 {
		t.Errorf("a full answer with an empty map cost %v allocations to look at, want 0", allocs)
	}
	node.Dict = map[string]nnLeaf{"a": full}
	node.List = nil
	if v, changed := p.apply(rv, true, &emptyWalk{}); !changed || v.Interface().(nnNode).List == nil {
		t.Errorf("the nil list was not made empty")
	}
	// Nil lists in the answer's own value cost nothing: the empty values
	// written are the plan's own.
	leaf, _ := emptyPlanFor(reflect.TypeFor[nnLeaf]())
	var nilLeaf nnLeaf
	lv := reflect.ValueOf(&nilLeaf).Elem()
	if allocs := testing.AllocsPerRun(20, func() { nilLeaf = nnLeaf{}; leaf.apply(lv, true, &emptyWalk{}) }); allocs != 0 {
		t.Errorf("three nil fields cost %v allocations, want 0", allocs)
	}
}

// pRow is a row of a long answer, its lists nil.
type pRow struct {
	ID   int64             `json:"id"`
	Name string            `json:"name"`
	Tags []string          `json:"tags"`
	Meta map[string]string `json:"meta"`
}

// A long list of rows with nil lists costs a copy of the list and of the
// first row that changes, not some for each row: once the list is copied,
// its rows are the fix-up's own and change in place.
func TestEmptyPlanCostPerRow(t *testing.T) {
	rows := make([]pRow, 10_000)
	p, _ := emptyPlanFor(reflect.TypeFor[[]pRow]())
	allocs := testing.AllocsPerRun(5, func() { emptied(p, rows) })
	t.Logf("10 000 rows with nil lists: %.0f allocations", allocs)
	if allocs > 5 {
		t.Fatalf("10 000 rows cost %.0f allocations, want at most 5", allocs)
	}
	if rows[0].Tags != nil || rows[9999].Meta != nil {
		t.Fatal("the handler's rows were changed")
	}
	out := emptied(p, rows)
	if out[0].Tags == nil || out[9999].Meta == nil {
		t.Fatal("the rows were not fixed")
	}
}

// The fix-up never changes whether encoding/json leaves a field out: a zero
// struct or array tagged omitzero stays zero, so it is still left out, and
// an omitempty list stays empty.
type pOmit struct {
	Inner  nnLeaf      `json:"inner,omitzero"`
	Arr    [1]nnLeaf   `json:"arr,omitzero"`
	Ptr    *nnLeaf     `json:"ptr,omitzero"`
	When   zeroWhen    `json:"when,omitzero"`
	Empty  []string    `json:"empty,omitempty"`
	Set    nnLeaf      `json:"set,omitzero"`
	Lists  [][]string  `json:"lists,omitempty"`
	Name   string      `json:"name"`
	Nested nnLeafOmits `json:"nested"`
}

type nnLeafOmits struct {
	Tags []string `json:"tags,omitzero"`
	Meta map[string]int
}

// zeroWhen says it is zero when its list is empty, which reflect would not.
type zeroWhen struct {
	Days []int `json:"days"`
}

func (z zeroWhen) IsZero() bool { return len(z.Days) == 0 }

func TestAnswerKeepsOmitzero(t *testing.T) {
	v := pOmit{Set: nnLeaf{Tags: []string{"x"}}, When: zeroWhen{Days: []int{}}, Lists: [][]string{}}
	want, _ := json.Marshal(v)
	app := New(Config{})
	Get(app, "/x", func(context.Context, None) (pOmit, error) { return v, nil })
	got := send(t, mustRegister(t, app), "GET", "/api/x", "").Body.String()
	// encoding/json writes set's nil lists, and nested's, as null; the route
	// writes them empty, and leaves out exactly what encoding/json leaves out.
	wantFixed := `{"set":{"tags":["x"],"raw":"","meta":{}},"name":"","nested":{"Meta":{}}}` + "\n"
	if got != wantFixed {
		t.Fatalf("body %s, want %s (encoding/json: %s)", got, wantFixed, want)
	}
}

// base is embedded unexported: encoding/json writes its fields as the
// outer type's, Describe types them so, and the fix-up fills them too.
type base struct {
	Labels []string `json:"labels"`
	inner
}

type inner struct {
	Codes map[string]int `json:"codes"`
}

type withBase struct {
	base
	Name string `json:"name"`
}

func TestAnswerFillsUnexportedEmbedded(t *testing.T) {
	shared := &withBase{}
	app := New(Config{})
	Get(app, "/x", func(context.Context, None) (withBase, error) { return withBase{}, nil })
	Get(app, "/list", func(context.Context, None) ([]withBase, error) { return []withBase{{}, {Name: "b"}}, nil })
	Get(app, "/ptr", func(context.Context, None) (*withBase, error) { return shared, nil })
	h := mustRegister(t, app)
	for path, want := range map[string]string{
		"/api/x":    `{"labels":[],"codes":{},"name":""}`,
		"/api/list": `[{"labels":[],"codes":{},"name":""},{"labels":[],"codes":{},"name":"b"}]`,
		"/api/ptr":  `{"labels":[],"codes":{},"name":""}`,
	} {
		if got := strings.TrimSpace(send(t, h, "GET", path, "").Body.String()); got != want {
			t.Errorf("%s: %s, want %s", path, got, want)
		}
	}
	if shared.Labels != nil || shared.Codes != nil {
		t.Fatal("the shared value was changed")
	}
	d, err := app.Describe()
	if err != nil {
		t.Fatal(err)
	}
	for _, ty := range d.Types {
		if ty.Name == "withBase" {
			for _, f := range ty.Fields {
				if f.Name != "name" && (strings.Contains(f.TS, "null") || f.Optional) {
					t.Errorf("Describe says %s is %s, optional %v", f.Name, f.TS, f.Optional)
				}
			}
		}
	}
}

// pBase is embedded through an unexported pointer: its lists cannot be
// replaced by copies, so such an answer is a registration mistake.
type pBase struct {
	Labels []string `json:"labels"`
}

type withPtrBase struct {
	*pBase
	Name string `json:"name"`
}

func TestAnswerRefusesUnexportedEmbeddedPointer(t *testing.T) {
	app := New(Config{})
	Get(app, "/x", func(context.Context, None) (withPtrBase, error) { return withPtrBase{}, nil })
	want := `warp: GET /x: withPtrBase embeds *pBase, a pointer to an unexported struct with lists in it, which Warp cannot write as [] when they are nil; embed pBase by value.`
	if errText(app.Err()) != want {
		t.Fatalf("Err() = %q\nwant %q", errText(app.Err()), want)
	}
}

// pTwo holds itself through two pointers.
type pTwo struct {
	Tags []string `json:"tags"`
	L    *pTwo    `json:"l"`
	R    *pTwo    `json:"r"`
}

// A value that holds itself through two pointers ends at once, as
// encoding/json's own cycle error; a deep value that does not hold itself
// is filled in whole, however deep.
func TestAnswerCyclesEndAndDeepValuesAreFilled(t *testing.T) {
	logs := captureLog(t)
	for _, tags := range [][]string{nil, {}} {
		loop := &pTwo{Tags: tags}
		loop.L, loop.R = loop, loop
		app := New(Config{})
		Get(app, "/x", func(context.Context, None) (*pTwo, error) { return loop, nil })
		h := mustRegister(t, app)
		done := make(chan int, 1)
		began := time.Now()
		go func() { done <- send(t, h, "GET", "/api/x", "").Code }()
		select {
		case code := <-done:
			if code != http.StatusInternalServerError || !strings.Contains(logs.String(), "cycle") {
				t.Fatalf("a two-pointer cycle answered %d, log %q", code, logs)
			}
			t.Logf("a two-pointer cycle ended in %v", time.Since(began))
		case <-time.After(10 * time.Second):
			t.Fatal("a two-pointer cycle had not ended after 10 s")
		}
	}
	var head *pTwo
	for range 1500 {
		head = &pTwo{L: head}
	}
	app := New(Config{})
	Get(app, "/deep", func(context.Context, None) (*pTwo, error) { return head, nil })
	body := send(t, mustRegister(t, app), "GET", "/api/deep", "").Body.String()
	if n := strings.Count(body, `"tags":[]`); n != 1500 || strings.Contains(body, `"tags":null`) {
		t.Fatalf("%d of 1500 lists filled in", n)
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

// pDag is built by a handler into a diamond: each level's two pointers are
// the same pointer, so 18 levels are 2^18 paths to one nil list.
type pDag struct {
	Tags []string `json:"tags"`
	A    *pDag    `json:"a"`
	B    *pDag    `json:"b"`
}

// A pointer the answer reaches through several paths is fixed once, so a
// diamond costs its levels, not its paths.
func TestAnswerDiamondIsFixedOnce(t *testing.T) {
	var leaf *pDag = &pDag{}
	for range 18 {
		leaf = &pDag{A: leaf, B: leaf}
	}
	p, _ := emptyPlanFor(reflect.TypeFor[*pDag]())
	allocs := testing.AllocsPerRun(3, func() { emptied(p, leaf) })
	if allocs > 200 {
		t.Fatalf("a diamond of 18 levels cost %.0f allocations, want at most 200 (its levels, not its 2^18 paths)", allocs)
	}
	out := emptied(p, leaf)
	for n := out; n != nil; n = n.A {
		if n.Tags == nil || n.A != n.B {
			t.Fatal("a level was not fixed, or the diamond came apart")
		}
	}
}
