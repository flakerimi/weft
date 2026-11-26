package data

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"time"
)

// kind is what a column holds, whatever Go type the field has.
type kind int

const (
	kindText kind = iota + 1
	kindInteger
	kindBool
	kindReal
	kindTime
	kindBlob
)

// column is one field of a model, kept in the column named after its json
// name.
type column struct {
	name   string // the json name, and the column's
	field  string // Product.Price, for sentences
	index  []int  // the path to the field, through embedded structs
	kind   kind
	null   bool // a pointer: NULL is nil
	unique bool // the rule unique: a unique index
	auto   bool // id, created_at or updated_at, which the store sets
	// What the column takes part in, in a list: its data tag, read by listTag.
	sort, filter, search bool
}

// table is a model's table: its columns in field order.
type table struct {
	name     string
	model    string // the Go type's name
	singular string // the name of one record, for sentences
	cols     []*column
	record   []int // the path to the embedded Record
}

var (
	recordType = reflect.TypeFor[Record]()
	timeType   = reflect.TypeFor[time.Time]()
	tableName  = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
	notName    = regexp.MustCompile(`[^a-z0-9]+`)
)

// suggestTable is a table name made from one that is not, for a sentence.
func suggestTable(name string) string {
	s := strings.Trim(notName.ReplaceAllString(strings.ToLower(name), "_"), "_")
	switch {
	case s == "":
		return "items"
	case s[0] >= '0' && s[0] <= '9':
		return "t_" + s
	}
	return s
}

// singular is a table's name for one record: the name without a trailing s.
func singular(table string) string {
	if len(table) > 1 && strings.HasSuffix(table, "s") {
		return table[:len(table)-1]
	}
	return table
}

func (t *table) col(name string) *column {
	for _, c := range t.cols {
		if c.name == name {
			return c
		}
	}
	return nil
}

// tableOf reads the columns of the model rt, kept in the table name, and
// returns every mistake in it as a sentence without warp: in front.
func tableOf(rt reflect.Type, name string) (*table, []string) {
	model := rt.Name()
	if model == "" {
		model = rt.String()
	}
	t := &table{name: name, model: model, singular: singular(name)}
	if rt.Kind() != reflect.Struct {
		return t, []string{fmt.Sprintf("%s: %s does not embed data.Record; embed it, since a store needs an id and two times.", name, model)}
	}
	w := walker{t: t, owners: map[string]*column{}}
	w.walk(rt, nil)
	if t.record == nil {
		w.mistakes = append([]string{fmt.Sprintf("%s: %s does not embed data.Record; embed it, since a store needs an id and two times.", name, model)}, w.mistakes...)
	}
	return t, w.mistakes
}

type walker struct {
	t        *table
	owners   map[string]*column
	mistakes []string
}

func (w *walker) fail(format string, args ...any) {
	w.mistakes = append(w.mistakes, w.t.name+": "+fmt.Sprintf(format, args...))
}

// walk adds the fields of rt as encoding/json sees them: an embedded struct
// without a json name gives its fields, json:"-" and unexported fields are
// left out.
func (w *walker) walk(rt reflect.Type, prefix []int) {
	for i := range rt.NumField() {
		f := rt.Field(i)
		index := append(slices.Clone(prefix), i)
		tag, hasTag := f.Tag.Lookup("json")
		if tag == "-" {
			continue
		}
		name, _, _ := strings.Cut(tag, ",")
		if f.Anonymous && name == "" {
			ft := f.Type
			switch {
			case ft == recordType:
				w.t.record = index
				w.walk(ft, index)
				continue
			case ft.Kind() == reflect.Pointer && ft.Elem().Kind() == reflect.Struct:
				if ft.Elem() != recordType { // *Record is not embedding it, which tableOf says
					w.fail("%s embeds *%s, a pointer; embed %s by value.", w.t.model, ft.Elem().Name(), ft.Elem().Name())
				}
				continue
			case ft.Kind() == reflect.Struct && ft != timeType:
				w.walk(ft, index)
				continue
			}
		}
		if !f.IsExported() {
			continue
		}
		at := w.t.model + "." + f.Name
		switch {
		case !hasTag:
			w.fail("%s has no json tag; add one, since the column is named after it.", at)
			continue
		case name == "":
			w.fail("%s has a json tag with no name; name it, since the column is named after it.", at)
			continue
		}
		k, null, ok := kindOf(f.Type)
		if !ok {
			w.fail("%s is a %s, which a store cannot keep; use text, a number, a bool, a time or bytes, or tag it json:\"-\".", at, f.Type)
			continue
		}
		c := &column{name: name, field: at, index: index, kind: k, null: null, auto: rt == recordType}
		for part := range strings.SplitSeq(f.Tag.Get("warp"), ",") {
			if strings.TrimSpace(part) == "unique" {
				c.unique = true
			}
		}
		w.listTag(c, f)
		if other := w.owners[name]; other != nil {
			switch {
			case other.auto:
				w.fail("%s has the json name %q, which data.Record keeps; rename it.", at, name)
			case c.auto:
				w.fail("%s has the json name %q, which data.Record keeps; rename it.", other.field, name)
			default:
				w.fail("%s and %s both have the json name %q; rename one.", other.field, at, name)
			}
			continue
		}
		w.owners[name] = c
		w.t.cols = append(w.t.cols, c)
	}
}

