package warp

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"reflect"
	"slices"
	"strconv"
	"strings"
)

// plan is how a typed route fills its input type: made once, when the route
// is registered.
type plan struct {
	none     bool // the input is None: nothing is read
	readBody bool // POST, PUT and PATCH read a JSON body
	// view is a struct of the json fields alone, which the body is decoded
	// into, so a body can never set a query or path field; raw is the same
	// struct with every field a json.RawMessage, to find an unknown field
	// the decoder met after a wrong type.
	view, raw reflect.Type
	body      [][]int // where view's field i sits in the input
	query     []param
	path      []param
}

// param is a field filled from the query or the path.
type param struct {
	key   string
	index []int
	typ   reflect.Type
	list  bool   // a []string of every value of the key
	ptr   bool   // a pointer, left nil when the key is absent
	text  bool   // a string, which keeps an empty value
	set   setter // converts one value into the field, or into what it points to
	wrong string // the message when a value does not convert
}

// setter converts s and stores it in v, reporting false when s does not
// convert.
type setter func(v reflect.Value, s string) bool

const notJSON = "The body is not valid JSON."

var stringType = reflect.TypeFor[string]()

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
	p := &plan{readBody: method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch}
	var mistakes []string
	wild := wildcards(path)
	hasField := map[string]bool{}
	jsonOwner := map[string]string{}
	var view, raw []reflect.StructField
	for _, f := range inputFields(t, nil, map[reflect.Type]bool{}) {
		at := name + "." + f.Name
		jsonKey, jsonOpts, isJSON := jsonTagOf(f.StructField)
		pathKey, queryKey := tagName(f.StructField, "path"), tagName(f.StructField, "query")
		var tags []string
		if isJSON {
			tags = append(tags, "a json")
		}
		if pathKey != "" {
			tags = append(tags, "a path")
			hasField[pathKey] = true
		}
		if queryKey != "" {
			tags = append(tags, "a query")
		}
		switch {
		case len(tags) == 0:
			mistakes = append(mistakes, fmt.Sprintf("%s has no json, query or path tag; add one.", at))
		case len(tags) == 2:
			mistakes = append(mistakes, fmt.Sprintf("%s has both %s and %s tag; keep one.", at, tags[0], tags[1]))
		case len(tags) == 3:
			mistakes = append(mistakes, fmt.Sprintf("%s has a json, a path and a query tag; keep one.", at))
		case isJSON:
			if !p.readBody {
				mistakes = append(mistakes, fmt.Sprintf("%s is tagged json, and %s reads no body; tag it query or path.", at, method))
				continue
			}
			if other, taken := jsonOwner[jsonKey]; taken {
				mistakes = append(mistakes, fmt.Sprintf("%s and %s both have the json key %q; rename one.", other, at, jsonKey))
				continue
			}
			jsonOwner[jsonKey] = at
			view = append(view, reflect.StructField{
				Name: "F" + strconv.Itoa(len(view)),
				Type: f.Type,
				Tag:  reflect.StructTag("json:" + strconv.Quote(jsonKey+jsonOpts)),
			})
			raw = append(raw, reflect.StructField{
				Name: "F" + strconv.Itoa(len(raw)),
				Type: reflect.TypeFor[json.RawMessage](),
				Tag:  reflect.StructTag("json:" + strconv.Quote(jsonKey)),
			})
			p.body = append(p.body, f.index)
		case pathKey != "":
			if !slices.Contains(wild, pathKey) {
				mistakes = append(mistakes, fmt.Sprintf("%s is tagged path:%q, and the path has no {%s}.", at, pathKey, pathKey))
			}
			prm, ok := paramFor(f.Type, false)
			if !ok {
				mistakes = append(mistakes, cannotFill(at, f.Type))
				continue
			}
			prm.key, prm.index = pathKey, f.index
			p.path = append(p.path, prm)
		default:
			prm, ok := paramFor(f.Type, true)
			if !ok {
				mistakes = append(mistakes, cannotFill(at, f.Type))
				continue
			}
			prm.key, prm.index = queryKey, f.index
			p.query = append(p.query, prm)
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
	}
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

// inputFields lists the fields of t that a request can fill. An embedded
// struct's fields are t's own; an unexported field is left out, and so is an
// embedded pointer to an unexported struct, which cannot be set.
func inputFields(t reflect.Type, prefix []int, seen map[reflect.Type]bool) []inputField {
	if seen[t] {
		return nil
	}
	seen[t] = true
	defer delete(seen, t)
	var out []inputField
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		index := append(slices.Clone(prefix), i)
		if f.Anonymous {
			ft := f.Type
			if ft.Kind() == reflect.Pointer {
				ft = ft.Elem()
			}
			if ft.Kind() == reflect.Struct && ft != timeType {
				if f.Type.Kind() != reflect.Pointer || f.IsExported() {
					out = append(out, inputFields(ft, index, seen)...)
				}
				continue
			}
		}
		if f.IsExported() {
			out = append(out, inputField{f, index})
		}
	}
	return out
}

