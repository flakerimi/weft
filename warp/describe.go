package warp

import (
	"encoding"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Description is what an app says of itself: every route, and every struct
// type its routes take and answer, as data. weft gen writes the app's
// TypeScript client, form schemas and routes table from it, so it is the
// contract between the Go side and the browser side. Its lists are never nil,
// and two descriptions of one app encode to the same JSON.
type Description struct {
	Routes []RouteInfo `json:"routes"`
	Types  []TypeInfo  `json:"types"`
}

// RouteInfo describes one route, as Routes lists it.
//
// A route added with Handle has Typed false, no parameters, no body and no
// output, and a Method of "" when it takes every method. A typed route's
// PathParams and Query are what it reads from the path and the query; Body
// names the type in Types its JSON body is read as, "" when it reads none.
// When that type has readonly fields, at any depth, Body names its twin, the
// type's name with Input after it. Partial is true for a route registered
// with Patch: every top-level field of its body may then be left out, while
// a nested object that is sent is checked whole. Output is nil when the
// route answers None (with 204) and for Handle.
type RouteInfo struct {
	Module     string      `json:"module"`
	Name       string      `json:"name"`
	Method     string      `json:"method"`
	Path       string      `json:"path"` // with /api in front, as registered: /api/products/{id}
	Status     int         `json:"status"`
	Permission string      `json:"permission"`
	Typed      bool        `json:"typed"`
	PathParams []ParamInfo `json:"path_params"`
	Query      []ParamInfo `json:"query"`
	Body       string      `json:"body"`
	Partial    bool        `json:"partial"`
	Output     *OutputInfo `json:"output"`
}

// ParamInfo describes a path parameter or a query field. Name is its key in
// the path or the query, and TS its TypeScript type: string, number, boolean,
// string[] or Record<string, string>. Style says how it is sent: "value" is
// one key=value; "list" is the key once for each value; "filter" is
// key[name]=value for each entry. A query field is always optional and a
// path parameter never is.
type ParamInfo struct {
	Name     string     `json:"name"`
	TS       string     `json:"ts"`
	Optional bool       `json:"optional"`
	Style    string     `json:"style"`
	Rules    []RuleInfo `json:"rules"`
}

// OutputInfo describes what a route answers: TS is a TypeScript type (a
// name in Types for a struct), of each element when List is true. A
// generator writes a list of nullable elements as (T | null)[].
type OutputInfo struct {
	TS   string `json:"ts"`
	List bool   `json:"list"`
}

// TypeInfo describes a struct type: Name is its name in TypeScript (its Go
// name without the package, a generic type's with its arguments' names after
// it, so List[Product] is ListProduct), Go is its Go name with the package,
// and Fields are what its JSON holds, in order. A twin, named with Input
// after the type's name, has the same Go name and leaves out the readonly
// fields, and its fields name the twins of the types they hold.
type TypeInfo struct {
	Name   string      `json:"name"`
	Go     string      `json:"go"`
	Fields []FieldInfo `json:"fields"`
}

// FieldInfo describes one field of a type: its JSON name, its TypeScript
// type, whether it is optional, whether it is readonly (answered and never
// read), and its rules as its warp tag has them. Optional means the field may
// be absent or, for a pointer, null: it is a pointer, or its json tag has
// omitempty or omitzero (on a type's own entry, which is what is answered;
// not on its Input twin, since those options never change what is read).
// A 64-bit integer is number, and loses precision above 2^53.
type FieldInfo struct {
	Name     string     `json:"name"`
	TS       string     `json:"ts"`
	Optional bool       `json:"optional"`
	Readonly bool       `json:"readonly"`
	Rules    []RuleInfo `json:"rules"`
}

// RuleInfo is one rule of a warp tag: its name and what follows its =, as
// written ("max" and "120"; "enum" and "draft|live"; "required" and "").
type RuleInfo struct {
	Name string `json:"name"`
	Arg  string `json:"arg"`
}

// Describe returns every route, in the order registered, and every struct
// type they take and answer, in the order first used, each followed by its
// twin when it has one. It reads what registration made, so it may be called
// before or after Start; it runs no module. Two types with one name, an
// unnamed struct, a field JSON cannot carry, a map keyed by anything but text
// or a whole number, a field with json ",string" and a type with half an
// encoding are mistakes, returned joined; a ,string that registration
// already reported, on a type a route also reads, is not reported again.
func (a *App) Describe() (Description, error) {
	a.mu.Lock()
	routes := slices.Clone(a.routes)
	plans := slices.Clone(a.plans)
	registered := slices.Clone(a.mistakes)
	a.mu.Unlock()

	d := &describer{fields: map[reflect.Type][]descField{}, read: map[reflect.Type]bool{}, half: map[reflect.Type]bool{}, registered: registered}
	for _, p := range plans {
		if p != nil && p.bodyType != nil {
			d.markRead(p.bodyType)
		}
	}
	out := Description{Routes: make([]RouteInfo, 0, len(routes)), Types: []TypeInfo{}}
	bodies := make([]reflect.Type, len(routes))
	for i, rt := range routes {
		info := RouteInfo{
			Module: rt.Module, Name: rt.Name, Method: rt.Method, Path: rt.Path,
			Status: rt.Status, Permission: rt.Permission,
			PathParams: []ParamInfo{}, Query: []ParamInfo{},
		}
		p := plans[i]
		if p != nil {
			info.Typed, info.Partial = true, p.partial
			at := strings.TrimSpace(rt.Method + " " + strings.TrimPrefix(rt.Path, "/api"))
			if !p.none {
				rules := rulesOf(rt.In)
				for _, pp := range p.path {
					info.PathParams = append(info.PathParams, paramInfo(pp, false, rules))
				}
				for _, q := range p.query {
					info.Query = append(info.Query, paramInfo(q, true, rules))
				}
			}
			if p.bodyType != nil {
				bodies[i] = p.bodyType
				d.use(p.bodyType, at, at+" takes", "give the route another input")
			}
			if rt.Out != noneType {
				t, list := rt.Out, isList(rt.Out)
				if list {
					t = t.Elem()
				}
				d.use(t, at, at+" answers", "give the route another output")
				d.unaddressed(rt.Out, true, at+" answers", map[unaddressedKey]bool{})
				info.Output = &OutputInfo{TS: d.ts(t, nil), List: list}
			}
		}
		out.Routes = append(out.Routes, info)
	}

	twin := d.twins()
	for i, b := range bodies {
		if b != nil {
			out.Routes[i].Body = d.ts(b, twin)
		}
	}
	seen := map[string]string{} // a TypeScript name to the Go type that has it
	name := func(ts, goName string) {
		if other, ok := seen[ts]; ok {
			d.fail("two types are named %q (%s and %s); rename one, since the generated TypeScript has one name for each.", ts, other, goName)
			return
		}
		seen[ts] = goName
	}
	for _, t := range d.types {
		goName := t.PkgPath() + "." + t.Name()
		full := TypeInfo{Name: tsName(t), Go: goName, Fields: []FieldInfo{}}
		name(full.Name, goName)
		for _, f := range d.fields[t] {
			full.Fields = append(full.Fields, d.fieldInfo(f, nil, false))
		}
		out.Types = append(out.Types, full)
		if !twin[t] {
			continue
		}
		input := TypeInfo{Name: full.Name + "Input", Go: goName, Fields: []FieldInfo{}}
		name(input.Name, "the input of "+goName)
		for _, f := range d.fields[t] {
			if !f.readonly {
				input.Fields = append(input.Fields, d.fieldInfo(f, twin, true))
			}
		}
		out.Types = append(out.Types, input)
	}
	if len(d.mistakes) > 0 {
		return Description{}, errors.Join(d.mistakes...)
	}
	return out, nil
}

// describer collects the struct types a description reaches, each once.
type describer struct {
	types    []reflect.Type // in order of first use
	fields   map[reflect.Type][]descField
	read     map[reflect.Type]bool // the types a body is read into, at any depth
	half     map[reflect.Type]bool // the types with half an encoding, reported once
	mistakes []error
	// registered are the app's registration mistakes, so that a ,string a
	// route reading the type reported is not reported again.
	registered []error
}

// reportedAt reports whether a registration mistake already ends with
// sentence, as one a route found in the type it reads does.
func (d *describer) reportedAt(sentence string) bool {
	for _, err := range d.registered {
		if strings.HasSuffix(err.Error(), ": "+sentence) {
			return true
		}
	}
	return false
}

// descField is a field of a type's JSON: its key, its Go name, its type,
// where it sits, whether it is readonly, whether its json tag has omitempty
// or omitzero, whether it has ,string, and its rules.
type descField struct {
	key, goName string
	typ         reflect.Type
	index       []int
	readonly    bool
	omits       bool
	quoted      bool
	rules       []RuleInfo
}

// markRead marks the struct type a value of type t holds as one a body is
// read into, and the struct types its fields that are not readonly hold: a
// readonly field is never read, so what it holds is only answered.
func (d *describer) markRead(t reflect.Type) {
	t = leaf(t)
	if t.Kind() != reflect.Struct || t == timeType || selfEncoded(t) != "" || t.Name() == "" || d.read[t] {
		return
	}
	d.read[t] = true
	for _, f := range readFields(t) {
		if !f.readonly {
			d.markRead(f.typ)
		}
	}
}

func (d *describer) fail(format string, args ...any) {
	d.mistakes = append(d.mistakes, fmt.Errorf("warp: "+format, args...))
}

// use adds the struct type a value of type t holds, and every struct type
// its fields hold, to the types described. A type a body is read into has
// its fields as the binder lists them; any other, as encoding/json writes
// them. route is the route that first reaches it ("GET /products"); where
// begins the sentence of a mistake ("GET /products answers"), and fix ends
// it when JSON cannot carry the type.
func (d *describer) use(t reflect.Type, route, where, fix string) {
	// encoding/json writes a map only when its key is text, a whole number
	// or writes itself as text; the generated TypeScript has Record<string, T>
	// for one keyed by text or a whole number alone.
	for k := t; k != timeType && selfEncoded(k) == "" && isContainer(k.Kind()) && !isBytes(k); k = k.Elem() {
		if k.Kind() == reflect.Map && k.Key().Kind() != reflect.String && !isIntegerKind(k.Key().Kind()) {
			lead := where
			if !strings.HasPrefix(where, route+" ") {
				lead = route + ": " + where
			}
			d.fail("%s a map keyed by %s; key it by text or a whole number, since JSON object keys are text.", lead, typeName(k.Key()))
		}
	}
	t = leaf(t)
	switch {
	case t == timeType:
	case selfEncoded(t) != "":
		if m := halfEncoded(t); m != "" && !d.half[t] {
			d.half[t] = true
			d.fail("%s", m)
		}
	case t.Kind() == reflect.Struct:
		if t.Name() == "" {
			d.fail("%s an unnamed struct; give the type a name, since the generated TypeScript needs one.", where)
			return
		}
		if _, ok := d.fields[t]; ok {
			return
		}
		var fields []descField
		if d.read[t] {
			fields = readFields(t)
		} else {
			fields = jsonFields(t)
		}
		d.fields[t] = fields
		d.types = append(d.types, t)
		for _, f := range fields {
			at := typeName(t) + "." + f.goName
			if f.quoted && !d.reportedAt(stringOption(at)) {
				d.fail("%s: %s", route, stringOption(at))
			}
			d.use(f.typ, route, at+" is", `give it another type, or tag it json:"-"`)
		}
	case t.Kind() == reflect.Chan || t.Kind() == reflect.Func || t.Kind() == reflect.UnsafePointer ||
		t.Kind() == reflect.Complex64 || t.Kind() == reflect.Complex128:
		d.fail("%s a %s, which JSON cannot carry; %s.", where, t, fix)
	}
}

type unaddressedKey struct {
	t    reflect.Type
	addr bool
}

// unaddressed reports a type of an answer whose MarshalJSON or MarshalText
// has a pointer receiver where encoding/json meets it unaddressable, in a
// map's values, so cannot call it and writes the underlying value, which the
// description does not say. An answer is written through a pointer, so its
// value, what pointers and lists hold, and the fields and array elements of
// those are addressable; a map's values, and what they hold by value, are
// not. addr is whether a value of type t is addressable there.
func (d *describer) unaddressed(t reflect.Type, addr bool, where string, seen map[unaddressedKey]bool) {
	if seen[unaddressedKey{t, addr}] {
		return
	}
	seen[unaddressedKey{t, addr}] = true
	if t.Kind() != reflect.Pointer && t.Kind() != reflect.Interface && t != timeType {
		pt := reflect.PointerTo(t)
		for _, m := range []struct {
			iface reflect.Type
			name  string
		}{{jsonMarshaler, "MarshalJSON"}, {textMarshaler, "MarshalText"}} {
			if t.Implements(m.iface) {
				return // written by its own method wherever it is
			}
			if pt.Implements(m.iface) {
				if !addr {
					d.fail("%s a map of %s, whose %s has a pointer receiver, so encoding/json writes the map's values without it; give %s a value receiver, or make the map hold *%s.",
						where, typeName(t), m.name, m.name, typeName(t))
				}
				return
			}
		}
	}
	switch t.Kind() {
	case reflect.Pointer, reflect.Slice:
		d.unaddressed(t.Elem(), true, where, seen)
	case reflect.Array:
		d.unaddressed(t.Elem(), addr, where, seen)
	case reflect.Map:
		d.unaddressed(t.Elem(), false, where, seen)
	case reflect.Struct:
		if t == timeType {
			return
		}
		for _, f := range jsonFields(t) {
			d.unaddressed(f.typ, addr, where, seen)
		}
	}
}

// structOf is the described struct type a value of type t holds, or nil.
func (d *describer) structOf(t reflect.Type) reflect.Type {
	t = leaf(t)
	if _, ok := d.fields[t]; ok {
		return t
	}
	return nil
}

// twins returns the types that get a twin: those a body is read into that
// have a readonly field or reach, so, a type that has one.
func (d *describer) twins() map[reflect.Type]bool {
	deep := map[reflect.Type]bool{}
	// A type can hold itself, so this marks until nothing changes.
	for changed := true; changed; {
		changed = false
		for _, t := range d.types {
			if deep[t] {
				continue
			}
			for _, f := range d.fields[t] {
				if inner := d.structOf(f.typ); f.readonly || inner != nil && deep[inner] {
					deep[t], changed = true, true
					break
				}
			}
		}
	}
	twin := map[reflect.Type]bool{}
	for _, t := range d.types {
		twin[t] = d.read[t] && deep[t]
	}
	return twin
}

// fieldInfo describes f in a type's entry, or with input in its twin's,
// where twin names the types that have one. omitempty and omitzero change
// what the server writes, never what it reads, so on a twin only a pointer
// is optional.
func (d *describer) fieldInfo(f descField, twin map[reflect.Type]bool, input bool) FieldInfo {
	return FieldInfo{
		Name: f.key, TS: d.ts(f.typ, twin), Optional: f.typ.Kind() == reflect.Pointer || f.omits && !input,
		Readonly: f.readonly, Rules: f.rules,
	}
}

var (
	jsonMarshaler = reflect.TypeFor[json.Marshaler]()
	textMarshaler = reflect.TypeFor[encoding.TextMarshaler]()
	jsonNumber    = reflect.TypeFor[json.Number]()
)

// ts is the TypeScript type of a value of type t. A struct is its name, or
// its twin's when twin has it.
func (d *describer) ts(t reflect.Type, twin map[reflect.Type]bool) string {
	switch {
	case t == timeType:
		return "string"
	case t == jsonNumber:
		return "number"
	case selfEncoded(t) != "":
		return selfEncoded(t)
	}
	switch t.Kind() {
	case reflect.Pointer:
		inner := d.ts(t.Elem(), twin)
		if strings.HasSuffix(inner, " | null") {
			return inner
		}
		return inner + " | null"
	case reflect.Slice, reflect.Array:
		if isBytes(t) {
			return "string"
		}
		inner := d.ts(t.Elem(), twin)
		if strings.Contains(inner, " | ") {
			inner = "(" + inner + ")"
		}
		return inner + "[]"
	case reflect.Map:
		return "Record<string, " + d.ts(t.Elem(), twin) + ">"
	case reflect.Struct:
		if twin[t] {
			return tsName(t) + "Input"
		}
		return tsName(t)
	}
	return kindTS(t.Kind())
}

// isIntegerKind reports whether k is an int or uint kind, uintptr included.
func isIntegerKind(k reflect.Kind) bool {
	return k >= reflect.Int && k <= reflect.Uintptr
}

// kindTS is the TypeScript type of a string, bool or number kind, and
// unknown for any other.
func kindTS(k reflect.Kind) string {
	switch {
	case k == reflect.String:
		return "string"
	case k == reflect.Bool:
		return "boolean"
	case isNumberKind(k) || k == reflect.Uintptr:
		return "number"
	}
	return "unknown"
}

// selfEncoded is the TypeScript type of a type that writes or reads itself:
// unknown as JSON, string as text; "" for any other type.
func selfEncoded(t reflect.Type) string {
	if t.Kind() == reflect.Pointer || t.Kind() == reflect.Interface {
		return ""
	}
	pt := reflect.PointerTo(t)
	implements := func(i reflect.Type) bool { return t.Implements(i) || pt.Implements(i) }
	switch {
	case implements(jsonMarshaler) || implements(jsonUnmarshaler):
		return "unknown"
	case implements(textMarshaler) || implements(textUnmarshaler):
		return "string"
	}
	return ""
}

// halfEncoded returns the sentence for a type that writes itself and does
// not read itself, or the reverse, as JSON or, when it has neither JSON half,
// as text; "" for any other type. The generated TypeScript has one type for
// both directions, so such a type cannot be described.
func halfEncoded(t reflect.Type) string {
	if t.Kind() == reflect.Pointer || t.Kind() == reflect.Interface || t == timeType {
		return ""
	}
	pt := reflect.PointerTo(t)
	implements := func(i reflect.Type) bool { return t.Implements(i) || pt.Implements(i) }
	for _, e := range []struct {
		as                    string
		writer, reader        reflect.Type
		writeName, readerName string
	}{
		{"JSON", jsonMarshaler, jsonUnmarshaler, "MarshalJSON", "UnmarshalJSON"},
		{"text", textMarshaler, textUnmarshaler, "MarshalText", "UnmarshalText"},
	} {
		const both = ", since the generated TypeScript has one type for both directions."
		switch w, r := implements(e.writer), implements(e.reader); {
		case w && r:
			return ""
		case w:
			return fmt.Sprintf("%s writes itself as %s but does not read itself; give it %s too%s", typeName(t), e.as, e.readerName, both)
		case r:
			return fmt.Sprintf("%s reads itself as %s but does not write itself; give it %s too%s", typeName(t), e.as, e.writeName, both)
		}
	}
	return ""
}

// isBytes reports whether t is a []byte, which JSON carries as a string.
func isBytes(t reflect.Type) bool {
	return t.Kind() == reflect.Slice && t.Elem().Kind() == reflect.Uint8 && selfEncoded(t.Elem()) == ""
}

// isList reports whether an output of type t is a list of values.
func isList(t reflect.Type) bool {
	return (t.Kind() == reflect.Slice || t.Kind() == reflect.Array) && !isBytes(t) && selfEncoded(t) == ""
}

// leaf is what a value of type t holds once pointers, lists and maps are
// looked through; t itself when it encodes itself.
func leaf(t reflect.Type) reflect.Type {
	for t != timeType && selfEncoded(t) == "" && isContainer(t.Kind()) && !isBytes(t) {
		t = t.Elem()
	}
	return t
}

// paramInfo describes a path parameter or query field of a binding plan.
func paramInfo(pp param, query bool, rules map[string][]RuleInfo) ParamInfo {
	info := ParamInfo{Name: pp.key, Optional: query, Style: "value", Rules: rules[indexKey(pp.index)]}
	switch {
	case pp.filter:
		info.TS, info.Style = "Record<string, string>", "filter"
	case pp.list:
		info.TS, info.Style = "string[]", "list"
	case pp.ptr:
		info.TS = kindTS(pp.typ.Elem().Kind())
	default:
		info.TS = kindTS(pp.typ.Kind())
	}
	if info.Rules == nil {
		info.Rules = []RuleInfo{}
	}
	return info
}

// rulesOf returns the rules validation parsed for the struct type t, by the
// index path of each field.
func rulesOf(t reflect.Type) map[string][]RuleInfo {
	m := map[string][]RuleInfo{}
	for _, spec := range specsOf(t) {
		rules := make([]RuleInfo, 0, len(spec.rules))
		for _, r := range spec.rules {
			rules = append(rules, RuleInfo{Name: r.name, Arg: r.arg})
		}
		m[indexKey(spec.index)] = rules
	}
	return m
}

func indexKey(index []int) string { return fmt.Sprint(index) }

// readFields returns the json fields of t, a type a body is read into (an
// input with json fields of its own, a body field's type, or a struct type
// inside one), as the binder lists them: the fields inputFields gives that
// have a json name the binder takes, readonly ones included. Registration
// has refused any other field of such a type, so these are what the plan's
// view holds, with the readonly fields beside them.
func readFields(t reflect.Type) []descField {
	inner, _ := inputFields(t, nil)
	rules := rulesOf(t)
	fields := []descField{}
	for _, f := range inner {
		key, opts, ok := jsonTagOf(f.StructField)
		if !ok || !validJSONName(key) {
			continue
		}
		fields = append(fields, descField{
			key: key, goName: f.Name, typ: f.Type, index: f.index, readonly: isReadonly(f.StructField),
			omits: hasJSONOption(opts, "omitempty") || hasJSONOption(opts, "omitzero"), quoted: hasJSONOption(opts, "string"),
			rules: rulesFor(rules, f.index),
		})
	}
	return fields
}

func rulesFor(rules map[string][]RuleInfo, index []int) []RuleInfo {
	if r := rules[indexKey(index)]; r != nil {
		return r
	}
	return []RuleInfo{}
}

// jsonFields returns the fields encoding/json writes for the struct type t,
// in its order, with their rules: an embedded struct's fields are t's own, a
// field with no json name has its Go name, and of two fields with one name
// the shallower wins, else the one with a json tag, else neither. A field
// with no json tag that is tagged query, path or body is a route's, not the
// JSON's, and is left out.
//
// It exists for the types the binder never plans: a type only answered, and
// what a readonly field holds. Nothing checked such a type at registration,
// so its fields are found as encoding/json finds them; a type a body is read
// into has its fields from readFields, the binder's own list.
func jsonFields(t reflect.Type) []descField {
	type candidate struct {
		descField
		depth  int
		tagged bool
	}
	var all []candidate
	inside := map[reflect.Type]bool{} // the embedded types being walked, so a type that embeds itself ends
	// viaPointer is whether t is reached through an embedded pointer, which
	// may be nil, when encoding/json writes none of its fields.
	var walk func(t reflect.Type, prefix []int, depth int, viaPointer bool)
	walk = func(t reflect.Type, prefix []int, depth int, viaPointer bool) {
		inside[t] = true
		defer delete(inside, t)
		for i := range t.NumField() {
			f := t.Field(i)
			tag := f.Tag.Get("json")
			if tag == "-" {
				continue
			}
			key, opts, _ := strings.Cut(tag, ",")
			opts = "," + opts
			index := append(slices.Clone(prefix), i)
			ft := f.Type
			if ft.Kind() == reflect.Pointer {
				ft = ft.Elem()
			}
			// As encoding/json: an embedded struct, exported or not, is looked
			// inside unless its tag names it; any other unexported field is
			// left out.
			embedsStruct := f.Anonymous && ft.Kind() == reflect.Struct
			if !f.IsExported() && !embedsStruct {
				continue
			}
			if embedsStruct && key == "" {
				if !inside[ft] {
					walk(ft, index, depth+1, viaPointer || f.Type.Kind() == reflect.Pointer)
				}
				continue
			}
			tagged := key != ""
			if !tagged {
				if _, body := f.Tag.Lookup("body"); body || tagName(f, "query") != "" || tagName(f, "path") != "" {
					continue
				}
				key = f.Name
			}
			all = append(all, candidate{descField{
				key: key, goName: f.Name, typ: f.Type, index: index, readonly: isReadonly(f),
				omits: viaPointer || hasJSONOption(opts, "omitempty") || hasJSONOption(opts, "omitzero"), quoted: hasJSONOption(opts, "string"),
			}, depth, tagged})
		}
	}
	walk(t, nil, 0, false)

	keep := make([]bool, len(all))
	byKey := map[string][]int{}
	var keys []string
	for i, c := range all {
		if _, ok := byKey[c.key]; !ok {
			keys = append(keys, c.key)
		}
		byKey[c.key] = append(byKey[c.key], i)
	}
	for _, k := range keys {
		var top, tagged []int
		for _, i := range byKey[k] {
			switch {
			case len(top) == 0 || all[i].depth < all[top[0]].depth:
				top = []int{i}
			case all[i].depth == all[top[0]].depth:
				top = append(top, i)
			}
		}
		for _, i := range top {
			if all[i].tagged {
				tagged = append(tagged, i)
			}
		}
		switch {
		case len(top) == 1:
			keep[top[0]] = true
		case len(tagged) == 1:
			keep[tagged[0]] = true
		}
	}
	rules := rulesOf(t)
	fields := []descField{}
	for i, c := range all {
		if keep[i] {
			f := c.descField
			f.rules = rulesFor(rules, f.index)
			fields = append(fields, f)
		}
	}
	return fields
}

// tsName is a struct type's name in TypeScript: its Go name, and for a
// generic type its base name with its arguments' names after it, so
// List[Product] is ListProduct and Page[[]Product] is PageProductList.
func tsName(t reflect.Type) string { return genericName(t.Name()) }

// genericName writes a type's name as reflect gives it, Base[arg,arg], as
// one name.
func genericName(s string) string {
	i := strings.IndexByte(s, '[')
	if i < 0 || !strings.HasSuffix(s, "]") {
		return s
	}
	name := s[:i]
	for _, arg := range splitArgs(s[i+1 : len(s)-1]) {
		name += argName(arg)
	}
	return name
}

// argName is the name of a type argument written as Go writes it, such as
// github.com/shop/app.Product, []*Product or map[string]int. A pointer is
// its type's name with OrNull after it, so Page[*Address] is PageAddressOrNull
// and never meets Page[Address].
func argName(a string) string {
	a = strings.TrimSpace(a)
	switch {
	case strings.HasPrefix(a, "*"): // may be null, so not the name of the type itself
		return argName(a[1:]) + "OrNull"
	case strings.HasPrefix(a, "["):
		end := closing(a, 0)
		return argName(a[end+1:]) + "List"
	case strings.HasPrefix(a, "map["):
		end := closing(a, 3)
		return "Map" + argName(a[4:end]) + argName(a[end+1:])
	case strings.HasPrefix(a, "interface {"), a == "any":
		return "Any"
	case strings.HasPrefix(a, "struct {"):
		return "Struct"
	}
	head := a
	if i := strings.IndexByte(a, '['); i >= 0 {
		head = a[:i]
	}
	if j := strings.LastIndexByte(head, '.'); j >= 0 {
		a = a[j+1:]
	}
	a = genericName(a)
	r, size := utf8.DecodeRuneInString(a)
	return string(unicode.ToUpper(r)) + strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			return r
		}
		return -1
	}, a[size:])
}

// closing is the index of the ] that closes the [ at open in s.
func closing(s string, open int) int {
	depth := 0
	for i := open; i < len(s); i++ {
		switch s[i] {
		case '[':
			depth++
		case ']':
			if depth--; depth == 0 {
				return i
			}
		}
	}
	return len(s) - 1
}

// splitArgs splits a type's arguments at the commas outside brackets.
func splitArgs(s string) []string {
	var args []string
	depth, start := 0, 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '[', '(', '{':
			depth++
		case ']', ')', '}':
			depth--
		case ',':
			if depth == 0 {
				args = append(args, s[start:i])
				start = i + 1
			}
		}
	}
	return append(args, s[start:])
}