// kindOf is the kind of column a field of type t is kept in, and whether it
// may be NULL; ok is false for a type a store cannot keep.
func kindOf(t reflect.Type) (k kind, null, ok bool) {
	if t.Kind() == reflect.Pointer {
		t, null = t.Elem(), true
	}
	switch {
	case t == timeType:
		return kindTime, null, true
	case t.Kind() == reflect.String:
		return kindText, null, true
	case t.Kind() == reflect.Bool:
		return kindBool, null, true
	case t.Kind() >= reflect.Int && t.Kind() <= reflect.Uint64:
		return kindInteger, null, true
	case t.Kind() == reflect.Float32 || t.Kind() == reflect.Float64:
		return kindReal, null, true
	case t.Kind() == reflect.Slice && t.Elem().Kind() == reflect.Uint8:
		return kindBlob, null, true
	}
	return 0, false, false
}

// The SQL a table needs. Every identifier is quoted and every value is a
// placeholder: no value is ever written into the text.

func (t *table) definition(d dialect, c *column) string {
	if c.name == "id" && c.auto {
		return d.quote(c.name) + " " + d.idColumn()
	}
	def := d.quote(c.name) + " " + d.columnType(c.kind)
	if !c.null {
		def += " NOT NULL DEFAULT " + d.zero(c.kind)
	}
	return def
}

func (t *table) createSQL(d dialect) string {
	defs := make([]string, len(t.cols))
	for i, c := range t.cols {
		defs[i] = t.definition(d, c)
	}
	return "CREATE TABLE " + d.quote(t.name) + " (" + strings.Join(defs, ", ") + ")"
}

func (t *table) addColumnSQL(d dialect, c *column) string {
	return "ALTER TABLE " + d.quote(t.name) + " ADD COLUMN " + t.definition(d, c)
}

// indexName is the name of the unique index on c.
func (t *table) indexName(c *column) string { return t.name + "_" + c.name + "_unique" }

func (t *table) indexSQL(d dialect) []string {
	var out []string
	for _, c := range t.cols {
		if c.unique {
			out = append(out, "CREATE UNIQUE INDEX IF NOT EXISTS "+d.quote(t.indexName(c))+" ON "+d.quote(t.name)+" ("+d.quote(c.name)+")")
		}
	}
	return out
}

// names is every column, quoted, for a SELECT or a RETURNING.
func (t *table) names(d dialect) string {
	names := make([]string, len(t.cols))
	for i, c := range t.cols {
		names[i] = d.quote(c.name)
	}
	return strings.Join(names, ", ")
}

func (t *table) selectSQL(d dialect) string {
	return "SELECT " + t.names(d) + " FROM " + d.quote(t.name) + " WHERE " + d.quote("id") + " = " + d.placeholder(1)
}

// inserted is every column but the id, which the database gives.
func (t *table) inserted() []*column {
	var out []*column
	for _, c := range t.cols {
		if !(c.auto && c.name == "id") {
			out = append(out, c)
		}
	}
	return out
}

func (t *table) insertSQL(d dialect) string {
	cols := t.inserted()
	names, marks := make([]string, len(cols)), make([]string, len(cols))
	for i, c := range cols {
		names[i], marks[i] = d.quote(c.name), d.placeholder(i+1)
	}
	return "INSERT INTO " + d.quote(t.name) + " (" + strings.Join(names, ", ") + ") VALUES (" + strings.Join(marks, ", ") +
		") RETURNING " + d.quote("id")
}

// updateSQL writes cols and updated_at, in that order, of the record whose id
// is the last value, and returns the record.
func (t *table) updateSQL(d dialect, cols []*column) string {
	sets := make([]string, 0, len(cols)+1)
	for i, c := range cols {
		sets = append(sets, d.quote(c.name)+" = "+d.placeholder(i+1))
	}
	sets = append(sets, d.quote("updated_at")+" = "+d.placeholder(len(cols)+1))
	return "UPDATE " + d.quote(t.name) + " SET " + strings.Join(sets, ", ") + " WHERE " + d.quote("id") + " = " + d.placeholder(len(cols)+2) +
		" RETURNING " + t.names(d)
}

func (t *table) deleteSQL(d dialect) string {
	return "DELETE FROM " + d.quote(t.name) + " WHERE " + d.quote("id") + " = " + d.placeholder(1)
}

