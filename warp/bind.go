package warp

import (
	"bytes"
	"context"
	"encoding"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"mime"
	"net/http"
	"net/url"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"unicode"
)

// plan is how a typed route fills its input type: made once, when the route
// is registered.
type plan struct {
	none     bool // the input is None: nothing is read
	readBody bool // POST, PUT and PATCH read a JSON body
	partial  bool // PATCH checks only the json fields the body sent
	// view is a struct of the json fields alone, which the body is decoded
	// into, so a body can never set a query or path field, nor a readonly
	// one; raw is the same struct with every field a json.RawMessage, to
	// find an unknown field the decoder met after a wrong type.
	view, raw reflect.Type
	body      [][]int // where view's field i sits in the input
	// keys are the body's keys, readonly ones included, to find which ones
	// the body sent and to refuse a readonly key at any depth.
	keys *object
	// readonly puts back to zero every readonly field the view's values
	// hold, at any depth, once the body is decoded; nil when they hold none.
	readonly *zeroPlan
	query    []param
	path     []param
	// checkSelf says the input has Validate(ctx); checkBody is where its
	// body field is when the body's type has it.
	checkSelf bool
	checkBody []int
	// bodyType is the type the body is read as, for Describe: the body
	// field's, or the input's own when it has json fields; nil when the
	// route reads no body.
	bodyType reflect.Type
}

// param is a field filled from the query or the path.
type param struct {
	key   string
	index []int
	typ   reflect.Type
	list  bool // a []string of every value of the key
	ptr   bool // a pointer, left nil when the key is absent
	text  bool // a string, which keeps an empty value
	// filter is a map[string]string of every key[name], key being the tag's
	// name, such as filter[status]=live.
	filter bool
	set    setter // converts one value into the field, or into what it points to
	wrong  string // the message when a value does not convert
}

// setter converts s and stores it in v, reporting false when s does not
// convert.
type setter func(v reflect.Value, s string) bool

const notJSON = "The body is not valid JSON."

var (
	stringType    = reflect.TypeFor[string]()
	validatorType = reflect.TypeFor[validator]()
)

// validator is an input, or an input's body type, that checks itself once
// its tag rules pass.
type validator interface {
	Validate(ctx context.Context) error
}

