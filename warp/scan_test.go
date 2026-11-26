package warp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// The model of TestScanMatchesEncodingJSON: readonly keys beside writable
// keys that look like them ("id" beside "ident"; the embedded readonly
// "key" beside "keys"), which a key sent in another case may fold onto;
// embedded structs exported and not; a Go name shadowed by an embedded
// struct's (SKU, as "sku" and as "code"); keys with an s or a k, which fold
// with ſ and the Kelvin sign; readonly fields at every depth, through
// pointers, lists, map values and embedded structs; and types that hold
// themselves, through a pointer and through a list, so depth is reached.
// Two keys of one object that fold the same, and an embedded pointer, are
// registration mistakes, so neither is here.
type pzGeo struct {
	Lat float64 `json:"lat"`
	Key string  `json:"key" warp:"readonly"`
}

// PzStamp is embedded, exported: a readonly key and a writable one.
type PzStamp struct {
	Key  string `json:"key" warp:"readonly"`
	Note string `json:"note"`
}

// pzLabel is embedded, unexported: a writable key that folds onto the
// outer type's readonly "id", and a field whose Go name the outer type has.
type pzLabel struct {
	Ident int    `json:"label_id"`
	SKU   string `json:"code"`
	Mark  string `json:"mark" warp:"readonly"`
}

type pzLine struct {
	pzLabel
	PzStamp
	ID   int64    `json:"id" warp:"readonly"`
	SKU  string   `json:"sku"`
	Shut string   `json:"keys"`
	Qty  *int     `json:"qty"`
	Geo  *pzGeo   `json:"geo"`
	Sub  []pzLine `json:"sub"`
}

type pzBody struct {
	ID     int64             `json:"id" warp:"readonly"`
	Ident  int64             `json:"ident"`
	Name   string            `json:"name"`
	Shout  string            `json:"shout"`
	Kind   string            `json:"kind"`
	Status string            `json:"status"`
	Price  float64           `json:"price"`
	Live   bool              `json:"live"`
	Tags   []string          `json:"tags"`
	Lines  []pzLine          `json:"lines"`
	ByKey  map[string]pzLine `json:"by_key"`
	Geo    pzGeo             `json:"geo"`
	Next   *pzBody           `json:"next"`
	Any    any               `json:"any"`
	Raw    json.RawMessage   `json:"raw"`
	PzStamp
}

type pzIn struct {
	Body pzBody `body:"json"`
}

// The mirror of the model: each field a hit, which records that
// encoding/json wrote it (called its decoder, null included), and when, or
// for a readonly field an ro. mTop is pzBody whole, its readonly fields
// included, as the scan knows it.
type hit[T any] struct {
	V   T
	Set bool
	At  int
}

// hitSeq numbers the writes of one decode, so the top-level keys can be put
// in the order encoding/json met them.
var hitSeq int

func (h *hit[T]) UnmarshalJSON(b []byte) error {
	if !h.Set {
		hitSeq++
		h.Set, h.At = true, hitSeq
	}
	return decodeStrict(b, &h.V)
}

func (h *hit[T]) written() (bool, int) { return h.Set, h.At }

type hitter interface {
	written() (set bool, at int)
}

// ro is a readonly field of the mirror: that encoding/json wrote it is kept
// in roWritten, since a later null for what holds it would drop the value.
type ro[T any] struct{ V T }

var roWritten bool

func (r *ro[T]) UnmarshalJSON(b []byte) error {
	roWritten = true
	return decodeStrict(b, &r.V)
}

type mGeo struct {
	Lat hit[float64] `json:"lat"`
	Key ro[string]   `json:"key"`
}

type MStamp struct {
	Key  ro[string]  `json:"key"`
	Note hit[string] `json:"note"`
}

type mLabel struct {
	Ident hit[int]    `json:"label_id"`
	SKU   hit[string] `json:"code"`
	Mark  ro[string]  `json:"mark"`
}

type mLine struct {
	mLabel
	MStamp
	ID   ro[int64]    `json:"id"`
	SKU  hit[string]  `json:"sku"`
	Shut hit[string]  `json:"keys"`
	Qty  hit[*int]    `json:"qty"`
	Geo  hit[*mGeo]   `json:"geo"`
	Sub  hit[[]mLine] `json:"sub"`
}

