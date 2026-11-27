package data

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"time"

	"github.com/flakerimi/weft/warp"
)

// Record is what every model embeds, by value: its id and the times it was
// made and last changed, which the store sets. All three are answered and
// never read from a request.
type Record struct {
	ID        int64     `json:"id" warp:"readonly"`
	CreatedAt time.Time `json:"created_at" warp:"readonly"`
	UpdatedAt time.Time `json:"updated_at" warp:"readonly"`
}

// ID is the input of a route that names one record by its path, such as
// GET /products/{id}.
type ID struct {
	ID int64 `path:"id"`
}

// Store keeps the records of the model T in one table. Its columns are the
// json names of T's fields, an embedded struct's included; a field tagged
// json:"-" is not kept. Text, numbers, bools, times and bytes can be kept, a
// pointer to one as a column that may be NULL. A field with the rule unique
// gets a unique index, and a value another record has is answered 422 on that
// field. A call the database refuses because another connection holds it past
// the busy timeout is a warp.Unavailable, answered 503.
type Store[T any] struct {
	db   *DB
	name string
	t    *table // nil when T cannot be kept
}

// NewStore returns the store of T in the table named table, which it records
// on db for the data module to make at start; nothing is opened. T embeds
// Record by value. A mistake in T or in the name is reported by the app.
// Call it from a module's Register and keep the store.
func NewStore[T any](db *DB, table string) *Store[T] {
	s := &Store[T]{db: db, name: table}
	if db == nil { // the app reports that the data module is missing
		return s
	}
	t, mistakes := tableOf(reflect.TypeFor[T](), table)
	ok := len(mistakes) == 0
	if !tableName.MatchString(table) {
		db.app.Mistake("a store's table is named %q; use lower-case letters, digits and _, such as %s.", table, suggestTable(table))
		ok = false
	}
	for _, m := range mistakes {
		db.app.Mistake("%s", m)
	}
	if ok && db.add(t) {
		s.t = t
	}
	return s
}

// ready returns what a call runs on: the transaction on ctx or the database.
func (s *Store[T]) ready(ctx context.Context) (querier, dialect, error) {
	if s.db == nil || s.t == nil {
		return nil, nil, fmt.Errorf("warp: the %s store cannot be used, since the app reported a mistake when it was made; fix that first.", s.name)
	}
	conn, d := s.db.handle()
	if conn == nil {
		return nil, nil, fmt.Errorf("warp: the %s store was used before the app started; use it from a handler, or after Start.", s.name)
	}
	if s.db.stopped() {
		return nil, nil, errStopping
	}
	q, err := s.db.q(ctx, conn)
	return q, d, err
}

// notFound is the answer for a record that does not exist.
func (s *Store[T]) notFound(id int64) error {
	return warp.NotFound(fmt.Sprintf("No %s has id %d.", s.t.singular, id))
}

// taken turns a unique violation into a 422 on the field, and a busy
// database into the 503; any other error is returned as it is.
func (s *Store[T]) taken(ctx context.Context, d dialect, err error) error {
	if name, ok := d.uniqueColumn(err, s.name); ok {
		if c := s.t.col(name); c != nil {
			return warp.Invalid(map[string]string{c.name: "This is already taken."})
		}
	}
	return unavailable(ctx, d, err)
}

// Get returns the record with the id, or a warp.NotFound.
func (s *Store[T]) Get(ctx context.Context, id int64) (T, error) {
	var v T
	q, d, err := s.ready(ctx)
	if err != nil {
		return v, err
	}
	err = s.t.scan(d, q.QueryRowContext(ctx, s.t.selectSQL(d), id), reflect.ValueOf(&v).Elem())
	if errors.Is(err, sql.ErrNoRows) {
		return v, s.notFound(id)
	}
	return v, unavailable(ctx, d, err)
}

// Create adds v as a new record and sets its ID, CreatedAt and UpdatedAt,
// whatever they held before. A value another record has in a unique field is
// a warp.Invalid on that field, and v is left as it was.
func (s *Store[T]) Create(ctx context.Context, v *T) error {
	q, d, err := s.ready(ctx)
	if err != nil {
		return err
	}
	if v == nil {
		return fmt.Errorf("data: Create of %s was given a nil record", s.name)
	}
	rv := reflect.ValueOf(v).Elem()
	now := s.db.now()
	cols := s.t.inserted()
	args := make([]any, len(cols))
	for i, c := range cols {
		if c.auto { // created_at and updated_at
			args[i] = d.timeValue(now)
			continue
		}
		if args[i], err = s.t.value(d, c, rv); err != nil {
			return err
		}
	}
	var id int64
	if err := q.QueryRowContext(ctx, s.t.insertSQL(d), args...).Scan(&id); err != nil {
		return s.taken(ctx, d, err)
	}
	rec := rv.FieldByIndex(s.t.record).Addr().Interface().(*Record)
	rec.ID, rec.CreatedAt, rec.UpdatedAt = id, now, now
	return nil
}