// planFor makes the plan for the input type t of a route at method and path,
// or returns every mistake in t as sentences.
func planFor(t reflect.Type, method, path string) (*plan, []string) {
	if t == noneType {
		return &plan{none: true}, nil
	}
	if t.Kind() != reflect.Struct {
		return nil, []string{fmt.Sprintf("the input %s must be a struct or warp.None.", typeName(t))}
	}
	name := typeName(t)
	p := &plan{
		readBody: method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch,
		partial:  method == http.MethodPatch,
	}
	var mistakes []string
	wild := wildcards(path)
	hasField := map[string]bool{}
	owner := map[string]string{} // "json name", "query name", "path name" -> the field that has it
	// claim reports whether at is the first field to have the key name of
	// kind, and records the mistake when it is not.
	claim := func(kind, key, at string) bool {
		if other, taken := owner[kind+" "+key]; taken {
			mistakes = append(mistakes, fmt.Sprintf("%s and %s both have the %s key %q; rename one.", other, at, kind, key))
			return false
		}
		owner[kind+" "+key] = at
		return true
	}
	folds := foldOwners{}
	var view, raw []reflect.StructField
	var top []topField // the body's keys for the scan, readonly ones included
	addView := func(key, opts string, ft reflect.Type, index []int) {
		top = append(top, topField{key, ft, false})
		view = append(view, reflect.StructField{
			Name: "F" + strconv.Itoa(len(view)),
			Type: ft,
			Tag:  reflect.StructTag("json:" + strconv.Quote(key+opts)),
		})
		raw = append(raw, reflect.StructField{
			Name: "F" + strconv.Itoa(len(raw)),
			Type: reflect.TypeFor[json.RawMessage](),
			Tag:  reflect.StructTag("json:" + strconv.Quote(key)),
		})
		p.body = append(p.body, index)
	}
	var bodyField string    // the body field's name, once one is seen
	var jsonFields []string // the names of the input's own json fields
	fields, embedded := inputFields(t, nil)
	mistakes = append(mistakes, embedded...)
	visited := map[reflect.Type]bool{} // the struct types inside json fields, checked once
	for _, f := range fields {
		at := name + "." + f.Name
		jsonKey, jsonOpts, isJSON := jsonTagOf(f.StructField)
		bodyKind, isBody := f.Tag.Lookup("body")
		pathKey, queryKey := tagName(f.StructField, "path"), tagName(f.StructField, "query")
		var tags []string
		if isBody {
			tags = append(tags, "a body")
		}
		if isJSON {
			tags = append(tags, "a json")
			jsonFields = append(jsonFields, f.Name)
		}
		if pathKey != "" {
			tags = append(tags, "a path")
			hasField[pathKey] = true
		}
		if queryKey != "" {
			tags = append(tags, "a query")
		}
		readonly := isReadonly(f.StructField)
		switch {
		case len(tags) == 0:
			mistakes = append(mistakes, fmt.Sprintf("%s has no json, query, path or body tag; add one.", at))
		case len(tags) == 2:
			mistakes = append(mistakes, fmt.Sprintf("%s has both %s and %s tag; keep one.", at, tags[0], tags[1]))
		case len(tags) > 2:
			mistakes = append(mistakes, fmt.Sprintf("%s has %s and %s tag; keep one.", at, strings.Join(tags[:len(tags)-1], ", "), tags[len(tags)-1]))
		case readonly && isBody:
			mistakes = append(mistakes, fmt.Sprintf("%s is tagged body and readonly; a body is always read, so remove readonly.", at))
		case readonly && !isJSON:
			mistakes = append(mistakes, fmt.Sprintf("%s is tagged %s and readonly; readonly is for json fields.", at, strings.TrimPrefix(tags[0], "a ")))
		case isBody:
			switch {
			case !p.readBody:
				mistakes = append(mistakes, fmt.Sprintf("%s is tagged body, and %s reads no body; remove it.", at, method))
			case bodyKind != "json":
				mistakes = append(mistakes, fmt.Sprintf("%s is tagged body:%q; the only body kind is json.", at, bodyKind))
			case f.Type.Kind() != reflect.Struct || f.Type == timeType || decodesItself(f.Type):
				mistakes = append(mistakes, fmt.Sprintf("%s is tagged body:\"json\" and is a %s; make it a struct.", at, f.Type))
			case bodyField != "":
				mistakes = append(mistakes, fmt.Sprintf("%s has two body fields, %s and %s; keep one.", name, bodyField, f.Name))
			default:
				// The body's fields are the input's json fields, at the same
				// index paths as an embedded struct's.
				bodyField = f.Name
				p.bodyType = f.Type
				mistakes = append(mistakes, bodyMistakes(f.Type, visited)...)
				inner, _ := inputFields(f.Type, f.index)
				for _, bf := range inner {
					key, opts, ok := jsonTagOf(bf.StructField)
					switch {
					case !ok || !validJSONName(key):
					case isReadonly(bf.StructField):
						top = append(top, topField{key, bf.Type, true})
					default:
						addView(key, opts, bf.Type, bf.index)
					}
				}
				if reflect.PointerTo(f.Type).Implements(validatorType) {
					p.checkBody = f.index
				}
			}
		case isJSON:
			if !p.readBody {
				mistakes = append(mistakes, fmt.Sprintf("%s is tagged json, and %s reads no body; tag it query or path.", at, method))
				continue
			}
			ok := false
			if m := jsonNameMistake(at, f.Name, jsonKey); m != "" {
				mistakes = append(mistakes, m)
			} else if ok = claim("json", jsonKey, at); ok {
				if m := folds.claim(name, f.Name, jsonKey); m != "" {
					mistakes = append(mistakes, m)
					ok = false
				}
			}
			if hasJSONOption(jsonOpts, "string") {
				mistakes = append(mistakes, stringOption(at))
				ok = false
			}
			if readonly {
				if ok {
					top = append(top, topField{jsonKey, f.Type, true})
				}
				continue // never read, so what is inside it is not the body's
			}
			mistakes = append(mistakes, bodyMistakes(f.Type, visited)...)
			if ok {
				addView(jsonKey, jsonOpts, f.Type, f.index)
			}
		case pathKey != "":
			if !slices.Contains(wild, pathKey) {
				mistakes = append(mistakes, fmt.Sprintf("%s is tagged path:%q, and the path has no {%s}.", at, pathKey, pathKey))
			}
			prm, ok := paramFor(f.Type, false)
			if !ok {
				mistakes = append(mistakes, cannotFill(at, f.Type))
				continue
			}
			if !claim("path", pathKey, at) {
				continue
			}
			prm.key, prm.index = pathKey, f.index
			p.path = append(p.path, prm)
		default:
			prm, ok := paramFor(f.Type, true)
			if f.Type.Kind() == reflect.Map {
				if f.Type.Key() != stringType || f.Type.Elem() != stringType {
					mistakes = append(mistakes, fmt.Sprintf("%s is a %s; a filter map is a map[string]string.", at, f.Type))
					continue
				}
				prm, ok = param{typ: f.Type, filter: true}, true
			}
			if !ok {
				mistakes = append(mistakes, cannotFill(at, f.Type))
				continue
			}
			if !claim("query", queryKey, at) {
				continue
			}
			prm.key, prm.index = queryKey, f.index
			p.query = append(p.query, prm)
		}
	}
	if bodyField != "" {
		for _, j := range jsonFields {
			mistakes = append(mistakes, fmt.Sprintf("%s has a body field and the json field %s; move %s into the body's type.", name, j, j))
		}
	}
	for _, w := range wild {
		if !hasField[w] {
			mistakes = append(mistakes, fmt.Sprintf("the path has {%s}, and %s has no field tagged path:%q.", w, name, w))
		}
	}
	mistakes = append(mistakes, checkRules(t)...)
	if len(mistakes) > 0 {
		return nil, mistakes
	}
	if p.readBody {
		p.view, p.raw = reflect.StructOf(view), reflect.StructOf(raw)
		p.keys = topObject(top)
		p.readonly = readonlyPlanFor(p.view)
		if len(jsonFields) > 0 {
			p.bodyType = t
		}
	}
	p.checkSelf = reflect.PointerTo(t).Implements(validatorType)
	return p, nil
}