type mBody struct {
	ID     ro[int64]             `json:"id"`
	Ident  hit[int64]            `json:"ident"`
	Name   hit[string]           `json:"name"`
	Shout  hit[string]           `json:"shout"`
	Kind   hit[string]           `json:"kind"`
	Status hit[string]           `json:"status"`
	Price  hit[float64]          `json:"price"`
	Live   hit[bool]             `json:"live"`
	Tags   hit[[]string]         `json:"tags"`
	Lines  hit[[]mLine]          `json:"lines"`
	ByKey  hit[map[string]mLine] `json:"by_key"`
	Geo    hit[mGeo]             `json:"geo"`
	Next   hit[*mBody]           `json:"next"`
	Any    hit[any]              `json:"any"`
	Raw    hit[json.RawMessage]  `json:"raw"`
	MStamp
}

type mTop = mBody

// topWritten is the top-level keys, readonly ones aside, that encoding/json
// wrote, in the order it met them, those of embedded structs included.
func topWritten(m *mTop) []string {
	type key struct {
		name string
		at   int
	}
	var keys []key
	var walk func(v reflect.Value)
	walk = func(v reflect.Value) {
		for i := range v.NumField() {
			f := v.Type().Field(i)
			if f.Anonymous {
				walk(v.Field(i))
				continue
			}
			if h, ok := v.Field(i).Addr().Interface().(hitter); ok {
				if set, at := h.written(); set {
					name, _, _ := jsonTagOf(f)
					keys = append(keys, key{name, at})
				}
			}
		}
	}
	walk(reflect.ValueOf(m).Elem())
	slices.SortFunc(keys, func(a, b key) int { return a.at - b.at })
	var names []string
	for _, k := range keys {
		names = append(names, k.name)
	}
	return names
}

// readonlyFolds are the model's readonly keys, folded: a key the scan
// refuses though encoding/json wrote no readonly field must fold onto one
// of these, since the scan refuses a key when any field it may set is
// readonly, whichever the decoder chooses.
var readonlyFolds = []string{"ID", "KEY", "MARK"}

// scanAgrees checks the scan against encoding/json on one body, which the
// mirror (the whole model, readonly fields included) decodes:
//   - when encoding/json writes a readonly field at some depth, the scan
//     refuses the body;
//   - when the scan takes the body, the plan's view decodes it too, the
//     scan says it sent exactly the top-level keys encoding/json wrote, in
//     its order, and the decoded input has no readonly field set;
//   - when the scan refuses a body encoding/json would have taken whole, the
//     key it names folds onto a readonly key of the model.
//
// It reports whether the mirror decoded the body, whether the scan refused
// it, and what disagreed.
func scanAgrees(p *plan, data []byte) (decoded, refused bool, problem string) {
	hitSeq, roWritten = 0, false
	var m mTop
	if err := decodeStrict(data, &m); err != nil {
		return false, false, ""
	}
	sent, err := p.walk(data)
	switch {
	case roWritten && err == nil:
		return true, false, "encoding/json writes a readonly field and the scan takes the body"
	case err == nil:
		view := reflect.New(p.view)
		if verr := decodeStrict(data, view.Interface()); verr != nil {
			return true, false, fmt.Sprintf("the scan takes the body and the view refuses it: %v", verr)
		}
		if !slices.Equal(sent, topWritten(&m)) {
			return true, false, fmt.Sprintf("the scan says %q was sent; encoding/json wrote %q", sent, topWritten(&m))
		}
		if p.readonly != nil {
			p.readonly.zero(view.Elem())
		}
		if name, set := readonlySet(view.Elem()); set {
			return true, false, fmt.Sprintf("the readonly %s is set", name)
		}
	case !roWritten:
		var e *Error
		errors.As(err, &e)
		key, _ := strings.CutPrefix(e.Message, "The body has the field ")
		key, _ = strings.CutSuffix(key, ", which this route does not take.")
		name, uerr := strconv.Unquote(key)
		if uerr != nil || !slices.Contains(readonlyFolds, string(appendFolded(nil, []byte(name)))) {
			return true, true, fmt.Sprintf("the scan refuses %v, which folds onto no readonly key", err)
		}
	}
	return true, err != nil, ""
}

