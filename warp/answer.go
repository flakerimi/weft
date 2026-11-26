package warp

import (
	"maps"
	"reflect"
	"slices"
)

// emptyPlan is how a typed route's answer is cleared of null lists, maps and
// bytes before it is written: encoding/json writes a nil slice or map as
// null, and the description (and the TypeScript generated from it) types
// such a field T[] or Record<string, T>, never null. It is made once per
// output type, when the route is registered; a type that holds no such
// value, at any depth, has no plan (nil), and its answer is written as it is.
//
// The fix-up writes in place only into the answer's own value, the variable
// the handler returned and the structs and arrays held inside it by value,
// which no one else can see. What it reaches through a pointer, a slice or
// a map may be shared (a cached record, a package-level default, a value
// another request is answering), so there it makes a copy, and only of what
// changes. What the handler returned is never changed.
//
// It does not look inside an interface (an any field), whose type is not
// known at registration, nor inside a type that writes itself (a
// json.Marshaler or encoding.TextMarshaler), nor into the fields of an
// unexported embedded struct, which cannot be set. A field tagged omitempty
// or omitzero is left nil, since encoding/json leaves it out then.
type emptyPlan struct {
	kind   reflect.Kind
	typ    reflect.Type
	empty  bool         // a slice or map, which is made empty when nil
	elem   *emptyPlan   // what a pointer points to, or the elements or values; nil when they need nothing
	fields []emptyField // a struct's fields that need something
}

type emptyField struct {
	index int
	keep  bool // omitempty or omitzero: left nil, looked inside when not
	plan  *emptyPlan
}

// maxEmptyDepth is how many pointers, lists and maps deep the fix-up goes.
// Past it a value is written as it is: a value that holds itself through a
// pointer never ends, and encoding/json then answers its cycle error.
const maxEmptyDepth = 1000

// emptyPlanFor returns the plan for an answer of type t, or nil when nothing
// in t is ever written as a null list, map or bytes.
func emptyPlanFor(t reflect.Type) *emptyPlan {
	b := emptyBuilder{plans: map[reflect.Type]*emptyPlan{}}
	top := b.plan(t)
	// A plan is needed when it makes something empty or holds a plan that is
	// needed. A type can hold itself, so this marks until nothing changes.
	needed := map[*emptyPlan]bool{}
	var needs func(p *emptyPlan) bool
	needs = func(p *emptyPlan) bool {
		if p == nil {
			return false
		}
		if p.empty || needed[p] {
			return true
		}
		if p.elem != nil && needed[p.elem] {
			return true
		}
		for _, f := range p.fields {
			if needed[f.plan] {
				return true
			}
		}
		return false
	}
	all := append(slices.Collect(maps.Values(b.plans)), b.extra...)
	for changed := true; changed; {
		changed = false
		for _, p := range all {
			if !needed[p] && needs(p) {
				needed[p], changed = true, true
			}
		}
	}
	for _, p := range all {
		if !needed[p.elem] {
			p.elem = nil
		}
		p.fields = slices.DeleteFunc(p.fields, func(f emptyField) bool { return !needed[f.plan] })
	}
	if !needed[top] || top.kind == reflect.Slice && top.elem == nil && !isBytes(t) {
		return nil // WriteJSON writes a nil list at the top as [] itself
	}
	return top
}

// emptyBuilder makes the plans of the types an answer reaches. A struct
// type's plan is made once, so a type that holds itself ends; the plans of
// pointers, lists and maps are made where they are met, and kept in extra.
type emptyBuilder struct {
	plans map[reflect.Type]*emptyPlan
	extra []*emptyPlan
}

func (b *emptyBuilder) plan(t reflect.Type) *emptyPlan {
	if selfEncoded(t) != "" {
		return nil
	}
	switch t.Kind() {
	case reflect.Pointer, reflect.Slice, reflect.Array, reflect.Map:
		p := &emptyPlan{kind: t.Kind(), typ: t, empty: t.Kind() == reflect.Slice || t.Kind() == reflect.Map}
		b.extra = append(b.extra, p)
		if !isBytes(t) {
			p.elem = b.plan(t.Elem())
		}
		return p
	case reflect.Struct:
		if p, ok := b.plans[t]; ok {
			return p
		}
		p := &emptyPlan{kind: reflect.Struct, typ: t}
		b.plans[t] = p
		for i := range t.NumField() {
			f := t.Field(i)
			tag := f.Tag.Get("json")
			if tag == "-" || !f.IsExported() {
				continue // not written, or an unexported embedded struct, which cannot be set
			}
			_, opts, _ := jsonTagOf(f)
			if fp := b.plan(f.Type); fp != nil {
				p.fields = append(p.fields, emptyField{
					index: i, keep: hasJSONOption(opts, "omitempty") || hasJSONOption(opts, "omitzero"), plan: fp,
				})
			}
		}
		return p
	}
	return nil
}

