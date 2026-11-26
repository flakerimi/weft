package data

import (
	"cmp"
	"context"
	"fmt"
	"maps"
	"math"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/flakerimi/weft/warp"
)

// Query is what a list is asked for, an ordinary typed input read from the
// query string, such as ?page=2&per_page=50&sort=-price&q=lamp&filter[status]=live.
//
// A field of the model takes part by its data tag: data:"sort", "filter" or
// "search", or several of them comma-separated. id, created_at and updated_at
// can always be sorted on.
//
//   - Page is counted from 1; 0 is 1.
//   - PerPage is how many records a page holds; 0 is 20, and at most 100.
//   - Sort is the json name of a sort field, with - in front for descending;
//     empty is -id, the newest first. Records that sort the same are in the
//     order of their ids, the same way round. One field only: a comma is a
//     field of that name, and there is none.
//   - Q keeps the records in which any search field contains the text. Case
//     is ignored as SQLite's LIKE ignores it, for the ASCII letters alone; %
//     and _ are taken literally. Empty is no search.
//   - Filter keeps the records whose filter field equals the value, converted
//     to the field's kind; filter[status]= with no value is no filter.
//
// A field that cannot be sorted, filtered or searched by, a filter value of
// the wrong kind, a sort longer than 64 bytes, a filter value longer than 200
// bytes, a Q longer than 200 characters, and a NUL byte in Q or a filter
// value are each a 400 naming the query's key; a page or per_page out of its
// range is a 422, as any rule is.
type Query struct {
	Page    int               `query:"page" warp:"min=1"`
	PerPage int               `query:"per_page" warp:"min=1,max=100"`
	Sort    string            `query:"sort"`
	Q       string            `query:"q"`
	Filter  map[string]string `query:"filter"`
}

// List is one page of a list: its records, never null, and the number of
// records the search and filters keep on every page together.
type List[T any] struct {
	Items   []T `json:"items"`
	Total   int `json:"total"`
	Page    int `json:"page"`
	PerPage int `json:"per_page"`
}

// The number of records on a page when the query names none, and the most;
// and the longest sort and filter value a query may send, in bytes. A sort
// longer than any field name is refused without being quoted back.
const (
	defaultPerPage = 20
	maxPerPage     = 100
	maxSort        = 64
	maxFilterValue = 200
	maxSearch      = 200 // characters
)

// listTag reads the data tag of the field f, whose column is c. The record's
// own columns can always be sorted on.
func (w *walker) listTag(c *column, f reflect.StructField) {
	c.sort = c.auto
	tag, ok := f.Tag.Lookup("data")
	if !ok {
		return
	}
	for part := range strings.SplitSeq(tag, ",") {
		part = strings.TrimSpace(part)
		if c.kind == kindBlob && (part == "sort" || part == "filter" || part == "search") {
			// The API sends bytes as base64, so no value from it compares.
			w.fail("%s is tagged data:%q, and bytes cannot be filtered, sorted or searched; remove the tag.", c.field, tag)
			return
		}
		switch part {
		case "":
		case "sort":
			c.sort = true
		case "filter":
			c.filter = true
		case "search":
			if c.kind != kindText {
				w.fail("%s is tagged data:%q, and search is for text; use sort or filter.", c.field, tag)
				continue
			}
			c.search = true
		default:
			w.fail("%s is tagged data:%q, and %s is none of sort, filter and search; use those, comma-separated.", c.field, tag, part)
		}
	}
}

// listing is the SQL of one list: the count, and the page with its values,
// which are the count's and then the limit and the offset. rows is empty when
// the page starts past any row a table can hold.
type listing struct {
	count, rows   string
	args, rowArgs []any
	page, perPage int
}

