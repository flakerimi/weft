package warp

import (
	"fmt"
	"math"
	"net/mail"
	"net/url"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

// rule is one comma-separated part of a warp tag.
type rule struct {
	name  string
	arg   string  // what follows "=", as written
	num   float64 // arg as a number, for min and max
	items []string
}

// fieldSpec is a validated field of a struct type: its key, where it sits, its
// rules and the mistakes found in them. A readonly field has no rules and is
// never validated; json says the key is the field's json name, so a body
// can send it.
type fieldSpec struct {
	key      string
	index    []int
	rules    []rule
	readonly bool
	json     bool
	problems []string
}

var (
	specCache sync.Map // reflect.Type -> []fieldSpec
	timeType  = reflect.TypeOf(time.Time{})
)

const ruleList = "required, email, url, min=, max=, enum= or unique"

// The limits on what a 422 lists, so that a small body cannot make a huge
// answer: at most maxListed fields, at most maxKeyBytes of key bytes in all,
// and never a key longer than maxKeyLen.
const (
	maxListed   = 100
	maxKeyBytes = 16 << 10
	maxKeyLen   = 1 << 10
)

// Validate checks v, a struct or a pointer to one, against the warp tags of
// its fields. It returns nil, or an *Error from Invalid that names each field
// that needs attention: at most 100 of them, with at most 16 KiB of keys in
// all, and none whose key is longer than 1 KiB. When a field is left out the
// message says so, and the walk stops once the 100 or the 16 KiB are spent.
// Fields are listed in a fixed order: struct fields as declared, list entries
// by index, map entries by key. A value with a pointer cycle is not supported.
func Validate(v any) error {
	return validate(reflect.ValueOf(v), false, nil)
}

// validate is Validate on rv. When partial, a json field of the struct
// itself is checked only when its key is in sent, the keys the body had; its
// query and path fields, and everything inside a field that is checked, are
// checked as Validate checks them.
func validate(rv reflect.Value, partial bool, sent []string) error {
	rv = derefValue(rv)
	if !rv.IsValid() || rv.Kind() != reflect.Struct {
		return nil
	}
	w := walker{partial: partial, sent: sent}
	w.validateStruct(rv)
	if len(w.fields) == 0 && !w.left {
		return nil
	}
	e := Invalid(w.fields)
	switch {
	case w.left && len(w.fields) == 0:
		e.Message, e.Fields = "More fields need attention than can be listed.", nil
	case w.left:
		e.Message = fmt.Sprintf("More fields need attention than are listed; the first %d are listed.", len(w.fields))
	}
	return e
}

// segment is one step of a field's key: a field's or a map entry's name, or a
// list index when index is not negative; end is the length of the key up to
// and with this step. The walk keeps the steps to where it is as a stack,
// pushing one before it goes in and popping it after, and writes them out as
// a key only for a field that is listed: a value that passes costs no string,
// and a key's length is known without building it. (A chain of segments
// linked to their parents would not stay on the stack: Go moves a value whose
// address flows back into itself through recursion to the heap.)
type segment struct {
	name  string
	index int
	end   int
}

// key writes the steps as a dotted key, such as lines.0.sku; no steps is "".
func key(at []segment) string {
	if len(at) == 0 {
		return ""
	}
	var digits [20]byte
	var b strings.Builder
	b.Grow(at[len(at)-1].end)
	for i, s := range at {
		if i > 0 {
			b.WriteByte('.')
		}
		if s.index >= 0 {
			b.Write(strconv.AppendInt(digits[:0], int64(s.index), 10))
		} else {
			b.WriteString(s.name)
		}
	}
	return b.String()
}

// walker is one run of Validate.
type walker struct {
	at     []segment
	fields map[string]string
	bytes  int  // the key bytes listed so far
	left   bool // a failing field was not listed
	done   bool // stop walking: the limits are spent or, probing, a field failed
	probe  bool // only find whether anything fails, listing nothing
	// partial checks only the json fields in sent at the top, for PATCH.
	partial bool
	sent    []string
}

// push makes the step name, or the list index when index is not negative,
// the one after the first depth steps.
func (w *walker) push(depth int, name string, index int) {
	n := len(name)
	if index >= 0 {
		n = 1
		for i := index; i >= 10; i /= 10 {
			n++
		}
	}
	if depth > 0 {
		n += w.at[depth-1].end + 1
	}
	w.at = append(w.at[:depth], segment{name: name, index: index, end: n})
}

// fail records that the field at the top of the stack fails with msg, if
// the limits let it be listed.
func (w *walker) fail(msg string) {
	if w.probe {
		w.done = true
		return
	}
	n := 0
	if len(w.at) > 0 {
		n = w.at[len(w.at)-1].end
	}
	switch {
	case n > maxKeyLen:
		w.left = true
	case len(w.fields) == maxListed || w.bytes+n > maxKeyBytes:
		w.left, w.done = true, true
	default:
		if w.fields == nil {
			w.fields = map[string]string{}
		}
		w.fields[key(w.at)] = msg
		w.bytes += n
	}
}

func (w *walker) validateStruct(rv reflect.Value) {
	depth := len(w.at)
	defer func() { w.at = w.at[:depth] }()
	for _, spec := range specsOf(rv.Type()) {
		if w.done {
			return
		}
		if spec.readonly || depth == 0 && w.partial && spec.json && !slices.Contains(w.sent, spec.key) {
			continue
		}
		fv, ok := fieldByIndex(rv, spec.index)
		if !ok {
			continue
		}
		w.push(depth, spec.key, -1)
		if msg := checkField(fv, spec.rules); msg != "" {
			w.fail(msg)
		}
		if !holdsStructs(fv.Type()) {
			continue
		}
		if inner, ok := reachable(fv); ok {
			w.validateInside(inner)
		}
	}
}

// validateInside validates the structs in v, a struct, or a slice, array or
// map whose element type can hold them, as reachable gives it. A list's
// entries are keyed by their index, a map's by its keys, formatted with
// fmt.Sprint. The stack of steps is as long again when it returns.
//
// A map's entries are listed in sorted key order, so that the same fields
// are listed on every run, and cheaply: each value is copied in turn into one
// reused value and probed for a failure, which builds nothing; only the
// entries that fail have their keys kept, sorted, and walked again to list
// what fails in them.
func (w *walker) validateInside(v reflect.Value) {
	depth := len(w.at)
	defer func() { w.at = w.at[:depth] }()
	switch v.Kind() {
	case reflect.Struct:
		w.validateStruct(v)
	case reflect.Slice, reflect.Array:
		if !holdsStructs(v.Type().Elem()) {
			return
		}
		for i := 0; i < v.Len() && !w.done; i++ {
			if inner, ok := reachable(v.Index(i)); ok {
				w.push(depth, "", i)
				w.validateInside(inner)
			}
		}
	case reflect.Map:
		if !holdsStructs(v.Type().Elem()) {
			return
		}
		type entry struct {
			name string
			key  reflect.Value
		}
		var failing []entry
		stringKeys := v.Type().Key().Kind() == reflect.String
		probe := reflect.New(v.Type().Elem()).Elem()
		for iter := v.MapRange(); iter.Next() && !w.done; {
			probe.SetIterValue(iter)
			inner, ok := reachable(probe)
			if !ok {
				continue
			}
			if w.probe {
				w.validateInside(inner)
				continue
			}
			w.probe = true
			w.validateInside(inner)
			fails := w.done
			w.probe, w.done = false, false
			if !fails {
				continue
			}
			k := iter.Key()
			name := ""
			if stringKeys {
				name = k.String()
			} else {
				name = fmt.Sprint(k.Interface())
			}
			failing = append(failing, entry{name, k})
		}
		slices.SortFunc(failing, func(a, b entry) int { return strings.Compare(a.name, b.name) })
		for _, e := range failing {
			if w.done {
				return
			}
			inner, _ := reachable(v.MapIndex(e.key))
			w.push(depth, e.name, -1)
			w.validateInside(inner)
		}
	}
}

// reachable returns what validation looks inside in v, following non-nil
// pointers, and false when v holds nothing to validate. An interface is
// looked inside only when it holds a struct or a non-nil pointer to one, a
// value built by hand: decoded JSON never puts a struct in an interface, so a
// decoded any is a leaf whatever its size.
func reachable(v reflect.Value) (reflect.Value, bool) {
	for v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return reflect.Value{}, false
		}
		v = v.Elem()
	}
	if v.Kind() == reflect.Interface {
		if v.IsNil() {
			return reflect.Value{}, false
		}
		v = v.Elem()
		if v.Kind() == reflect.Pointer {
			if v.IsNil() {
				return reflect.Value{}, false
			}
			v = v.Elem()
		}
		return v, isDescendable(v)
	}
	switch v.Kind() {
	case reflect.Struct:
		return v, isDescendable(v)
	case reflect.Slice, reflect.Array, reflect.Map:
		return v, v.Len() > 0
	}
	return reflect.Value{}, false
}

