package warp

import (
	"maps"
	"reflect"
	"slices"
)

// zeroPlan puts back to zero every field tagged readonly that a decoded
// body's value holds, at any depth: through pointers, lists, arrays, map
// values and embedded structs, exported or not. It is what makes "a readonly
// field is never read" hold whatever the decoder chose, even where the scan
// and the decoder disagree. It does not look inside a type that decodes
// itself, and need not: registration refuses a readonly field there
// (readonlyInside), since what such a decoder sets cannot be seen. It is made once per plan, when the route is
// registered; a type that holds no readonly field has no plan (nil), and
// costs nothing.
type zeroPlan struct {
	kind   reflect.Kind
	typ    reflect.Type
	elem   *zeroPlan // what a pointer points to, the elements or the values
	fields []zeroField
}

type zeroField struct {
	index    int
	readonly bool      // put back to zero
	plan     *zeroPlan // else what is inside it
}

// readonlyPlanFor returns the plan for a decoded value of type t, or nil
// when t holds no readonly field.
func readonlyPlanFor(t reflect.Type) *zeroPlan {
	b := zeroBuilder{plans: map[reflect.Type]*zeroPlan{}}
	top := b.plan(t)
	// A plan is needed when it has a readonly field or holds a plan that is
	// needed. A type can hold itself, so this marks until nothing changes.
	needed := map[*zeroPlan]bool{}
	all := append(slices.Collect(maps.Values(b.plans)), b.extra...)
	for changed := true; changed; {
		changed = false
		for _, p := range all {
			if needed[p] {
				continue
			}
			need := p.elem != nil && needed[p.elem]
			for _, f := range p.fields {
				need = need || f.readonly || needed[f.plan]
			}
			if need {
				needed[p], changed = true, true
			}
		}
	}
	for _, p := range all {
		if !needed[p.elem] {
			p.elem = nil
		}
		p.fields = slices.DeleteFunc(p.fields, func(f zeroField) bool { return !f.readonly && !needed[f.plan] })
	}
	if !needed[top] {
		return nil
	}
	return top
}

type zeroBuilder struct {
	plans map[reflect.Type]*zeroPlan // of struct types, made once
	extra []*zeroPlan
}

func (b *zeroBuilder) plan(t reflect.Type) *zeroPlan {
	if t == timeType || decodesItself(t) {
		return nil
	}
	switch t.Kind() {
	case reflect.Pointer, reflect.Slice, reflect.Array, reflect.Map:
		p := &zeroPlan{kind: t.Kind(), typ: t}
		b.extra = append(b.extra, p)
		p.elem = b.plan(t.Elem())
		return p
	case reflect.Struct:
		if p, ok := b.plans[t]; ok {
			return p
		}
		p := &zeroPlan{kind: reflect.Struct, typ: t}
		b.plans[t] = p
		for i := range t.NumField() {
			f := t.Field(i)
			// What encoding/json can set: an exported field, or an embedded
			// struct by value, whose exported fields reflect can set too.
			embedded := f.Anonymous && f.Type.Kind() == reflect.Struct
			if f.Tag.Get("json") == "-" || !f.IsExported() && !embedded {
				continue
			}
			if !embedded && isReadonly(f) {
				p.fields = append(p.fields, zeroField{index: i, readonly: true})
				continue
			}
			if fp := b.plan(f.Type); fp != nil {
				p.fields = append(p.fields, zeroField{index: i, plan: fp})
			}
		}
		return p
	}
	return nil
}

// zero puts back to zero, in place, the readonly fields v holds, and
// reports whether one was set. v is a value the decoder has just made, which
// no one else holds. A map's value is written back only when a field of it
// was set, so a map that sent none costs one value to read entries into.
func (p *zeroPlan) zero(v reflect.Value) (changed bool) {
	switch p.kind {
	case reflect.Pointer:
		if !v.IsNil() && p.elem != nil {
			return p.elem.zero(v.Elem())
		}
	case reflect.Slice, reflect.Array:
		if p.elem != nil {
			for i := range v.Len() {
				changed = p.elem.zero(v.Index(i)) || changed
			}
		}
	case reflect.Map:
		if p.elem == nil || v.Len() == 0 {
			return false
		}
		value := reflect.New(p.typ.Elem()).Elem() // a map value cannot be set in place
		iter := v.MapRange()
		for iter.Next() {
			value.SetIterValue(iter)
			if p.elem.zero(value) {
				v.SetMapIndex(iter.Key(), value)
				changed = true
			}
		}
	case reflect.Struct:
		for _, f := range p.fields {
			fv := v.Field(f.index)
			if f.readonly {
				if !fv.IsZero() {
					fv.SetZero()
					changed = true
				}
			} else {
				changed = f.plan.zero(fv) || changed
			}
		}
	}
	return changed
}