// list makes the SQL of the list q asks for, or the 400 that names each part
// of q the table cannot take. Nothing from q is written into the text: the
// sort and filter names are looked up among the columns and their names
// written, and every value is a placeholder.
func (t *table) list(d dialect, q Query) (*listing, error) {
	l := &listing{page: max(q.Page, 1), perPage: q.PerPage}
	switch {
	case l.perPage <= 0:
		l.perPage = defaultPerPage
	case l.perPage > maxPerPage:
		l.perPage = maxPerPage
	}
	problems := map[string]string{}
	var conds []string
	arg := func(v any) string {
		l.args = append(l.args, v)
		return d.placeholder(len(l.args))
	}

	sort, desc := strings.CutPrefix(cmp.Or(q.Sort, "-id"), "-")
	by := t.col(sort)
	switch {
	case len(q.Sort) > maxSort:
		problems["sort"] = fmt.Sprintf("The sort field is longer than %d bytes.", maxSort)
	case by == nil || !by.sort:
		problems["sort"] = fmt.Sprintf("%s cannot be sorted by %q; use one of: %s.", t.name, sort, t.listed(func(c *column) bool { return c.sort }))
	}

	filters := t.listed(func(c *column) bool { return c.filter })
	for _, name := range slices.Sorted(maps.Keys(q.Filter)) {
		key := "filter[" + name + "]"
		c := t.col(name)
		switch {
		case filters == "":
			problems[key] = fmt.Sprintf("%s cannot be filtered; no field of %s is tagged data:\"filter\".", t.name, t.model)
			continue
		case c == nil || !c.filter:
			problems[key] = fmt.Sprintf("%s cannot be filtered by %q; use one of: %s.", t.name, name, filters)
			continue
		case q.Filter[name] == "":
			continue // no filter, as the table on the page sends an empty one
		case len(q.Filter[name]) > maxFilterValue:
			problems[key] = fmt.Sprintf("A filter value is longer than %d bytes.", maxFilterValue)
			continue
		case strings.ContainsRune(q.Filter[name], 0):
			problems[key] = "The value has a character that cannot be filtered on."
			continue
		}
		v, wrong := filterValue(d, c, q.Filter[name])
		if wrong != "" {
			problems[key] = wrong
			continue
		}
		conds = append(conds, d.quote(c.name)+" = "+arg(v))
	}

	if q.Q != "" {
		var ors []string
		text := d.containsText(q.Q)
		for _, c := range t.cols {
			if c.search {
				ors = append(ors, d.contains(c.name, len(l.args)+1))
				l.args = append(l.args, text)
			}
		}
		switch {
		case len(ors) == 0:
			problems["q"] = fmt.Sprintf("%s cannot be searched; no field of %s is tagged data:\"search\".", t.name, t.model)
		case utf8.RuneCountInString(q.Q) > maxSearch:
			problems["q"] = fmt.Sprintf("The search text is longer than %d characters.", maxSearch)
		case strings.ContainsRune(q.Q, 0): // LIKE ends a pattern at a NUL, which would match every row
			problems["q"] = "The search text has a character that cannot be searched for."
		default:
			conds = append(conds, "("+strings.Join(ors, " OR ")+")")
		}
	}

	if len(problems) > 0 {
		e := warp.BadRequest(fmt.Sprintf("%d fields have a problem.", len(problems)))
		if len(problems) == 1 {
			e.Message = "1 field has a problem."
		}
		e.Fields = problems
		return nil, e
	}

	where := ""
	if len(conds) > 0 {
		where = " WHERE " + strings.Join(conds, " AND ")
	}
	from := " FROM " + d.quote(t.name) + where
	l.count = "SELECT count(*)" + from
	// A page whose offset no int64 holds is past every row.
	if int64(l.page-1) > math.MaxInt64/int64(l.perPage) {
		return l, nil
	}
	dir := " ASC"
	if desc {
		dir = " DESC"
	}
	order := d.quote(by.name) + dir
	if by.name != "id" {
		order += ", " + d.quote("id") + dir
	}
	l.rowArgs = append(slices.Clone(l.args), int64(l.perPage), int64(l.page-1)*int64(l.perPage))
	n := len(l.args)
	l.rows = "SELECT " + t.names(d) + from + " ORDER BY " + order +
		" LIMIT " + d.placeholder(n+1) + " OFFSET " + d.placeholder(n+2)
	return l, nil
}

// listed is the names of the columns that keep, in alphabetical order, for a
// sentence.
func (t *table) listed(keep func(c *column) bool) string {
	var names []string
	for _, c := range t.cols {
		if keep(c) {
			names = append(names, c.name)
		}
	}
	slices.Sort(names)
	return strings.Join(names, ", ")
}

// filterValue converts s, a filter's value, to what column c holds, or says
// what to send in its place.
func filterValue(d dialect, c *column, s string) (any, string) {
	switch c.kind {
	case kindInteger:
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return nil, "Send a whole number."
		}
		return n, ""
	case kindReal:
		f, err := strconv.ParseFloat(s, 64)
		if err != nil || math.IsNaN(f) || math.IsInf(f, 0) {
			return nil, "Send a number."
		}
		return f, ""
	case kindBool:
		b, err := strconv.ParseBool(s)
		if err != nil {
			return nil, "Send true or false."
		}
		return b, ""
	case kindTime:
		at, err := time.Parse(time.RFC3339Nano, s)
		if err != nil {
			return nil, "Send a time like 2026-11-25T09:30:00Z."
		}
		return d.timeValue(at), ""
	case kindBlob:
		return []byte(s), ""
	}
	return s, ""
}

// List returns the page of records q asks for, and how many records its
// search and filters keep in all. A page past the end has no items. A part
// of q the model cannot take is a 400 naming each one, with the field's name
// in the query: sort, q or filter[name]. Called from Go, a page below 1 is 1
// and more than 100 records a page is 100.
func (s *Store[T]) List(ctx context.Context, q Query) (List[T], error) {
	out := List[T]{Items: []T{}}
	conn, d, err := s.ready(ctx)
	if err != nil {
		return out, err
	}
	l, err := s.t.list(d, q)
	if err != nil {
		return out, err
	}
	out.Page, out.PerPage = l.page, l.perPage
	var total int64
	if err := conn.QueryRowContext(ctx, l.count, l.args...).Scan(&total); err != nil {
		return out, unavailable(ctx, d, err)
	}
	out.Total = int(total)
	if l.rows == "" {
		return out, nil
	}
	rows, err := conn.QueryContext(ctx, l.rows, l.rowArgs...)
	if err != nil {
		return out, unavailable(ctx, d, err)
	}
	defer rows.Close()
	for rows.Next() {
		var v T
		if err := s.t.scan(d, rows, reflect.ValueOf(&v).Elem()); err != nil {
			return out, unavailable(ctx, d, err)
		}
		out.Items = append(out.Items, v)
	}
	return out, unavailable(ctx, d, rows.Err())
}
