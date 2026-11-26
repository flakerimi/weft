package data

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"time"
)

// First is the record whose field, named as in JSON, equals value: the one
// with the lowest id when several do. ok is false when none does.
func (s *Store[T]) First(ctx context.Context, field string, value any) (v T, ok bool, err error) {
	q, d, err := s.ready(ctx)
	if err != nil {
		return v, false, err
	}
	c := s.t.col(field)
	if c == nil {
		return v, false, fmt.Errorf("data: %s has no field %q to look up", s.name, field)
	}
	if at, isTime := value.(time.Time); isTime { // compared as the store writes it
		value = d.timeValue(at)
	}
	err = s.t.scan(d, q.QueryRowContext(ctx, s.t.firstSQL(d, c), value), reflect.ValueOf(&v).Elem())
	switch {
	case errors.Is(err, sql.ErrNoRows):
		var zero T
		return zero, false, nil
	case err != nil:
		var zero T
		return zero, false, err
	}
	return v, true, nil
}

func (t *table) firstSQL(d dialect, c *column) string {
	return "SELECT " + t.names(d) + " FROM " + d.quote(t.name) + " WHERE " + d.quote(c.name) + " = " + d.placeholder(1) +
		" ORDER BY " + d.quote("id") + " LIMIT 1"
}
