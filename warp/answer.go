package warp

import (
	"fmt"
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
// and into copies it made itself. What it reaches through a pointer, a slice
// or a map may be shared (a cached record, a package-level default, a value
// another request is answering), so there it makes a copy, and only of what
// changes. What the handler returned is never changed. An empty list or map
// it writes is one the plan holds, shared by every answer, which only the
// encoder reads.
//
// It looks inside embedded structs, exported or not, as encoding/json does.
// It does not look inside an interface (an any field), whose type is not
// known at registration, nor inside a type that writes itself (a
// json.Marshaler or encoding.TextMarshaler). A field encoding/json leaves
// out, omitempty on an empty list or omitzero on a zero value, is left as
// the handler returned it, so it is still left out. A value that holds
// itself through pointers, lists or maps is found as encoding/json finds it,
// by the path it is on once past 1000 of them; the fix-up then gives up, and
// the encoder answers its cycle error as it would have.
type emptyPlan struct {
	kind   reflect.Kind
	typ    reflect.Type
	empty  reflect.Value // a slice's or map's empty value, written for nil; invalid for other kinds
	elem   *emptyPlan    // what a pointer points to, or the elements or values; nil when they need nothing
	fields []emptyField  // a struct's fields that need something
}

type emptyField struct {
	index     int
	plan      *emptyPlan
	embedded  bool // an unexported embedded struct: changed in place on a copy, since it cannot be set whole
	omitEmpty bool
	omitZero  func(v reflect.Value) bool // omitzero: whether encoding/json leaves the value out; nil without it
}

// startDetectingCycles is how many pointers, lists and maps deep the
// fix-up goes before it watches for a value it is already inside, as
// encoding/json's encoder does (startDetectingCyclesAfter).
const startDetectingCycles = 1000

// emptyPlanFor returns the plan for an answer of type t, or nil when nothing
// in t is ever written as a null list, map or bytes, and the mistakes of a
// type it cannot fix: an unexported embedded pointer to a struct that holds
// a list, which cannot be replaced by a copy.
func emptyPlanFor(t reflect.Type) (*emptyPlan, []string) {
	b := emptyBuilder{plans: map[reflect.Type]*emptyPlan{}}
	top := b.plan(t)
	// A plan is needed when it makes something empty or holds a plan that is
	// needed. A type can hold itself, so this marks until nothing changes.
	needed := map[*emptyPlan]bool{}
	needs := func(p *emptyPlan) bool {
		if p.empty.IsValid() || p.elem != nil && needed[p.elem] {
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
	var mistakes []string
	for _, bl := range b.blocked {
		if needed[bl.plan] {
			mistakes = append(mistakes, fmt.Sprintf("%s embeds *%s, a pointer to an unexported struct with lists in it, which Warp cannot write as [] when they are nil; embed %s by value.",
				typeName(bl.in), typeName(bl.typ), typeName(bl.typ)))
		}
	}
	for _, p := range all {
		if !needed[p.elem] {
			p.elem = nil
		}
		p.fields = slices.DeleteFunc(p.fields, func(f emptyField) bool { return !needed[f.plan] })
	}
	if !needed[top] || top.kind == reflect.Slice && top.elem == nil && !isBytes(t) {
		return nil, mistakes // WriteJSON writes a nil list at the top as [] itself
	}
	return top, mistakes
}

// emptyBuilder makes the plans of the types an answer reaches. A struct
// type's plan is made once, so a type that holds itself ends; the plans of
// pointers, lists and maps are made where they are met, and kept in extra.
type emptyBuilder struct {
	plans   map[reflect.Type]*emptyPlan
	extra   []*emptyPlan
	blocked []blockedEmbed
}

// blockedEmbed is an unexported embedded pointer to a struct, in the struct
// in, whose plan, if needed, cannot be carried out.
type blockedEmbed struct {
	in, typ reflect.Type
	plan    *emptyPlan
}

func (b *emptyBuilder) plan(t reflect.Type) *emptyPlan {
	if selfEncoded(t) != "" {
		return nil
	}
	switch t.Kind() {
	case reflect.Pointer, reflect.Slice, reflect.Array, reflect.Map:
		p := &emptyPlan{kind: t.Kind(), typ: t}
		switch t.Kind() {
		case reflect.Slice:
			p.empty = reflect.MakeSlice(t, 0, 0)
		case reflect.Map:
			p.empty = reflect.MakeMapWithSize(t, 0)
		}
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
			key, opts, tagged := jsonTagOf(f)
			if !tagged && f.Tag.Get("json") == "-" {
				continue // not written
			}
			// encoding/json flattens an embedded struct with no json name into
			// this one, exported or not; with a name it is a field like any
			// other, and an unexported one is not written.
			flat := f.Anonymous && key == ""
			embedded := flat && !f.IsExported()
			if !f.IsExported() && !flat {
				continue
			}
			if embedded && f.Type.Kind() == reflect.Pointer {
				if f.Type.Elem().Kind() == reflect.Struct {
					if fp := b.plan(f.Type.Elem()); fp != nil {
						b.blocked = append(b.blocked, blockedEmbed{t, f.Type.Elem(), fp})
					}
				}
				continue
			}
			if embedded && f.Type.Kind() != reflect.Struct {
				continue // an unexported embedded non-struct is not written
			}
			if fp := b.plan(f.Type); fp != nil {
				ef := emptyField{index: i, plan: fp, embedded: embedded}
				if !flat { // a flattened struct's options are not encoding/json's to read
					ef.omitEmpty = hasJSONOption(opts, "omitempty")
					if hasJSONOption(opts, "omitzero") {
						ef.omitZero = zeroTest(f.Type)
					}
				}
				p.fields = append(p.fields, ef)
			}
		}
		return p
	}
	return nil
}

type isZeroer interface{ IsZero() bool }

var isZeroerType = reflect.TypeFor[isZeroer]()

// zeroTest is how encoding/json tells that a field of type t tagged omitzero
// is zero: its IsZero method, or reflect's.
func zeroTest(t reflect.Type) func(reflect.Value) bool {
	switch {
	case t.Kind() == reflect.Interface && t.Implements(isZeroerType):
		return func(v reflect.Value) bool {
			return v.IsNil() || v.Elem().Kind() == reflect.Pointer && v.Elem().IsNil() || v.Interface().(isZeroer).IsZero()
		}
	case t.Kind() == reflect.Pointer && t.Implements(isZeroerType):
		return func(v reflect.Value) bool { return v.IsNil() || v.Interface().(isZeroer).IsZero() }
	case t.Implements(isZeroerType):
		return func(v reflect.Value) bool { return v.Interface().(isZeroer).IsZero() }
	case reflect.PointerTo(t).Implements(isZeroerType):
		return func(v reflect.Value) bool {
			if !v.CanAddr() {
				v2 := reflect.New(v.Type()).Elem()
				v2.Set(v)
				v = v2
			}
			return v.Addr().Interface().(isZeroer).IsZero()
		}
	}
	return reflect.Value.IsZero
}

// omitted reports whether encoding/json leaves the field out with the value
// v, as the handler returned it.
func (f *emptyField) omitted(v reflect.Value) bool {
	if f.omitZero != nil && f.omitZero(v) {
		return true
	}
	k := v.Kind()
	return f.omitEmpty && (k == reflect.Slice || k == reflect.Map) && v.Len() == 0
}

// emptyWalk is one fix-up of an answer: how deep it is in pointers, lists
// and maps, what it is inside of once past startDetectingCycles, and
// whether it gave up on a value that holds itself.
type emptyWalk struct {
	depth     int
	inside    map[any]struct{}
	abandoned bool
}

// emptied returns v with its null lists, maps and bytes made empty, or v as
// it was when it holds itself. It is a function of its own so that a route
// whose answer has no plan never takes its answer's address, which would
// move it to the heap.
func emptied[T any](p *emptyPlan, v T) T {
	rv := reflect.ValueOf(&v).Elem()
	var w emptyWalk
	out, changed := p.apply(rv, true, &w)
	if w.abandoned {
		return v // what is left is the encoder's to refuse
	}
	if changed {
		rv.Set(out)
	}
	return v
}

// enter notes that the walk goes into the pointer, list or map v, and
// reports false, giving the walk up, when it is already inside it.
func (w *emptyWalk) enter(v reflect.Value) (key any, ok bool) {
	if w.depth++; w.depth <= startDetectingCycles {
		return nil, true
	}
	switch v.Kind() {
	case reflect.Slice:
		key = struct {
			ptr uintptr
			len int
		}{v.Pointer(), v.Len()}
	default:
		key = v.Pointer()
	}
	if w.inside == nil {
		w.inside = map[any]struct{}{}
	}
	if _, in := w.inside[key]; in {
		w.abandoned = true
		return nil, false
	}
	w.inside[key] = struct{}{}
	return key, true
}

func (w *emptyWalk) leave(key any) {
	w.depth--
	if key != nil {
		delete(w.inside, key)
	}
}

// apply returns v with every nil list, map and bytes it holds made empty, as
// the plan says. When own is true, v is the answer's own or a copy the
// fix-up made (addressable, and seen by no one else) and is changed in
// place; otherwise what changes is copied. changed reports whether anything
// did. When the walk is abandoned, what it returns is not to be used.
func (p *emptyPlan) apply(v reflect.Value, own bool, w *emptyWalk) (_ reflect.Value, changed bool) {
	switch p.kind {
	case reflect.Pointer:
		if v.IsNil() || p.elem == nil {
			return v, false
		}
		key, ok := w.enter(v)
		if !ok {
			return v, false
		}
		defer w.leave(key)
		inner, changed := p.elem.apply(v.Elem(), false, w)
		if !changed || w.abandoned {
			return v, false
		}
		if inner.CanAddr() && (p.elem.kind == reflect.Struct || p.elem.kind == reflect.Array) {
			return inner.Addr(), true // the copy apply made
		}
		ptr := reflect.New(p.typ.Elem())
		ptr.Elem().Set(inner)
		return ptr, true
	case reflect.Slice, reflect.Map:
		if v.IsNil() {
			return p.empty, true
		}
		if p.elem == nil || v.Len() == 0 {
			return v, false
		}
		key, ok := w.enter(v)
		if !ok {
			return v, false
		}
		defer w.leave(key)
		if p.kind == reflect.Map {
			return p.applyMap(v, w)
		}
		var out reflect.Value // a copy, made at the first element that changes
		for i := range v.Len() {
			if out.IsValid() {
				// The copy is the fix-up's own: its elements change in place.
				if e, changed := p.elem.apply(out.Index(i), true, w); changed && !w.abandoned {
					out.Index(i).Set(e)
				}
			} else if e, changed := p.elem.apply(v.Index(i), false, w); changed && !w.abandoned {
				out = reflect.MakeSlice(p.typ, v.Len(), v.Len())
				reflect.Copy(out, v)
				out.Index(i).Set(e)
			}
			if w.abandoned {
				return v, false
			}
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
			e, changed := p.elem.apply(out.Index(i), copied, w)
			if w.abandoned {
				return v, false
			}
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
		for i := range p.fields {
			f := &p.fields[i]
			if f.omitted(v.Field(f.index)) {
				continue // encoding/json leaves it out: so it stays
			}
			if f.embedded && !copied {
				// An unexported embedded struct cannot be set whole, nor copied
				// out, but its exported fields can be set: so it is changed in
				// place, on a copy of this struct.
				out = reflect.New(p.typ).Elem()
				out.Set(v)
				copied = true
			}
			fv := out.Field(f.index)
			if copied && (f.plan.kind == reflect.Struct || f.plan.kind == reflect.Array) {
				_, changed := f.plan.apply(fv, true, w) // in place: fv is the fix-up's own
				if w.abandoned {
					return v, false
				}
				some = some || changed
				continue
			}
			e, changed := f.plan.apply(fv, false, w)
			if w.abandoned {
				return v, false
			}
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
		if !some {
			return v, false
		}
		return out, true
	}
	return v, false
}

// applyMap returns the map v with its values fixed, copied when one changes.
// Looking costs one allocation, the value each entry is read into, which is
// the fix-up's own, so it changes in place.
func (p *emptyPlan) applyMap(v reflect.Value, w *emptyWalk) (reflect.Value, bool) {
	var out reflect.Value
	var iter reflect.MapIter
	iter.Reset(v)
	value := reflect.New(p.typ.Elem()).Elem()
	for iter.Next() {
		value.SetIterValue(&iter)
		e, changed := p.elem.apply(value, true, w)
		if w.abandoned {
			return v, false
		}
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
