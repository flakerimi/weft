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
	validateStruct(rv, "", fields)
	if len(fields) == 0 {
		return nil
	}
	return Invalid(fields)
}

func validateStruct(rv reflect.Value, prefix string, fields map[string]string) {
	for _, spec := range specsOf(rv.Type()) {
		fv, ok := fieldByIndex(rv, spec.index)
		if !ok {
			continue
		}
		key := spec.key
		if prefix != "" {
			key = prefix + "." + key
		}
		if msg := checkField(fv, spec.rules); msg != "" {
			fields[key] = msg
		}
		if holdsStructs(fv.Type()) {
			validateInside(fv, key, fields)
		}
	}
}

// validateInside validates the structs in v, a struct, or a pointer, slice,
// array or map that holds them. A list's entries are keyed by their index
// after key, a map's by its keys, formatted with fmt.Sprint and visited in
// sorted order.
func validateInside(v reflect.Value, key string, fields map[string]string) {
	v = derefValue(v)
	switch {
	case isDescendable(v):
		validateStruct(v, key, fields)
	case !v.IsValid():
	case v.Kind() == reflect.Slice || v.Kind() == reflect.Array:
		for i := 0; i < v.Len(); i++ {
			validateInside(v.Index(i), key+"."+strconv.Itoa(i), fields)
		}
	case v.Kind() == reflect.Map:
		keys := v.MapKeys()
		names := make(map[reflect.Value]string, len(keys))
		for _, k := range keys {
			names[k] = fmt.Sprint(k.Interface())
		}
		slices.SortFunc(keys, func(a, b reflect.Value) int { return strings.Compare(names[a], names[b]) })
		for _, k := range keys {
			validateInside(v.MapIndex(k), key+"."+names[k], fields)
		}
	}
}

// holdsStructs reports whether a value of type t may be, or hold through
// pointers, slices, arrays and map values, a struct to validate. An interface
// may hold one.
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