// readonlySet reports the first readonly field of v, a decoded value of the
// model, that is not zero.
func readonlySet(v reflect.Value) (string, bool) {
	switch v.Kind() {
	case reflect.Pointer, reflect.Interface:
		if !v.IsNil() {
			return readonlySet(v.Elem())
		}
	case reflect.Slice, reflect.Array:
		for i := range v.Len() {
			if name, set := readonlySet(v.Index(i)); set {
				return name, true
			}
		}
	case reflect.Map:
		for _, e := range v.Seq2() {
			if name, set := readonlySet(e); set {
				return name, true
			}
		}
	case reflect.Struct:
		for i := range v.NumField() {
			f := v.Type().Field(i)
			if isReadonly(f) && !v.Field(i).IsZero() {
				return v.Type().Name() + "." + f.Name, true
			}
			if f.IsExported() || f.Anonymous {
				if name, set := readonlySet(v.Field(i)); set {
					return name, true
				}
			}
		}
	}
	return "", false
}

func scanPlan(t testing.TB) *plan {
	t.Helper()
	p, mistakes := planFor(reflect.TypeFor[pzIn](), "POST", "/x")
	if mistakes != nil {
		t.Fatal(mistakes)
	}
	return p
}

// bodyGen writes random bodies of the model: keys in any case, with ſ and K
// for s and k, with escapes, sent twice, unknown; values null, of the wrong
// type, of every container, nested.
type bodyGen struct {
	r   *rand.Rand
	buf strings.Builder
}

type genField struct {
	key  string
	kind string // str, num, bool, any, list:<kind>, map:<kind>, obj:<object>
}

var genObjects = map[string][]genField{
	"body": {
		{"id", "num"}, {"ident", "num"}, {"name", "str"}, {"shout", "str"}, {"kind", "str"}, {"status", "str"}, {"price", "num"},
		{"live", "bool"}, {"tags", "list:str"}, {"lines", "list:obj:line"}, {"by_key", "map:obj:line"},
		{"geo", "obj:geo"}, {"next", "obj:body"}, {"any", "any"}, {"raw", "any"}, {"key", "str"}, {"note", "str"},
	},
	"line": {
		{"id", "num"}, {"label_id", "num"}, {"sku", "str"}, {"code", "str"}, {"qty", "num"}, {"geo", "obj:geo"},
		{"key", "str"}, {"keys", "str"}, {"note", "str"}, {"mark", "str"}, {"sub", "list:obj:line"},
	},
	"geo": {{"lat", "num"}, {"key", "str"}},
}

var oddKeys = []string{"zz", "nam", "name ", "naмe", `n\u0000ame`, `\ud83d\ude00`, `\ud800`, "", "ſ", "K"}

func (g *bodyGen) body() []byte {
	g.buf.Reset()
	g.object("body", 0)
	return []byte(g.buf.String())
}

func (g *bodyGen) space() {
	if g.r.IntN(8) == 0 {
		g.buf.WriteString([]string{" ", "\n", "\t ", "\r\n"}[g.r.IntN(4)])
	}
}

func (g *bodyGen) object(name string, depth int) {
	fields := genObjects[name]
	g.buf.WriteByte('{')
	for i := range g.r.IntN(6) {
		if i > 0 {
			g.buf.WriteByte(',')
		}
		g.space()
		if g.r.IntN(25) == 0 {
			g.key(oddKeys[g.r.IntN(len(oddKeys))])
			g.space()
			g.buf.WriteByte(':')
			g.value("any", depth+1)
			continue
		}
		f := fields[g.r.IntN(len(fields))]
		g.key(f.key)
		g.space()
		g.buf.WriteByte(':')
		g.space()
		g.value(f.kind, depth+1)
		g.space()
	}
	g.buf.WriteByte('}')
}