// holdsStructs reports whether a value of type t can be, or hold through
// pointers, slices, arrays and map values, a struct to validate. An interface
// can, when a struct was put in it by hand.
func holdsStructs(t reflect.Type) bool {
	for isContainer(t.Kind()) {
		t = t.Elem()
	}
	return t.Kind() == reflect.Interface || t.Kind() == reflect.Struct && t != timeType
}

func isDescendable(v reflect.Value) bool {
	return v.IsValid() && v.Kind() == reflect.Struct && v.Type() != timeType
}

// derefValue follows pointers and interfaces; a nil one gives the zero Value.
func derefValue(v reflect.Value) reflect.Value {
	for v.IsValid() && (v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface) {
		if v.IsNil() {
			return reflect.Value{}
		}
		v = v.Elem()
	}
	return v
}

// fieldByIndex walks an index path and reports false when it meets a nil
// embedded pointer.
func fieldByIndex(rv reflect.Value, index []int) (reflect.Value, bool) {
	for i, x := range index {
		if i > 0 {
			for rv.Kind() == reflect.Pointer {
				if rv.IsNil() {
					return reflect.Value{}, false
				}
				rv = rv.Elem()
			}
		}
		rv = rv.Field(x)
	}
	return rv, true
}

func isNumberKind(k reflect.Kind) bool {
	switch k {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	}
	return false
}

