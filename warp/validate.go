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
// rules and the mistakes found in them.
type fieldSpec struct {
	key      string
	index    []int
	rules    []rule
	problems []string
}

var (
	specCache sync.Map // reflect.Type -> []fieldSpec
	timeType  = reflect.TypeOf(time.Time{})
)

const ruleList = "required, email, url, min=, max=, enum= or unique"

// Validate checks v, a struct or a pointer to one, against the warp tags of
// its fields. It returns nil, or an *Error from Invalid that names each field
// that needs attention.
func Validate(v any) error {
	rv := derefValue(reflect.ValueOf(v))
	if !rv.IsValid() || rv.Kind() != reflect.Struct {
		return nil
	}
	fields := map[string]string{}
	var at []segment
	validateStruct(rv, &at, fields)
	if len(fields) == 0 {
		return nil
	}
	return Invalid(fields)
}

// segment is one step of a field's key: a field's or a map entry's name, or a
// list index when index is not negative. The walk keeps the steps to where it
// is as a stack, pushing one before it goes in and popping it after, and
// writes them out as a key only for a field that fails: a value that passes
// costs no string. (A chain of segments linked to their parents would not
// stay on the stack: Go moves a value whose address flows back into itself
// through recursion to the heap.)
type segment struct {
	name  string
	index int
}

// key writes the steps as a dotted key, such as lines.0.sku.
func key(at []segment) string {
	var digits [20]byte
	size := len(at) - 1
	for _, s := range at {
		if s.index >= 0 {
			size += len(strconv.AppendInt(digits[:0], int64(s.index), 10))
		} else {
			size += len(s.name)
		}
	}
	var b strings.Builder
	b.Grow(size)
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

func validateStruct(rv reflect.Value, at *[]segment, fields map[string]string) {
	depth := len(*at)
	for _, spec := range specsOf(rv.Type()) {
		fv, ok := fieldByIndex(rv, spec.index)
		if !ok {
			continue
		}
		*at = append((*at)[:depth], segment{name: spec.key, index: -1})
		if msg := checkField(fv, spec.rules); msg != "" {
			fields[key(*at)] = msg
		}
		if !holdsStructs(fv.Type()) {
			continue
		}
		if inner, ok := reachable(fv); ok {
			validateInside(inner, at, fields)
		}
	}
	*at = (*at)[:depth]
}

// validateInside validates the structs in v, a struct, or a slice, array or
// map whose element type can hold them, as reachable gives it. A list's
// entries are keyed by their index after at, a map's by its keys, formatted
// with fmt.Sprint and visited in sorted order. at is the stack of steps to v;
// it is as long again when validateInside returns.
func validateInside(v reflect.Value, at *[]segment, fields map[string]string) {
	depth := len(*at)
	defer func() { *at = (*at)[:depth] }()
	switch v.Kind() {
	case reflect.Struct:
		validateStruct(v, at, fields)
	case reflect.Slice, reflect.Array:
		if !holdsStructs(v.Type().Elem()) {
			return
		}
		for i := 0; i < v.Len(); i++ {
			if inner, ok := reachable(v.Index(i)); ok {
				*at = append((*at)[:depth], segment{index: i})
				validateInside(inner, at, fields)
			}
		}
	case reflect.Map:
		if !holdsStructs(v.Type().Elem()) {
			return
		}
		type entry struct {
			name  string
			value reflect.Value
		}
		var entries []entry
		probe := reflect.New(v.Type().Elem()).Elem()
		for iter := v.MapRange(); iter.Next(); {
			probe.SetIterValue(iter) // reused, so an entry with nothing in it costs nothing
			if _, ok := reachable(probe); ok {
				inner, _ := reachable(iter.Value())
				entries = append(entries, entry{fmt.Sprint(iter.Key().Interface()), inner})
			}
		}
		slices.SortFunc(entries, func(a, b entry) int { return strings.Compare(a.name, b.name) })
		for _, e := range entries {
			*at = append((*at)[:depth], segment{name: e.name, index: -1})
			validateInside(e.value, at, fields)
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
func checkField(fv reflect.Value, rules []rule) string {
	v := derefValue(fv)
	for _, r := range rules {
		if r.name == "required" {
			// A pointer says "present, and it may be zero".
			if fv.Kind() == reflect.Pointer && !fv.IsNil() {
				continue
			}
			if !v.IsValid() || isBlank(v) {
				return "This is required."
			}
			continue
		}
		// Every other rule lets an empty value pass.
		if !v.IsValid() || isEmpty(v) {
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
		key := fieldKey(f)
		if key == "" {
			continue
		}
		spec := fieldSpec{key: key, index: index}
		if tag := f.Tag.Get("warp"); tag != "" {
			spec.rules, spec.problems = parseRules(t, f, tag)
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

func parseRules(owner reflect.Type, f reflect.StructField, tag string) ([]rule, []string) {
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
	return rules, problems
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