func cannotFill(at string, t reflect.Type) string {
	return fmt.Sprintf("%s is a %s, which a query or path value cannot fill; use a string, a number or a bool.", at, t)
}

// typeName is a type's name, or how Go writes it when it has none.
func typeName(t reflect.Type) string {
	if t.Name() != "" {
		return t.Name()
	}
	return t.String()
}

// inputField is an exported field of an input, an embedded struct's included,
// with the index path to it.
type inputField struct {
	reflect.StructField
	index []int
}

// inputFields lists the fields of the struct type t that a request can fill.
// An embedded struct's fields are t's own, and an unexported field is left
// out. Some fields are mistakes, returned as sentences: an unexported field
// tagged body, which would never be filled; an embedded struct tagged body,
// which would not be the body field it looks like; an embedded pointer to a
// struct, which stays nil when the body leaves its fields out, so reading a
// promoted field panics; an embedded struct with a json name, which
// encoding/json reads as a named object while warp makes its fields the
// input's own; and one tagged json:"-", which encoding/json skips.
func inputFields(t reflect.Type, prefix []int) ([]inputField, []string) {
	var out []inputField
	var mistakes []string
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		index := append(slices.Clone(prefix), i)
		if _, body := f.Tag.Lookup("body"); body && f.Anonymous {
			mistakes = append(mistakes, fmt.Sprintf("%s embeds %s tagged body; give the body field a name.", typeName(t), typeName(f.Type)))
			continue
		}
		if f.Anonymous {
			if f.Type.Kind() == reflect.Pointer && f.Type.Elem().Kind() == reflect.Struct && f.Type.Elem() != timeType {
				elem := typeName(f.Type.Elem())
				mistakes = append(mistakes, fmt.Sprintf("%s embeds *%s, a pointer; embed %s by value.", typeName(t), elem, elem))
				continue
			}
			if f.Type.Kind() == reflect.Struct && f.Type != timeType {
				if f.Tag.Get("json") == "-" {
					mistakes = append(mistakes, fmt.Sprintf("%s embeds %s with json:\"-\"; give the field a name to leave it out, or drop the tag to make %s's fields the input's own.",
						typeName(t), typeName(f.Type), typeName(f.Type)))
					continue
				}
				if key, _, ok := jsonTagOf(f); ok && key != "" {
					mistakes = append(mistakes, fmt.Sprintf("%s embeds %s with the json name %q; give the field a name, or drop the tag to make %s's fields the input's own.",
						typeName(t), typeName(f.Type), key, typeName(f.Type)))
					continue
				}
				if isReadonly(f) {
					mistakes = append(mistakes, fmt.Sprintf("%s embeds %s tagged readonly; tag the fields inside it, or give the field a name.", typeName(t), typeName(f.Type)))
				}
				fields, more := inputFields(f.Type, index)
				out, mistakes = append(out, fields...), append(mistakes, more...)
				continue
			}
		}
		if f.IsExported() {
			out = append(out, inputField{f, index})
		} else if _, body := f.Tag.Lookup("body"); body {
			mistakes = append(mistakes, fmt.Sprintf("%s.%s is tagged body and is not exported; export it.", typeName(t), f.Name))
		}
	}
	return out, mistakes
}