func toFloat(v reflect.Value) float64 {
	switch {
	case v.CanInt():
		return float64(v.Int())
	case v.CanUint():
		return float64(v.Uint())
	}
	return v.Float()
}

// isBlank is the test for required: a string is blank when it is empty once
// trimmed.
func isBlank(v reflect.Value) bool {
	if v.Kind() == reflect.String {
		return strings.TrimSpace(v.String()) == ""
	}
	return isEmpty(v)
}

// isEmpty is the test that lets every other rule pass: a value that was not
// given. A string of spaces is not empty.
func isEmpty(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Slice, reflect.Map:
		return v.Len() == 0
	case reflect.Struct:
		if v.Type() == timeType {
			return v.Interface().(time.Time).IsZero()
		}
	}
	return v.IsZero()
}

// checkField returns the message of the first rule the value fails, or "".
// A pointer that is not nil says the value was sent, even when it is zero:
// required passes, and every other rule applies to what it points at. A
// value not behind a pointer that is empty passes every rule but required.
func checkField(fv reflect.Value, rules []rule) string {
	v := derefValue(fv)
	sent := fv.Kind() == reflect.Pointer && !fv.IsNil()
	for _, r := range rules {
		if r.name == "required" {
			if sent {
				continue
			}
			if !v.IsValid() || isBlank(v) {
				return "This is required."
			}
			continue
		}
		if !v.IsValid() || !sent && isEmpty(v) {
			continue
		}
		if msg := checkRule(v, r); msg != "" {
			return msg
		}
	}
	return ""
}

