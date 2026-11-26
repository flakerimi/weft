package warp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"
)

// The model of TestScanMatchesEncodingJSON: keys that differ only in case
// ("name" and "NAME", the readonly "id" and the writable "ID"), keys with an
// s or a k, which fold with ſ and the Kelvin sign, readonly fields at every
// depth, through pointers, lists and map values, and a type that holds
// itself.
type pzGeo struct {
	Lat float64 `json:"lat"`
	Key string  `json:"key" warp:"readonly"`
}

type pzLine struct {
	ID  int64  `json:"id" warp:"readonly"`
	Idx int    `json:"ID"`
	SKU string `json:"sku"`
	Qty *int   `json:"qty"`
	Geo *pzGeo `json:"geo"`
}

type pzBody struct {
	ID     int64             `json:"id" warp:"readonly"`
	Name   string            `json:"name"`
	Shout  string            `json:"NAME"`
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
}

type pzIn struct {
	Body pzBody `body:"json"`
}

// The mirror of the model: each field a hit, which records that
// encoding/json wrote it (called its decoder, null included), and when, or
// for a readonly field an ro. mTop is pzBody as the plan's view has it,
// without its readonly id.
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

type mLine struct {
	ID  ro[int64]   `json:"id"`
	Idx hit[int]    `json:"ID"`
	SKU hit[string] `json:"sku"`
	Qty hit[*int]   `json:"qty"`
	Geo hit[*mGeo]  `json:"geo"`
}

type mBody struct {
	ID     ro[int64]             `json:"id"`
	Name   hit[string]           `json:"name"`
	Shout  hit[string]           `json:"NAME"`
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
}

type mTop struct {
	Name   hit[string]           `json:"name"`
	Shout  hit[string]           `json:"NAME"`
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
}

// topWritten is the top-level keys encoding/json wrote, in the order it met
// them.
func topWritten(m *mTop) []string {
	type key struct {
		name string
		at   int
	}
	var keys []key
	v := reflect.ValueOf(m).Elem()
	for i := range v.NumField() {
		if set, at := v.Field(i).Addr().Interface().(hitter).written(); set {
			name, _, _ := jsonTagOf(v.Type().Field(i))
			keys = append(keys, key{name, at})
		}
	}
	slices.SortFunc(keys, func(a, b key) int { return a.at - b.at })
	var names []string
	for _, k := range keys {
		names = append(names, k.name)
	}
	return names
}

// scanAgrees checks the scan against encoding/json on one body: when
// encoding/json decodes it into the plan's view, the scan refuses it exactly
// when encoding/json writes a readonly field at some depth, and otherwise
// says the body sent exactly the top-level keys encoding/json wrote, in its
// order. It reports whether the body decoded, and what disagreed.
func scanAgrees(p *plan, data []byte) (decoded bool, problem string) {
	realErr := decodeStrict(data, reflect.New(p.view).Interface())
	hitSeq, roWritten = 0, false
	var m mTop
	mirrorErr := decodeStrict(data, &m)
	if (realErr == nil) != (mirrorErr == nil) {
		return false, fmt.Sprintf("the input decodes with %v and the mirror with %v", realErr, mirrorErr)
	}
	if realErr != nil {
		return false, ""
	}
	sent, err := p.walk(data)
	if want := roWritten; (err != nil) != want {
		return true, fmt.Sprintf("encoding/json writes a readonly field: %v; the scan answers %v", want, err)
	}
	if err == nil && !slices.Equal(sent, topWritten(&m)) {
		return true, fmt.Sprintf("the scan says %q was sent; encoding/json wrote %q", sent, topWritten(&m))
	}
	return true, ""
}

func scanPlan(t testing.TB) *plan {
	t.Helper()
	p, mistakes := planFor(reflect.TypeFor[pzIn](), "POST", "/x")
	if mistakes != nil {
		t.Fatal(mistakes)
	}
	// The mirror's top is the view: the same keys in the same order.
	var keys []string
	for f := range reflect.TypeFor[mTop]().Fields() {
		name, _, _ := jsonTagOf(f)
		keys = append(keys, name)
	}
	if !slices.Equal(keys, p.keys.names) {
		t.Fatalf("the mirror has %q, the view %q", keys, p.keys.names)
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
		{"id", "num"}, {"name", "str"}, {"NAME", "str"}, {"kind", "str"}, {"status", "str"}, {"price", "num"},
		{"live", "bool"}, {"tags", "list:str"}, {"lines", "list:obj:line"}, {"by_key", "map:obj:line"},
		{"geo", "obj:geo"}, {"next", "obj:body"}, {"any", "any"}, {"raw", "any"},
	},
	"line": {{"id", "num"}, {"ID", "num"}, {"sku", "str"}, {"qty", "num"}, {"geo", "obj:geo"}},
	"geo":  {{"lat", "num"}, {"key", "str"}},
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
	decoded, refused := 0, 0
	for range scanCases {
		data := g.body()
		ok, problem := scanAgrees(p, data)
		if problem != "" {
			t.Fatalf("%s\nbody %s", problem, data)
		}
		if ok {
			decoded++
			if _, err := p.walk(data); err != nil {
				refused++
			}
		}
	}
	// The bodies reach both sides of the scan.
	if decoded < scanCases/4 || refused < decoded/20 || refused > decoded*9/10 {
		t.Fatalf("of %d bodies %d decoded and %d of those sent a readonly field; change the generator so more do each", scanCases, decoded, refused)
	}
	t.Logf("%d bodies, %d decoded, %d of those refused", scanCases, decoded, refused)
}

func FuzzScan(f *testing.F) {
	g := &bodyGen{r: rand.New(rand.NewPCG(scanSeed, 1))}
	for range 50 {
		f.Add(g.body())
	}
	f.Add([]byte(`{"next":{"ID":1}}`))
	f.Add([]byte(`{"lines":[{"Id":1}]}`))
	p := scanPlan(f)
	f.Fuzz(func(t *testing.T, data []byte) {
		data = bytes.Trim(data, " \t\r\n")
		if !json.Valid(data) || len(data) == 0 || data[0] != '{' || tooDeep(data, maxNesting) {
			return
		}
		if _, problem := scanAgrees(p, data); problem != "" {
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
// that match the last of 100 fields only by case is scanned about as fast as
// one that matches the only field of one.
func TestScanWideStructCost(t *testing.T) {
	timeOf := func(n int, key string) time.Duration {
		p, mistakes := planFor(wideInput(n), "POST", "/x")
		if mistakes != nil {
			t.Fatal(mistakes)
		}
		pair := `"` + key + `":1,`
		body := []byte("{" + strings.Repeat(pair, (1<<20)/len(pair)) + `"f000":1}`)
		if keys, err := p.walk(body); err != nil {
			t.Fatalf("keys %q, err %v", keys, err)
		}
		best := time.Hour
		for range 5 {
			start := time.Now()
			p.walk(body)
			best = min(best, time.Since(start))
		}
		return best
	}
	wide, narrow := timeOf(100, "F099"), timeOf(1, "F000")
	t.Logf("1 MiB against 100 fields: %v; against 1 field: %v", wide, narrow)
	if wide > 4*narrow+2*time.Millisecond {
		t.Fatalf("a 1 MiB body took %v against 100 fields and %v against 1; matching grows with the fields", wide, narrow)
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