var (
	jsonUnmarshaler = reflect.TypeFor[json.Unmarshaler]()
	textUnmarshaler = reflect.TypeFor[encoding.TextUnmarshaler]()
)

// decodesItself reports whether t reads itself from JSON or from text, so
// that what is inside it is its own affair.
func decodesItself(t reflect.Type) bool {
	pt := reflect.PointerTo(t)
	return t.Implements(jsonUnmarshaler) || pt.Implements(jsonUnmarshaler) ||
		t.Implements(textUnmarshaler) || pt.Implements(textUnmarshaler)
}

// bodyMistakes returns the mistakes in the struct types a json field of type
// t holds, through pointers, slices, arrays and map values, each type checked
// once. Inside the body only json tags make sense: every exported field needs
// a usable json name, json:"-" leaves a field out, and a query, path or body
// tag is a mistake. time.Time, a type that decodes itself and what a readonly
// field holds are not looked inside.
func bodyMistakes(t reflect.Type, visited map[reflect.Type]bool) []string {
	for !decodesItself(t) && isContainer(t.Kind()) {
		t = t.Elem()
	}
	if t != timeType && decodesItself(t) {
		return readonlyInside(t, t, visited)
	}
	if t.Kind() != reflect.Struct || t == timeType || visited[t] {
		return nil
	}
	visited[t] = true
	name := typeName(t)
	fields, mistakes := inputFields(t, nil)
	owner := map[string]string{}
	folds := foldOwners{}
	for _, f := range fields {
		at := name + "." + f.Name
		key, opts, hasJSON := jsonTagOf(f.StructField)
		if !hasJSON && f.Tag.Get("json") == "-" {
			continue
		}
		if hasJSONOption(opts, "string") {
			mistakes = append(mistakes, stringOption(at))
		}
		bad := false
		if _, ok := f.Tag.Lookup("body"); ok {
			mistakes = append(mistakes, fmt.Sprintf("%s is tagged body inside a body field; only the route's input itself has a body field.", at))
			bad = true
		}
		for _, tag := range []string{"query", "path"} {
			if tagName(f.StructField, tag) != "" {
				mistakes = append(mistakes, fmt.Sprintf("%s is tagged %s inside a body field; only the route's input itself takes query and path tags.", at, tag))
				bad = true
			}
		}
		switch {
		case !hasJSON:
			if !bad {
				mistakes = append(mistakes, fmt.Sprintf("%s has no json tag; add one.", at))
			}
		case jsonNameMistake(at, f.Name, key) != "":
			mistakes = append(mistakes, jsonNameMistake(at, f.Name, key))
		case owner[key] != "":
			mistakes = append(mistakes, fmt.Sprintf("%s and %s both have the json key %q; rename one.", owner[key], at, key))
		default:
			owner[key] = at
			if m := folds.claim(name, f.Name, key); m != "" {
				mistakes = append(mistakes, m)
			}
		}
		if !isReadonly(f.StructField) { // never read, so what is inside it is not the body's
			mistakes = append(mistakes, bodyMistakes(f.Type, visited)...)
		}
	}
	return mistakes
}

// foldOwners are the keys of one object, folded as encoding/json folds them
// (appendFolded), to the Go name of the field that has each.
type foldOwners map[string]string

// claim records the field named field, of the type named in, with key, and
// returns the mistake when another field of the object has a key that
// differs from it only in case. encoding/json would set the one or the other
// for a key sent in another case, and a decoder may choose either (jsonv2
// takes the shallowest), so the body scan would have to read the value as
// both, at every level; with one field to each folded key it reads each
// value once.
func (o foldOwners) claim(in, field, key string) string {
	f := string(appendFolded(nil, []byte(key)))
	if other, taken := o[f]; taken {
		return fmt.Sprintf("%s has the fields %s and %s, whose keys differ only in case; rename one.", in, other, field)
	}
	o[f] = field
	return ""
}