func checkRule(v reflect.Value, r rule) string {
	k := v.Kind()
	switch r.name {
	case "email":
		if k != reflect.String {
			return ""
		}
		a, err := mail.ParseAddress(v.String())
		if err != nil || a.Name != "" {
			return "Enter an email address."
		}
	case "url":
		if k != reflect.String {
			return ""
		}
		u, err := url.Parse(v.String())
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return "Enter a web address that starts with http:// or https://."
		}
	case "enum":
		if k != reflect.String {
			return ""
		}
		for _, item := range r.items {
			if v.String() == item {
				return ""
			}
		}
		return "Choose one of: " + strings.Join(r.items, ", ") + "."
	case "min", "max":
		var size float64
		var low, high string
		switch {
		case k == reflect.String:
			size = float64(utf8.RuneCountInString(v.String()))
			low, high = "Use at least %s characters.", "Use at most %s characters."
		case isNumberKind(k):
			size = toFloat(v)
			low, high = "Enter %s or more.", "Enter %s or less."
		case k == reflect.Slice || k == reflect.Array:
			size = float64(v.Len())
			low, high = "Choose at least %s.", "Choose at most %s."
		default:
			return ""
		}
		if r.name == "min" && size < r.num {
			return fmt.Sprintf(low, r.arg)
		}
		if r.name == "max" && size > r.num {
			return fmt.Sprintf(high, r.arg)
		}
	}
	return ""
}

// specsOf returns the validated fields of a struct type, parsed once.
func specsOf(t reflect.Type) []fieldSpec {
	if cached, ok := specCache.Load(t); ok {
		return cached.([]fieldSpec)
	}
	specs := collectSpecs(t, nil, map[reflect.Type]bool{})
	specCache.Store(t, specs)
	return specs
}

func collectSpecs(t reflect.Type, prefix []int, seen map[reflect.Type]bool) []fieldSpec {
	if seen[t] {
		return nil
	}
	seen[t] = true
	defer delete(seen, t)
	var specs []fieldSpec
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		index := append(append([]int{}, prefix...), i)
		if f.Anonymous {
			ft := f.Type
			if ft.Kind() == reflect.Pointer {
				ft = ft.Elem()
			}
			if ft.Kind() == reflect.Struct && ft != timeType {
				specs = append(specs, collectSpecs(ft, index, seen)...)
				continue
			}
		}
		if !f.IsExported() {
			continue
		}
		// A body field's fields are keyed as the input's own, as an embedded
		// struct's are.
		if _, ok := f.Tag.Lookup("body"); ok && f.Type.Kind() == reflect.Struct && f.Type != timeType {
			specs = append(specs, collectSpecs(f.Type, index, seen)...)
			continue
		}
		key := fieldKey(f)
		if key == "" {
			continue
		}
		jsonKey, _, _ := strings.Cut(f.Tag.Get("json"), ",")
		spec := fieldSpec{key: key, index: index, json: jsonKey == key}
		if tag := f.Tag.Get("warp"); tag != "" {
			spec.rules, spec.readonly, spec.problems = parseRules(t, f, tag)
		}
		specs = append(specs, spec)
	}
	return specs
}

// fieldKey is the name in the json tag, else the query tag, else the path tag.
// json:"-" is no name, so a field a typed route fills from the query or the
// path is validated under that key.
func fieldKey(f reflect.StructField) string {
	for _, tag := range []string{"json", "query", "path"} {
		name, _, _ := strings.Cut(f.Tag.Get(tag), ",")
		if name != "" && name != "-" {
			return name
		}
	}
	return ""
}