// Update writes the fields of v that fields names, by their json names, and
// updated_at, to the record with the id, and returns the record as it is
// now; v itself is not changed. With no fields it writes nothing, not even
// updated_at, and returns the record. A name T does not have, or id,
// created_at or updated_at, which the store sets, is an error, and nothing
// is written. A field of T's own tagged warp:"readonly" is written: readonly
// is about what a request may send, and server code may set such a field. A missing
// record is a warp.NotFound; a value another record has in a unique field is
// a warp.Invalid on that field.
func (s *Store[T]) Update(ctx context.Context, id int64, v *T, fields []string) (T, error) {
	var out T
	q, d, err := s.ready(ctx)
	if err != nil {
		return out, err
	}
	var cols []*column
	for _, name := range fields {
		c := s.t.col(name)
		switch {
		case c == nil:
			return out, fmt.Errorf("data: %s has no field %q to update", s.name, name)
		case c.auto:
			return out, fmt.Errorf("data: %s has no field %q to update, since the store sets it", s.name, name)
		}
		if !slices.Contains(cols, c) {
			cols = append(cols, c)
		}
	}
	if len(cols) == 0 {
		return s.Get(ctx, id)
	}
	if v == nil {
		return out, fmt.Errorf("data: Update of %s was given a nil record", s.name)
	}
	rv := reflect.ValueOf(v).Elem()
	args := make([]any, 0, len(cols)+2)
	for _, c := range cols {
		a, err := s.t.value(d, c, rv)
		if err != nil {
			return out, err
		}
		args = append(args, a)
	}
	args = append(args, d.timeValue(s.db.now()), id)
	err = s.t.scan(d, q.QueryRowContext(ctx, s.t.updateSQL(d, cols), args...), reflect.ValueOf(&out).Elem())
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return out, s.notFound(id)
	case err != nil:
		return out, s.taken(ctx, d, err)
	}
	return out, nil
}

// Delete removes the record with the id, or answers a warp.NotFound.
func (s *Store[T]) Delete(ctx context.Context, id int64) error {
	q, d, err := s.ready(ctx)
	if err != nil {
		return err
	}
	res, err := q.ExecContext(ctx, s.t.deleteSQL(d), id)
	if err != nil {
		return unavailable(ctx, d, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return s.notFound(id)
	}
	return nil
}

// DeleteBy removes every record whose field, named as in JSON, equals value,
// and returns how many it removed: none is 0, not an error. value is compared
// as First compares it.
func (s *Store[T]) DeleteBy(ctx context.Context, field string, value any) (int, error) {
	q, d, err := s.ready(ctx)
	if err != nil {
		return 0, err
	}
	c := s.t.col(field)
	if c == nil {
		return 0, fmt.Errorf("data: %s has no field %q to delete by", s.name, field)
	}
	arg, err := s.t.lookupValue(d, c, value)
	if err != nil {
		return 0, err
	}
	return s.deleted(ctx, d, q, s.t.deleteBySQL(d, c), arg)
}

// DeleteBefore removes every record whose time field, named as in JSON, is
// before t (a record at t itself stays), and returns how many it removed. A
// NULL time is before nothing.
func (s *Store[T]) DeleteBefore(ctx context.Context, field string, t time.Time) (int, error) {
	q, d, err := s.ready(ctx)
	if err != nil {
		return 0, err
	}
	c := s.t.col(field)
	switch {
	case c == nil:
		return 0, fmt.Errorf("data: %s has no field %q to delete by", s.name, field)
	case c.kind != kindTime:
		return 0, fmt.Errorf("data: %s cannot delete by %q before a time, since it is not a time; name a time field.", s.name, field)
	}
	return s.deleted(ctx, d, q, s.t.deleteBeforeSQL(d, c), d.timeValue(t))
}

// deleted runs a DELETE and returns how many rows it removed.
func (s *Store[T]) deleted(ctx context.Context, d dialect, q querier, query string, args ...any) (int, error) {
	res, err := q.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, unavailable(ctx, d, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}
	return int(n), nil
}