// readonlyInside returns the mistakes of the type t holds, inside dec, a
// type that decodes itself: a field tagged readonly there, at any depth, is
// one, since Warp cannot see what dec's own decoder sets, so it can neither
// refuse the key nor put the field back to zero.
func readonlyInside(dec, t reflect.Type, visited map[reflect.Type]bool) []string {
	for isContainer(t.Kind()) {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct || t == timeType || visited[t] {
		return nil
	}
	visited[t] = true
	var mistakes []string
	for i := range t.NumField() {
		f := t.Field(i)
		if !f.IsExported() && !f.Anonymous || f.Tag.Get("json") == "-" {
			continue
		}
		if isReadonly(f) {
			mistakes = append(mistakes, fmt.Sprintf("%s decodes itself, so Warp cannot see what it sets, and %s.%s is tagged readonly; drop the tag or the type's own decoding.",
				typeName(dec), typeName(t), f.Name))
			continue
		}
		mistakes = append(mistakes, readonlyInside(dec, f.Type, visited)...)
	}
	return mistakes
}

func isContainer(k reflect.Kind) bool {
	return k == reflect.Pointer || k == reflect.Slice || k == reflect.Array || k == reflect.Map
}

// jsonNameMistake returns the sentence for a json key that the field at, whose
// Go name is field, cannot use, or "".
func jsonNameMistake(at, field, key string) string {
	switch {
	case key == "":
		return fmt.Sprintf("%s has a json tag with no name; name it, as in json:%q.", at, snakeCase(field))
	case !validJSONName(key):
		return fmt.Sprintf("%s has the json name %q, which JSON tags cannot carry; use letters, digits, _, - and .", at, key)
	}
	return ""
}

// jsonTagOf returns the key and options of f's json tag, the key "" when the
// tag has no name; json:"-" is no tag.
func jsonTagOf(f reflect.StructField) (key, opts string, ok bool) {
	tag, ok := f.Tag.Lookup("json")
	if !ok || tag == "-" {
		return "", "", false
	}
	key, opts, _ = strings.Cut(tag, ",")
	if opts != "" {
		opts = "," + opts
	}
	return key, opts, true
}

// hasJSONOption reports whether opts, a json tag's options as jsonTagOf gives
// them (",omitempty,string"), have option.
func hasJSONOption(opts, option string) bool {
	for o := range strings.SplitSeq(opts, ",") {
		if o == option {
			return true
		}
	}
	return false
}

// stringOption is the mistake of the field at, whose json tag has ,string:
// encoding/json would send and read its number or bool as a quoted string,
// and the generated TypeScript says number.
func stringOption(at string) string {
	return fmt.Sprintf("%s has json:\",string\"; remove ,string, since Warp sends and reads a number as a number.", at)
}

// validJSONName reports whether name can be a json key: letters, digits, _,
// - and ., but not the name - alone. encoding/json takes all of these as a
// tag name; it takes a few more, which a typed route has no need of.
func validJSONName(name string) bool {
	if name == "" || name == "-" {
		return false
	}
	for _, r := range name {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && !strings.ContainsRune("_-.", r) {
			return false
		}
	}
	return true
}

// snakeCase writes a Go field name as a json key: UserID is user_id.
func snakeCase(name string) string {
	rs := []rune(name)
	var b strings.Builder
	for i, r := range rs {
		if unicode.IsUpper(r) {
			if i > 0 && (unicode.IsLower(rs[i-1]) || unicode.IsDigit(rs[i-1]) ||
				(i+1 < len(rs) && unicode.IsUpper(rs[i-1]) && unicode.IsLower(rs[i+1]))) {
				b.WriteByte('_')
			}
			r = unicode.ToLower(r)
		}
		b.WriteRune(r)
	}
	return b.String()
}

// tagName is the name in f's tag called tag; "" and "-" are no name.
func tagName(f reflect.StructField, tag string) string {
	name, _, _ := strings.Cut(f.Tag.Get(tag), ",")
	if name == "-" {
		return ""
	}
	return name
}

// wildcards returns the names of the path's wildcards, {name} and {name...},
// in order; {$} is not one.
func wildcards(path string) []string {
	var names []string
	for rest := path; ; {
		i := strings.IndexByte(rest, '{')
		if i < 0 {
			return names
		}
		j := strings.IndexByte(rest[i:], '}')
		if j < 0 {
			return names
		}
		if name := strings.TrimSuffix(rest[i+1:i+j], "..."); name != "$" {
			names = append(names, name)
		}
		rest = rest[i+j+1:]
	}
}

// paramFor returns how to fill a field of type t from a query value, or from
// a path value when query is false.
func paramFor(t reflect.Type, query bool) (param, bool) {
	if set, wrong, ok := scalar(t); ok {
		return param{typ: t, set: set, wrong: wrong, text: t.Kind() == reflect.String}, true
	}
	if !query {
		return param{}, false
	}
	if t.Kind() == reflect.Slice && t.Elem() == stringType {
		return param{typ: t, list: true}, true
	}
	if t.Kind() == reflect.Pointer {
		if set, wrong, ok := scalar(t.Elem()); ok {
			return param{typ: t, ptr: true, set: set, wrong: wrong, text: t.Elem().Kind() == reflect.String}, true
		}
	}
	return param{}, false
}

// scalar returns the setter for a string, bool, int, uint or float kind, and
// the message when a value does not convert.
func scalar(t reflect.Type) (setter, string, bool) {
	switch t.Kind() {
	case reflect.String:
		return func(v reflect.Value, s string) bool { v.SetString(s); return true }, "", true
	case reflect.Bool:
		return func(v reflect.Value, s string) bool {
			b, err := strconv.ParseBool(s)
			if err != nil {
				return false
			}
			v.SetBool(b)
			return true
		}, "Send true or false.", true
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		bits := t.Bits()
		return func(v reflect.Value, s string) bool {
			n, err := strconv.ParseInt(s, 10, bits)
			if err != nil {
				return false
			}
			v.SetInt(n)
			return true
		}, "Send a whole number.", true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		bits := t.Bits()
		return func(v reflect.Value, s string) bool {
			n, err := strconv.ParseUint(s, 10, bits)
			if err != nil {
				return false
			}
			v.SetUint(n)
			return true
		}, "Send a whole number.", true
	case reflect.Float32, reflect.Float64:
		bits := t.Bits()
		return func(v reflect.Value, s string) bool {
			f, err := strconv.ParseFloat(s, bits)
			if err != nil || math.IsNaN(f) || math.IsInf(f, 0) {
				return false
			}
			v.SetFloat(f)
			return true
		}, "Send a number.", true
	}
	return nil, "", false
}

// fill sets dst, the input, from r by the plan: the body, then the query,
// then the path. It returns the top-level keys the body sent, as walk gives
// them, or the *Error to answer when one of them is wrong.
func (p *plan) fill(w http.ResponseWriter, r *http.Request, dst reflect.Value, limit int64) ([]string, error) {
	if p.none {
		return nil, nil
	}
	var sent []string
	if p.readBody {
		var err error
		if sent, err = p.decode(w, r, dst, limit); err != nil {
			return nil, err
		}
	}
	if len(p.query) > 0 {
		values := r.URL.Query()
		wrong := map[string]string{}
		for _, q := range p.query {
			if q.filter {
				m, err := filterOf(values, q.key)
				if err != nil {
					return nil, err
				}
				if m != nil {
					dst.FieldByIndex(q.index).Set(reflect.ValueOf(m).Convert(q.typ))
				}
				continue
			}
			vals := values[q.key]
			if len(vals) == 0 {
				continue
			}
			field := dst.FieldByIndex(q.index)
			switch {
			case q.list:
				field.Set(reflect.ValueOf(slices.Clone(vals)).Convert(q.typ))
			case vals[0] == "" && !q.text:
				// An empty number or bool is as if it were not sent.
			case q.ptr:
				v := reflect.New(q.typ.Elem())
				if !q.set(v.Elem(), vals[0]) {
					wrong[q.key] = q.wrong
					continue
				}
				field.Set(v)
			default:
				if !q.set(field, vals[0]) {
					wrong[q.key] = q.wrong
				}
			}
		}
		if len(wrong) > 0 {
			return nil, wrongType(wrong)
		}
	}
	for _, pp := range p.path {
		if !pp.set(dst.FieldByIndex(pp.index), r.PathValue(pp.key)) {
			return nil, NotFound(fmt.Sprintf("Nothing is at %s.", r.URL.Path))
		}
	}
	return sent, nil
}

// The bounds on a filter map: at most maxFilterKeys keys, none longer than
// maxFilterKey bytes.
const (
	maxFilterKeys = 32
	maxFilterKey  = 64
)

// filterOf returns the map of every prefix[name] in values, name to its first
// value, or nil when there is none. The keys are counted and measured before
// the map is made. A key that is not prefix[name], such as prefix[] or
// prefix[a, is no filter key, and is ignored as any other key is.
func filterOf(values url.Values, prefix string) (map[string]string, error) {
	n, long := 0, false
	for k := range values {
		if name, ok := filterKey(k, prefix); ok {
			n++
			long = long || len(name) > maxFilterKey
		}
	}
	switch {
	case n > maxFilterKeys:
		return nil, BadRequest(fmt.Sprintf("The query has more %s[...] keys than a route takes (%d).", prefix, maxFilterKeys))
	case long:
		return nil, BadRequest(fmt.Sprintf("A %s[...] key is longer than %d bytes.", prefix, maxFilterKey))
	case n == 0:
		return nil, nil
	}
	m := make(map[string]string, n)
	for k, vals := range values {
		if name, ok := filterKey(k, prefix); ok {
			m[name] = vals[0]
		}
	}
	return m, nil
}

// filterKey returns name when k is prefix[name], name not empty and with no
// bracket in it.
func filterKey(k, prefix string) (string, bool) {
	rest, ok := strings.CutPrefix(k, prefix)
	if !ok || len(rest) < 3 || rest[0] != '[' || rest[len(rest)-1] != ']' {
		return "", false
	}
	name := rest[1 : len(rest)-1]
	return name, !strings.ContainsAny(name, "[]")
}

// decode reads the JSON body, at most limit bytes, into the input's json
// fields, and returns the top-level keys it sent.
func (p *plan) decode(w http.ResponseWriter, r *http.Request, dst reflect.Value, limit int64) ([]string, error) {
	var data []byte
	if r.Body != nil {
		var err error
		data, err = io.ReadAll(http.MaxBytesReader(w, r.Body, limit))
		if err != nil {
			if tooBig := (*http.MaxBytesError)(nil); errors.As(err, &tooBig) {
				return nil, TooLarge(fmt.Sprintf("The body is larger than %s.", sizeOf(limit)))
			}
			return nil, BadRequest(notJSON)
		}
	}
	data = bytes.Trim(data, " \t\r\n")
	if len(data) == 0 {
		data = []byte("{}")
	} else if !isJSONMedia(r.Header.Get("Content-Type")) {
		return nil, UnsupportedMedia("Send the body as application/json.")
	}
	if tooDeep(data, maxNesting) {
		return nil, BadRequest(fmt.Sprintf("The body is nested more than %d levels deep.", maxNesting))
	}
	if !json.Valid(data) {
		return nil, BadRequest(notJSON)
	}
	if data[0] != '{' {
		return nil, BadRequest("The body must be a JSON object.")
	}
	view := reflect.New(p.view)
	if err := decodeStrict(data, view.Interface()); err != nil {
		// A readonly key is refused as one the route does not take, whatever
		// the type of its value, before the decoder's complaint about it.
		if _, readonly := p.walk(data); readonly != nil {
			return nil, readonly
		}
		return nil, p.bodyError(data, err)
	}
	sent, err := p.walk(data)
	if err != nil {
		return nil, err
	}
	// The scan refuses a readonly key, matching keys as encoding/json does.
	// Should a decoder ever match one the scan did not (another version of
	// encoding/json, GOEXPERIMENT=jsonv2), the field is still never set:
	// every readonly field is put back to zero here.
	if p.readonly != nil {
		p.readonly.zero(view.Elem())
	}
	for i, index := range p.body {
		dst.FieldByIndex(index).Set(view.Elem().Field(i))
	}
	return sent, nil
}

// isJSONMedia reports whether a Content-Type is application/json or a type
// ending in +json, such as application/merge-patch+json, with any
// parameters. A cross-site form or fetch can send text/plain with the
// visitor's cookies and no preflight; taking JSON alone keeps another site
// from sending a typed route a body. It does not protect a route that takes
// no body, which is answered whatever the Content-Type.
func isJSONMedia(contentType string) bool {
	media, _, err := mime.ParseMediaType(contentType)
	return err == nil && (media == "application/json" ||
		strings.Contains(media, "/") && strings.HasSuffix(media, "+json"))
}

// maxNesting is how deep a typed route's body may nest objects and lists,
// the body's own object counted: deeper bodies are refused before decoding.
const maxNesting = 64

// tooDeep reports whether data nests objects and lists more than max deep,
// counting { and [ outside strings in one pass, with no allocation.
func tooDeep(data []byte, max int) bool {
	depth := 0
	inString, escaped := false, false
	for _, c := range data {
		switch {
		case inString:
			switch {
			case escaped:
				escaped = false
			case c == '\\':
				escaped = true
			case c == '"':
				inString = false
			}
		case c == '"':
			inString = true
		case c == '{' || c == '[':
			if depth++; depth > max {
				return true
			}
		case c == '}' || c == ']':
			depth--
		}
	}
	return false
}

func decodeStrict(data []byte, v any) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}