// isReadonly reports whether f's warp tag has the rule readonly.
func isReadonly(f reflect.StructField) bool {
	for part := range strings.SplitSeq(f.Tag.Get("warp"), ",") {
		if strings.TrimSpace(part) == "readonly" {
			return true
		}
	}
	return false
}

// parseRules reads the rules of f's warp tag, and whether it has readonly,
// which no other rule may go beside: a readonly field is answered and never
// read, so it has no rules.
func parseRules(owner reflect.Type, f reflect.StructField, tag string) ([]rule, bool, []string) {
	at := owner.Name()
	if at == "" {
		at = owner.String()
	}
	at += "." + f.Name
	ft := f.Type
	for ft.Kind() == reflect.Pointer {
		ft = ft.Elem()
	}
	k := ft.Kind()
	isText := k == reflect.String
	isNumber := isNumberKind(k)
	isList := k == reflect.Slice || k == reflect.Array

	if isReadonly(f) {
		for part := range strings.SplitSeq(tag, ",") {
			if part = strings.TrimSpace(part); part != "" && part != "readonly" {
				return nil, true, []string{fmt.Sprintf("%s has the rule \"readonly\" beside other rules; a readonly field is never read, so remove them.", at)}
			}
		}
		return nil, true, nil
	}
	var rules []rule
	var problems []string
	for _, part := range strings.Split(tag, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		name, arg, _ := strings.Cut(part, "=")
		r := rule{name: name, arg: arg}
		switch name {
		case "required":
		case "email", "url", "enum":
			if !isText {
				problems = append(problems, fmt.Sprintf("%s has the rule %q, which is for text; remove it or make the field a string.", at, name))
				continue
			}
			if name == "enum" {
				for _, item := range strings.Split(arg, "|") {
					if item != "" {
						r.items = append(r.items, item)
					}
				}
				if len(r.items) == 0 {
					problems = append(problems, fmt.Sprintf("%s has \"enum=\" with nothing to choose from; write enum=a|b.", at))
					continue
				}
			}
		case "min", "max":
			if !isText && !isNumber && !isList {
				problems = append(problems, fmt.Sprintf("%s has the rule %q, which is for text, numbers and lists; remove it.", at, name))
				continue
			}
			var err error
			if isNumber {
				r.num, err = strconv.ParseFloat(arg, 64)
				if err == nil && (math.IsNaN(r.num) || math.IsInf(r.num, 0)) {
					err = strconv.ErrSyntax
				}
			} else {
				var n uint64
				n, err = strconv.ParseUint(arg, 10, 53)
				r.num = float64(n)
			}
			if err != nil {
				problems = append(problems, fmt.Sprintf("%s has %q; write a number after %s=.", at, part, name))
				continue
			}
		case "unique":
			if !isText && !isNumber {
				problems = append(problems, fmt.Sprintf("%s has the rule %q, which is for text and numbers; remove it.", at, name))
				continue
			}
		default:
			problems = append(problems, fmt.Sprintf("%s has the rule %q, which warp does not know; use %s.", at, name, ruleList))
			continue
		}
		rules = append(rules, r)
	}
	return rules, false, problems
}

// checkRules returns every mistake in the warp tags of t, a struct type, and of
// the struct types inside it, as sentences. A mistake is reported once.
func checkRules(t reflect.Type) []string {
	var out []string
	checkType(t, map[reflect.Type]bool{}, &out)
	return out
}

func checkType(t reflect.Type, visited map[reflect.Type]bool, out *[]string) {
	for isContainer(t.Kind()) {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct || t == timeType || visited[t] {
		return
	}
	visited[t] = true
	specs := specsOf(t)
	for _, spec := range specs {
		*out = append(*out, spec.problems...)
		checkType(fieldTypeByIndex(t, spec.index), visited, out)
	}
}

func fieldTypeByIndex(t reflect.Type, index []int) reflect.Type {
	for _, x := range index {
		for t.Kind() == reflect.Pointer {
			t = t.Elem()
		}
		t = t.Field(x).Type
	}
	return t
}