// value is what column c of the model rv is written as.
func (t *table) value(d dialect, c *column, rv reflect.Value) (any, error) {
	fv := rv.FieldByIndex(c.index)
	if fv.Kind() == reflect.Pointer {
		if fv.IsNil() {
			return nil, nil
		}
		fv = fv.Elem()
	}
	switch c.kind {
	case kindText:
		return fv.String(), nil
	case kindBool:
		return fv.Bool(), nil
	case kindReal:
		return fv.Float(), nil
	case kindTime:
		return d.timeValue(fv.Interface().(time.Time)), nil
	case kindBlob:
		if b := fv.Bytes(); b != nil {
			return b, nil
		}
		return []byte{}, nil // a NOT NULL column is never NULL
	}
	if fv.CanUint() {
		if u := fv.Uint(); u > math.MaxInt64 {
			return nil, fmt.Errorf("data: %s is %d, more than the column %s.%s can keep", c.field, u, t.name, c.name)
		}
		return int64(fv.Uint()), nil
	}
	return fv.Int(), nil
}

// scanner is a *sql.Row or *sql.Rows.
type scanner interface{ Scan(dest ...any) error }

// scan reads one row of every column into the model rv.
func (t *table) scan(d dialect, row scanner, rv reflect.Value) error {
	held := make([]any, len(t.cols))
	ptrs := make([]any, len(t.cols))
	for i := range held {
		ptrs[i] = &held[i]
	}
	if err := row.Scan(ptrs...); err != nil {
		return err
	}
	for i, c := range t.cols {
		if err := t.set(d, c, rv.FieldByIndex(c.index), held[i]); err != nil {
			return err
		}
	}
	return nil
}

// set puts src, a value the driver read from column c, into the field fv.
func (t *table) set(d dialect, c *column, fv reflect.Value, src any) error {
	if src == nil {
		fv.SetZero()
		if c.kind == kindBlob && !c.null {
			fv.SetBytes([]byte{})
		}
		return nil
	}
	e := fv
	if fv.Kind() == reflect.Pointer {
		e = reflect.New(fv.Type().Elem()).Elem()
	}
	wrong := fmt.Errorf("data: column %s.%s holds %v, which %s cannot take; check what wrote it", t.name, c.name, src, c.field)
	switch c.kind {
	case kindText:
		switch s := src.(type) {
		case string:
			e.SetString(s)
		case []byte:
			e.SetString(string(s))
		default:
			return wrong
		}
	case kindBool:
		switch b := src.(type) {
		case bool:
			e.SetBool(b)
		case int64:
			e.SetBool(b != 0)
		default:
			return wrong
		}
	case kindReal:
		switch n := src.(type) {
		case float64:
			e.SetFloat(n)
		case int64:
			e.SetFloat(float64(n))
		default:
			return wrong
		}
	case kindTime:
		at, err := d.scanTime(src)
		if err != nil {
			return fmt.Errorf("data: column %s.%s holds %v, which %s cannot take; check what wrote it", t.name, c.name, src, c.field)
		}
		e.Set(reflect.ValueOf(at))
	case kindBlob:
		b, ok := src.([]byte)
		if !ok {
			s, isText := src.(string)
			if !isText {
				return wrong
			}
			b = []byte(s)
		}
		e.SetBytes(b)
	case kindInteger:
		n, ok := src.(int64)
		if !ok {
			return wrong
		}
		switch {
		case e.CanUint():
			if n < 0 || e.OverflowUint(uint64(n)) {
				return wrong
			}
			e.SetUint(uint64(n))
		default:
			if e.OverflowInt(n) {
				return wrong
			}
			e.SetInt(n)
		}
	}
	if fv.Kind() == reflect.Pointer {
		fv.Set(e.Addr())
	}
	return nil
}

// querier is a *sql.DB or a *sql.Tx.
type querier interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// migrate brings every table up to date, in one transaction: a missing
// table is made, a missing column is added, and a column of another type
// stops it all. Nothing is dropped or changed.
func migrate(ctx context.Context, db *sql.DB, d dialect, tables []*table) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, t := range tables {
		have, err := d.columns(ctx, tx, t.name)
		if err != nil {
			return err
		}
		var steps []string
		if len(have) == 0 {
			steps = append(steps, t.createSQL(d))
		}
		for _, c := range t.cols {
			typ, ok := have[c.name]
			switch {
			case len(have) == 0:
			case !ok:
				steps = append(steps, t.addColumnSQL(d, c))
			case !strings.EqualFold(typ, d.columnType(c.kind)):
				return fmt.Errorf("warp: table %s has column %s as %s, and %s needs %s; change the field back, or alter the column yourself, since Warp never changes or drops a column.",
					t.name, c.name, typ, c.field, d.columnType(c.kind))
			}
		}
		for _, step := range append(steps, t.indexSQL(d)...) {
			if _, err := tx.ExecContext(ctx, step); err != nil {
				if name, ok := d.uniqueColumn(err, t.name); ok {
					if c := t.col(name); c != nil {
						return fmt.Errorf("warp: table %s has the same %s twice or more, so %s cannot be unique; make the values differ, make the field a pointer so that rows without a value hold NULL, or remove the rule.", t.name, c.name, c.field)
					}
				}
				return fmt.Errorf("warp: table %s cannot be brought up to date (%w); check the database DB_URL names.", t.name, err)
			}
		}
	}
	return tx.Commit()
}