// key writes k mangled: each letter in either case, an s or k as ſ or K
// now and then, and characters escaped.
func (g *bodyGen) key(k string) {
	if strings.Contains(k, `\`) { // already written as JSON
		g.buf.WriteString(`"` + k + `"`)
		return
	}
	g.buf.WriteByte('"')
	for _, r := range k {
		switch {
		case (r == 's' || r == 'S') && g.r.IntN(6) == 0:
			r = 'ſ'
		case (r == 'k' || r == 'K') && g.r.IntN(6) == 0:
			r = 'K'
		case r >= 'a' && r <= 'z' && g.r.IntN(5) == 0:
			r -= 'a' - 'A'
		case r >= 'A' && r <= 'Z' && g.r.IntN(5) == 0:
			r += 'a' - 'A'
		}
		if g.r.IntN(6) == 0 {
			format := []string{`\u%04x`, `\u%04X`}[g.r.IntN(2)]
			fmt.Fprintf(&g.buf, format, r)
			continue
		}
		g.buf.WriteRune(r)
	}
	g.buf.WriteByte('"')
}

func (g *bodyGen) value(kind string, depth int) {
	switch n := g.r.IntN(20); {
	case n == 0 || depth > 5:
		g.buf.WriteString("null")
		return
	case n == 1:
		kind = []string{"str", "num", "bool", "list:any", "map:any"}[g.r.IntN(5)] // likely the wrong type
	}
	switch {
	case kind == "str":
		g.buf.WriteString([]string{`"lamp"`, `""`, `"a\"b\\cé"`, `"x y"`}[g.r.IntN(4)])
	case kind == "num":
		g.buf.WriteString([]string{"0", "7", "-3", "1.5", "1e2"}[g.r.IntN(5)])
	case kind == "bool":
		g.buf.WriteString([]string{"true", "false"}[g.r.IntN(2)])
	case kind == "any":
		g.value([]string{"str", "num", "bool", "list:any", "map:any", "obj:geo"}[g.r.IntN(6)], depth)
	case strings.HasPrefix(kind, "list:"):
		g.buf.WriteByte('[')
		for i := range g.r.IntN(4) {
			if i > 0 {
				g.buf.WriteByte(',')
			}
			g.space()
			g.value(kind[5:], depth+1)
		}
		g.buf.WriteByte(']')
	case strings.HasPrefix(kind, "map:"):
		g.buf.WriteByte('{')
		for i := range g.r.IntN(3) {
			if i > 0 {
				g.buf.WriteByte(',')
			}
			g.buf.WriteString(strconv.Quote([]string{"a", "id", "ID", "b"}[g.r.IntN(4)]) + ":")
			g.value(kind[4:], depth+1)
		}
		g.buf.WriteByte('}')
	case strings.HasPrefix(kind, "obj:"):
		g.object(kind[4:], depth)
	}
}

// scanCases is how many bodies TestScanMatchesEncodingJSON tries, from
// scanSeed: enough for every kind of key and value to meet the others, in
// well under a second.
const (
	scanCases = 4000
	scanSeed  = 20261125
)

// The scan is written by hand, to match keys as encoding/json does. This
// holds it to that on generated bodies, so a change in encoding/json that
// would let a readonly field be set, or a key be sent unseen, fails here.
func TestScanMatchesEncodingJSON(t *testing.T) {
	p := scanPlan(t)
	g := &bodyGen{r: rand.New(rand.NewPCG(scanSeed, 0))}
	decoded, refused, stricter := 0, 0, 0
	for range scanCases {
		data := g.body()
		hitSeq, roWritten = 0, false
		ok, no, problem := scanAgrees(p, data)
		if problem != "" {
			t.Fatalf("%s\nbody %s", problem, data)
		}
		if ok {
			decoded++
		}
		if no {
			refused++
			if !roWritten {
				stricter++
			}
		}
	}
	// The bodies reach both sides of the scan.
	if decoded < scanCases/4 || refused < decoded/20 || refused > decoded*9/10 {
		t.Fatalf("of %d bodies %d decoded and %d of those were refused; change the generator so more do each", scanCases, decoded, refused)
	}
	t.Logf("%d bodies, %d decoded, %d of those refused (%d where encoding/json chose a writable field)", scanCases, decoded, refused, stricter)
}

func FuzzScan(f *testing.F) {
	g := &bodyGen{r: rand.New(rand.NewPCG(scanSeed, 1))}
	for range 50 {
		f.Add(g.body())
	}
	f.Add([]byte(`{"next":{"Id":1}}`))
	f.Add([]byte(`{"lines":[{"ID":1}]}`))
	f.Add([]byte(`{"lines":[{"sub":[{"sub":[{"Key":"x"}]}]}]}`))
	p := scanPlan(f)
	f.Fuzz(func(t *testing.T, data []byte) {
		data = bytes.Trim(data, " \t\r\n")
		if !json.Valid(data) || len(data) == 0 || data[0] != '{' || tooDeep(data, maxNesting) {
			return
		}
		if _, _, problem := scanAgrees(p, data); problem != "" {
			t.Fatalf("%s\nbody %s", problem, data)
		}
	})
}

// wideInput is an input of n json fields, f000 to f(n-1).
func wideInput(n int) reflect.Type {
	var fields []reflect.StructField
	for i := range n {
		fields = append(fields, reflect.StructField{
			Name: fmt.Sprintf("F%03d", i), Type: reflect.TypeFor[int](),
			Tag: reflect.StructTag(fmt.Sprintf(`json:"f%03d"`, i)),
		})
	}
	return reflect.StructOf(fields)
}

// Matching a key costs the same however many fields the object has: an
// exact map and a folded one, made at registration. A 1 MiB body of keys
// that match the last of 100 fields only by case takes two looks a key, as
// one that matches the only field of one does, and no allocation for each.
func TestScanWideStructCost(t *testing.T) {
	for _, c := range []struct {
		n   int
		key string
	}{{100, "F099"}, {1, "F000"}, {100, "f050"}} {
		p, mistakes := planFor(wideInput(c.n), "POST", "/x")
		if mistakes != nil {
			t.Fatal(mistakes)
		}
		pair := `"` + c.key + `":1,`
		keys := (1 << 20) / len(pair)
		body := []byte("{" + strings.Repeat(pair, keys) + `"f000":1}`)
		s := bodyScan{data: body}
		if _, err := s.object(0, p.keys, true); err != nil {
			t.Fatal(err)
		}
		if s.compares > 2*(keys+1) {
			t.Errorf("%d keys against %d fields took %d looks, want at most 2 a key", keys+1, c.n, s.compares)
		}
		// The list of keys sent, and the record of those past the 64th.
		if allocs := testing.AllocsPerRun(3, func() { p.walk(body) }); allocs > 2 {
			t.Errorf("%d keys against %d fields cost %.0f allocations, want at most 2", keys+1, c.n, allocs)
		}
	}
}

// A key with an escape is unquoted without allocating: a body of a thousand
// costs what a body of one does.
func TestScanEscapedKeysCost(t *testing.T) {
	type In struct {
		A  int `json:"a"`
		Sk int `json:"sk"`
	}
	p, mistakes := planFor(reflect.TypeFor[In](), "POST", "/x")
	if mistakes != nil {
		t.Fatal(mistakes)
	}
	body := []byte("{" + strings.Repeat(`"\u0061":1,"\u0053\u212a":2,"\u017fk":2,"\u00e9t\u00e9":3,`, 1000) + `"a":1}`)
	keys, err := p.walk(body)
	if err != nil || !slices.Equal(keys, []string{"a", "sk"}) {
		t.Fatalf("keys %q, err %v", keys, err)
	}
	allocs := testing.AllocsPerRun(5, func() { p.walk(body) })
	t.Logf("%.0f allocations", allocs)
	if allocs > 1 {
		t.Fatalf("4000 escaped keys cost %.0f allocations, want at most 1 (the list of keys sent)", allocs)
	}
}

// Whatever the scan says, a readonly field is never set: every one the
// decoder set, through each path a body can take, is put back to zero. The
// scan is left out here, as if it had missed every key.
func TestReadonlyIsZeroedWhateverTheScanSays(t *testing.T) {
	p := scanPlan(t)
	for _, body := range []string{
		`{"lines":[{"sku":"a"},{"id":5}]}`,            // a list
		`{"by_key":{"a":{"id":5}}}`,                   // a map value
		`{"geo":{"key":"x"}}`,                         // a struct
		`{"next":{"next":{"geo":{"key":"x"}}}}`,       // through pointers, a type holding itself
		`{"lines":[{"key":"x"}]}`,                     // an exported embedded struct
		`{"lines":[{"mark":"x"}]}`,                    // an unexported embedded struct
		`{"by_key":{"a":{"geo":{"key":"x"}}}}`,        // a pointer inside a map value
		`{"next":{"key":"x","lines":[{"MARK":"y"}]}}`, // folded, deep
		`{"lines":[{"sub":[{"sub":[{"id":5}]}]}]}`,    // a list of the type itself
	} {
		view := reflect.New(p.view)
		if err := decodeStrict([]byte(body), view.Interface()); err != nil {
			t.Fatalf("%s: %v", body, err)
		}
		if _, set := readonlySet(view.Elem()); !set {
			t.Fatalf("%s: the decoder set no readonly field, so this row tests nothing", body)
		}
		p.readonly.zero(view.Elem())
		if name, set := readonlySet(view.Elem()); set {
			t.Errorf("%s: %s is still set", body, name)
		}
	}
	// A body type with no readonly field has no plan, and one with them
	// costs nothing to put right when there are no maps to look into.
	if readonlyPlanFor(reflect.TypeFor[struct {
		A []Item          `json:"a"`
		B map[string]Item `json:"b"`
		C *bodyFields     `json:"c"`
	}]()) != nil {
		t.Error("a type with no readonly field has a plan")
	}
	view := reflect.New(p.view)
	decodeStrict([]byte(`{"lines":[{"id":5},{"key":"x"}],"next":{"geo":{"key":"x"}}}`), view.Interface())
	if allocs := testing.AllocsPerRun(20, func() { p.readonly.zero(view.Elem()) }); allocs != 0 {
		t.Errorf("putting readonly fields back cost %v allocations, want 0", allocs)
	}
}

type sWBase struct {
	Ident int `json:"Id"`
}

// sShallowRO has a writable key in an embedded struct and the outer type's
// own readonly key, which differ only in case: encoding/json v1 would set
// the first for "ID", and jsonv2 the shallower, the readonly one. Such a
// type is refused at registration.
type sShallowRO struct {
	sWBase
	ID int `json:"id" warp:"readonly"`
}

func TestReadonlyFoldedBesideAnEmbeddedKey(t *testing.T) {
	type In struct {
		Body struct {
			In sShallowRO `json:"in"`
		} `body:"json"`
	}
	app := New(Config{})
	Post(app, "/x", func(_ context.Context, in In) (None, error) { return None{}, nil })
	if want := `warp: POST /x: sShallowRO has the fields Ident and ID, whose keys differ only in case; rename one.`; errText(app.Err()) != want {
		t.Fatalf("Err() = %q\nwant %q", errText(app.Err()), want)
	}
}

// The top level refuses as the levels below do: a key that folds onto a
// readonly field is refused there too, though the view leaves the readonly
// field out.
func TestReadonlyFoldedAtTheTop(t *testing.T) {
	type Top struct {
		ID    int `json:"id" warp:"readonly"`
		Other int `json:"other"`
	}
	top, mistakes := planFor(reflect.TypeFor[struct {
		Body Top `body:"json"`
	}](), "POST", "/x")
	if mistakes != nil {
		t.Fatal(mistakes)
	}
	nested, mistakes := planFor(reflect.TypeFor[struct {
		Body struct {
			In Top `json:"in"`
		} `body:"json"`
	}](), "POST", "/x")
	if mistakes != nil {
		t.Fatal(mistakes)
	}
	for _, c := range []struct {
		key   string
		taken bool
	}{{"id", false}, {"Id", false}, {"iD", false}, {"ID", false}, {"other", true}, {"OTHER", true}} {
		_, errTop := top.walk([]byte(`{"` + c.key + `":5}`))
		_, errNested := nested.walk([]byte(`{"in":{"` + c.key + `":5}}`))
		if (errTop == nil) != c.taken || (errNested == nil) != c.taken {
			t.Errorf("%q: at the top %v, one level down %v; want taken %v at both", c.key, errTop, errNested, c.taken)
		}
	}
}

// zDeep holds itself through two keys that both reach a readonly field: a
// deep body is read once, each key looked up at most twice, however many
// fields its keys could fold onto.
type zDeep struct {
	ID int    `json:"id" warp:"readonly"`
	A  *zDeep `json:"ab"`
	B  *zDeep `json:"cd"`
}

func TestScanDeepBodyCost(t *testing.T) {
	p, mistakes := planFor(reflect.TypeFor[struct {
		Body zDeep `body:"json"`
	}](), "POST", "/x")
	if mistakes != nil {
		t.Fatal(mistakes)
	}
	const depth = maxNesting - 1
	body := []byte(strings.Repeat(`{"Ab":`, depth) + "{}" + strings.Repeat("}", depth))
	s := bodyScan{data: body}
	if _, err := s.object(0, p.keys, true); err != nil {
		t.Fatal(err)
	}
	if s.compares > 2*depth {
		t.Fatalf("a body %d deep took %d looks, want at most %d", depth, s.compares, 2*depth)
	}
}