// bodyError turns the decoder's error into the answer. The decoder reports
// the first problem it meets, so on a wrong type the body is read again with
// every field taking any value, to put an unknown field first. An error that
// is none of a syntax error, an unknown field and a wrong type comes from a
// field's own decoder, as time.Time's on "tomorrow".
func (p *plan) bodyError(data []byte, err error) error {
	var te *json.UnmarshalTypeError
	if errors.As(err, &te) {
		if name, ok := unknownField(decodeStrict(data, reflect.New(p.raw).Interface())); ok {
			return unknown(name)
		}
		// Under GOEXPERIMENT=jsonv2 a field's own decoder refusing a value
		// (base64 for bytes) is a type error that wraps its cause; encoding/json
		// v1 returns the cause itself.
		if errors.Unwrap(te) != nil {
			return BadRequest(cannotTake)
		}
		if te.Field != "" {
			return wrongType(map[string]string{fieldPath(p.view, te.Field): sendWhat(te.Type)})
		}
		return BadRequest(notJSON)
	}
	if name, ok := unknownField(err); ok {
		return unknown(name)
	}
	if se := (*json.SyntaxError)(nil); errors.As(err, &se) {
		return BadRequest(notJSON)
	}
	return BadRequest(cannotTake)
}

const cannotTake = "The body has a value its field cannot take."