// jsonTagOf returns the key and options of f's json tag; json:"-" is no tag.
func jsonTagOf(f reflect.StructField) (key, opts string, ok bool) {
	tag, ok := f.Tag.Lookup("json")
	if !ok || tag == "-" {
		return "", "", false
	}
	key, opts, _ = strings.Cut(tag, ",")
	if opts != "" {
		opts = "," + opts
	}
	if key == "" {
		key = f.Name
	}
	return key, opts, true
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
// then the path. It returns the *Error to answer when one of them is wrong.
func (p *plan) fill(w http.ResponseWriter, r *http.Request, dst reflect.Value, limit int64) error {
	if p.none {
		return nil
	}
	if p.readBody {
		if err := p.decode(w, r, dst, limit); err != nil {
			return err
		}
	}
	if len(p.query) > 0 {
		values := r.URL.Query()
		wrong := map[string]string{}
		for _, q := range p.query {
			vals := values[q.key]
			if len(vals) == 0 {
				continue
			}
			field := fieldAt(dst, q.index)
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
			return wrongType(wrong)
		}
	}
	for _, pp := range p.path {
		if !pp.set(fieldAt(dst, pp.index), r.PathValue(pp.key)) {
			return NotFound(fmt.Sprintf("Nothing is at %s.", r.URL.Path))
		}
	}
	return nil
}

// decode reads the JSON body, at most limit bytes, into the input's json
// fields.
func (p *plan) decode(w http.ResponseWriter, r *http.Request, dst reflect.Value, limit int64) error {
	var data []byte
	if r.Body != nil {
		var err error
		data, err = io.ReadAll(http.MaxBytesReader(w, r.Body, limit))
		if err != nil {
			if tooBig := (*http.MaxBytesError)(nil); errors.As(err, &tooBig) {
				return TooLarge(fmt.Sprintf("The body is larger than %s.", sizeOf(limit)))
			}
			return BadRequest(notJSON)
		}
	}
	data = bytes.Trim(data, " \t\r\n")
	if len(data) == 0 {
		data = []byte("{}")
	}
	if !json.Valid(data) {
		return BadRequest(notJSON)
	}
	if data[0] != '{' {
		return BadRequest("The body must be a JSON object.")
	}
	view := reflect.New(p.view)
	if err := decodeStrict(data, view.Interface()); err != nil {
		return p.bodyError(data, err)
	}
	for i, index := range p.body {
		if fv := view.Elem().Field(i); !fv.IsZero() {
			fieldAt(dst, index).Set(fv)
		}
	}
	return nil
}

func decodeStrict(data []byte, v any) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}

// bodyError turns the decoder's error into the answer. The decoder reports
// the first problem it meets, so on a wrong type the body is read again with
// every field taking any value, to put an unknown field first.
func (p *plan) bodyError(data []byte, err error) error {
	var te *json.UnmarshalTypeError
	if errors.As(err, &te) {
		if name, ok := unknownField(decodeStrict(data, reflect.New(p.raw).Interface())); ok {
			return unknown(name)
		}
		if te.Field != "" {
			return wrongType(map[string]string{te.Field: sendWhat(te.Type)})
		}
		return BadRequest(notJSON)
	}
	if name, ok := unknownField(err); ok {
		return unknown(name)
	}
	return BadRequest(notJSON)
}

// unknownField reads the field's name from the decoder's error.
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

// fieldAt returns the field at index in v, making each embedded pointer it
// passes through.
func fieldAt(v reflect.Value, index []int) reflect.Value {
	for i, x := range index {
		if i > 0 && v.Kind() == reflect.Pointer {
			if v.IsNil() {
				v.Set(reflect.New(v.Type().Elem()))
			}
			v = v.Elem()
		}
		v = v.Field(x)
	}
	return v
}
