package data

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"reflect"
	"time"
)

// First is the record whose field, named as in JSON, equals value: the one
// with the lowest id when several do. ok is false when none does.
//
// value is compared as the store writes the field: a pointer is what it
// points to, a named type is its kind, and a time is the store's text for it
// (to the nanosecond, in UTC). Text is compared exactly, case included, and a
// number as a number, so 40 finds a price of 40.0. nil and a nil pointer,
// which no value equals, and a value of a kind the column does not hold (a
// struct, a list, text for a number) are a plain error naming the field.
func (s *Store[T]) First(ctx context.Context, field string, value any) (v T, ok bool, err error) {
	q, d, err := s.ready(ctx)
	if err != nil {
		return v, false, err
	}
	c := s.t.col(field)
	if c == nil {
		return v, false, fmt.Errorf("data: %s has no field %q to look up", s.name, field)
	}
	arg, err := s.t.lookupValue(d, c, value)
	if err != nil {
		return v, false, err
	}
	err = s.t.scan(d, q.QueryRowContext(ctx, s.t.firstSQL(d, c), arg), reflect.ValueOf(&v).Elem())
	switch {
	case errors.Is(err, sql.ErrNoRows):
		var zero T
		return zero, false, nil
	case err != nil:
		var zero T
		return zero, false, unavailable(ctx, d, err)
	}
	return v, true, nil
}

// lookupValue is value as column c is written, for First.
func (t *table) lookupValue(d dialect, c *column, value any) (any, error) {
	rv := reflect.ValueOf(value)
	for rv.IsValid() && rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			rv = reflect.Value{}
			break
		}
		rv = rv.Elem()
	}
	if !rv.IsValid() {
		return nil, fmt.Errorf("data: %s cannot look up %q by nil, which no value equals; give it a value.", t.name, c.name)
	}
	wrong := func(want string) error {
		return fmt.Errorf("data: %s cannot look up %q by a value of type %s; give it %s.", t.name, c.name, rv.Type(), want)
	}
	k := rv.Kind()
	switch c.kind {
	case kindText:
		if k == reflect.String {
			return rv.String(), nil
		}
		return nil, wrong("text")
	case kindBool:
		if k == reflect.Bool {
			return rv.Bool(), nil
		}
		return nil, wrong("true or false")
	case kindInteger, kindReal:
		switch {
		case rv.CanInt():
			return rv.Int(), nil
		case rv.CanUint():
			if rv.Uint() > math.MaxInt64 {
				return nil, fmt.Errorf("data: %s cannot look up %q by %d, more than the column keeps; give it a smaller number.", t.name, c.name, rv.Uint())
			}
			return int64(rv.Uint()), nil
		case rv.CanFloat() && c.kind == kindReal:
			return rv.Float(), nil
		case c.kind == kindReal:
			return nil, wrong("a number")
		}
		return nil, wrong("a whole number")
	case kindTime:
		if rv.Type().ConvertibleTo(timeType) && k == reflect.Struct {
			return d.timeValue(rv.Convert(timeType).Interface().(time.Time)), nil
		}
		return nil, wrong("a time.Time")
	case kindBlob:
		if k == reflect.Slice && rv.Type().Elem().Kind() == reflect.Uint8 {
			return rv.Bytes(), nil
		}
		return nil, wrong("bytes")
	}
	return nil, wrong("a value the column holds")
}

func (t *table) firstSQL(d dialect, c *column) string {
	return "SELECT " + t.names(d) + " FROM " + d.quote(t.name) + " WHERE " + d.quote(c.name) + " = " + d.placeholder(1) +
		" ORDER BY " + d.quote("id") + " LIMIT 1"
}