// fieldPath is the path of a type error's field, field as the decoder gives
// it in a value of type t, keeping only the keys of struct fields: a list's
// index and a map's key are left out ("tags", not "tags.0"), as encoding/json
// v1 writes it and jsonv2 does not.
func fieldPath(t reflect.Type, field string) string {
	var kept []string
	for seg := range strings.SplitSeq(field, ".") {
		for t != nil && t.Kind() == reflect.Pointer {
			t = t.Elem()
		}
		switch {
		case t == nil:
			kept = append(kept, seg)
		case t.Kind() == reflect.Slice || t.Kind() == reflect.Array || t.Kind() == reflect.Map:
			t = t.Elem() // seg is an index or a key
		case t.Kind() == reflect.Struct:
			kept = append(kept, seg)
			t = fieldByJSONKey(t, seg)
		default:
			kept = append(kept, seg)
			t = nil
		}
	}
	return strings.Join(kept, ".")
}

// fieldByJSONKey is the type of t's field whose json key is key, embedded
// structs' included, or nil.
func fieldByJSONKey(t reflect.Type, key string) reflect.Type {
	fields, _ := inputFields(t, nil)
	for _, f := range fields {
		if k, _, ok := jsonTagOf(f.StructField); ok && (k == key || strings.EqualFold(k, key)) {
			return f.Type
		}
	}
	return nil
}