// emptied returns v with its null lists, maps and bytes made empty. It is a
// function of its own so that a route whose answer has no plan never takes
// its answer's address, which would move it to the heap.
func emptied[T any](p *emptyPlan, v T) T {
	rv := reflect.ValueOf(&v).Elem()
	if out, changed := p.apply(rv, true, 0); changed {
		rv.Set(out)
	}
	return v
}

// apply returns v with every nil list, map and bytes it holds made empty, as
// the plan says. When own is true, v is the answer's own (addressable, and
// seen by no one else) and is changed in place; otherwise what changes is
// copied. changed reports whether anything did.
func (p *emptyPlan) apply(v reflect.Value, own bool, depth int) (_ reflect.Value, changed bool) {
	if depth > maxEmptyDepth {
		return v, false
	}
	switch p.kind {
	case reflect.Pointer:
		if v.IsNil() || p.elem == nil {
			return v, false
		}
		inner, changed := p.elem.apply(v.Elem(), false, depth+1)
		if !changed {
			return v, false
		}
		ptr := reflect.New(p.typ.Elem())
		ptr.Elem().Set(inner)
		return ptr, true
	case reflect.Slice, reflect.Map:
		if v.IsNil() {
			if p.kind == reflect.Slice {
				return reflect.MakeSlice(p.typ, 0, 0), true
			}
			return reflect.MakeMapWithSize(p.typ, 0), true
		}
		if p.elem == nil {
			return v, false
		}
		if p.kind == reflect.Map {
			return p.applyMap(v, depth)
		}
		var out reflect.Value // a copy, made at the first element that changes
		for i := range v.Len() {
			e, changed := p.elem.apply(v.Index(i), false, depth+1)
			if !changed {
				continue
			}
			if !out.IsValid() {
				out = reflect.MakeSlice(p.typ, v.Len(), v.Len())
				reflect.Copy(out, v)
			}
			out.Index(i).Set(e)
		}
		if !out.IsValid() {
			return v, false
		}
		return out, true
	case reflect.Array:
		if p.elem == nil {
			return v, false
		}
		out, copied, some := v, own, false
		for i := range v.Len() {
			e, changed := p.elem.apply(out.Index(i), copied, depth)
			if !changed {
				continue
			}
			if !copied {
				out = reflect.New(p.typ).Elem()
				out.Set(v)
				copied = true
			}
			out.Index(i).Set(e)
			some = true
		}
		return out, some
	case reflect.Struct:
		out, copied, some := v, own, false
		for _, f := range p.fields {
			fv := out.Field(f.index)
			if f.keep && (f.plan.kind == reflect.Slice || f.plan.kind == reflect.Map) && fv.IsNil() {
				continue
			}
			e, changed := f.plan.apply(fv, copied, depth)
			if !changed {
				continue
			}
			if !copied {
				out = reflect.New(p.typ).Elem()
				out.Set(v)
				copied = true
			}
			out.Field(f.index).Set(e)
			some = true
		}
		return out, some
	}
	return v, false
}

// applyMap returns the map v with its values fixed, copied when one changes.
// Looking costs one allocation, the value each entry is read into.
func (p *emptyPlan) applyMap(v reflect.Value, depth int) (reflect.Value, bool) {
	var out reflect.Value
	var iter reflect.MapIter
	iter.Reset(v)
	value := reflect.New(p.typ.Elem()).Elem()
	for iter.Next() {
		value.SetIterValue(&iter)
		e, changed := p.elem.apply(value, false, depth+1)
		if !changed {
			continue
		}
		if !out.IsValid() {
			out = copyMap(v)
		}
		out.SetMapIndex(iter.Key(), e)
	}
	if !out.IsValid() {
		return v, false
	}
	return out, true
}

// copyMap returns a new map with v's entries.
func copyMap(v reflect.Value) reflect.Value {
	out := reflect.MakeMapWithSize(v.Type(), v.Len())
	iter := v.MapRange()
	for iter.Next() {
		out.SetMapIndex(iter.Key(), iter.Value())
	}
	return out
}