// unknownField reads the field's name from the decoder's error. encoding/json
// has no error type for an unknown field, so this depends on its wording,
// json: unknown field "name"; when the wording changes, the unknown-field
// rows of TestBodyFailures and TestBodyCannotSetPathField fail.
func unknownField(err error) (string, bool) {
	if err == nil {
		return "", false
	}
	quoted, ok := strings.CutPrefix(err.Error(), "json: unknown field ")
	if !ok {
		return "", false
	}
	name, uerr := strconv.Unquote(quoted)
	return name, uerr == nil
}

func unknown(name string) *Error {
	return BadRequest(fmt.Sprintf("The body has the field %q, which this route does not take.", name))
}

// wrongType is the 400 answer naming each field whose value has the wrong
// type.
func wrongType(fields map[string]string) *Error {
	e := BadRequest(fmt.Sprintf("%d fields have the wrong type.", len(fields)))
	if len(fields) == 1 {
		e.Message = "1 field has the wrong type."
	}
	e.Fields = fields
	return e
}

// sendWhat says what kind of value a field of type t takes.
func sendWhat(t reflect.Type) string {
	for t != nil && t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	switch {
	case t == nil:
	case t.Kind() == reflect.String:
		return "Send text."
	case isNumberKind(t.Kind()):
		return "Send a number."
	case t.Kind() == reflect.Bool:
		return "Send true or false."
	case t.Kind() == reflect.Slice || t.Kind() == reflect.Array:
		return "Send a list."
	case t.Kind() == reflect.Struct || t.Kind() == reflect.Map:
		return "Send an object."
	}
	return "Send a different kind of value."
}

// sizeOf writes a body limit in whole megabytes, or in bytes when it is not
// a whole number of them.
func sizeOf(limit int64) string {
	if limit >= 1<<20 && limit%(1<<20) == 0 {
		return fmt.Sprintf("%d MB", limit>>20)
	}
	return fmt.Sprintf("%d bytes", limit)
}
