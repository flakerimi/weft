package data

import (
	"math"
	"reflect"
	"strings"
	"testing"
)

func TestDataTagMistakes(t *testing.T) {
	type Product struct {
		Record
		Name  string  `json:"name" data:"sort,filter,search"`
		Price float64 `json:"price" data:"search"`
		Stock int     `json:"stock" data:"sort, rank"`
		Live  bool    `json:"live" data:"filter,search"`
		Code  *string `json:"code" data:"search"`
		Note  string  `json:"note" data:" sort , filter ,"`
		Photo []byte  `json:"photo" data:"filter"`
		Thumb []byte  `json:"thumb" data:"sort,search"`
		Raw   []byte  `json:"raw" data:""`
	}
	got := stores(t, func(db *DB) { NewStore[Product](db, "products") })
	want := []string{
		`warp: products: Product.Price is tagged data:"search", and search is for text; use sort or filter.`,
		`warp: products: Product.Stock is tagged data:"sort, rank", and rank is none of sort, filter and search; use those, comma-separated.`,
		`warp: products: Product.Live is tagged data:"filter,search", and search is for text; use sort or filter.`,
		`warp: products: Product.Photo is tagged data:"filter", and bytes cannot be filtered, sorted or searched; remove the tag.`,
		`warp: products: Product.Thumb is tagged data:"sort,search", and bytes cannot be filtered, sorted or searched; remove the tag.`,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("mistakes\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}

	// What each column takes part in; the record's three can always be sorted on.
	type Fine struct {
		Record
		Name  string  `json:"name" data:"sort,filter,search"`
		Code  *string `json:"code" data:"search"`
		Note  string  `json:"note" data:" sort , filter ,"`
		Price float64 `json:"price"`
	}
	tb, mistakes := tableOf(reflect.TypeFor[Fine](), "fines")
	if len(mistakes) > 0 {
		t.Fatal(mistakes)
	}
	type part struct{ Sort, Filter, Search bool }
	wantParts := map[string]part{
		"id": {true, false, false}, "created_at": {true, false, false}, "updated_at": {true, false, false},
		"name": {true, true, true}, "code": {false, false, true}, "note": {true, true, false}, "price": {},
	}
	for _, c := range tb.cols {
		if got := (part{c.sort, c.filter, c.search}); got != wantParts[c.name] {
			t.Errorf("%s takes part as %+v, want %+v", c.name, got, wantParts[c.name])
		}
	}
}

func TestListSQL(t *testing.T) {
	type Item struct {
		Record
		Name   string  `json:"name" data:"sort,search"`
		Note   *string `json:"note" data:"search"`
		Price  float64 `json:"price" data:"sort,filter"`
		Status string  `json:"status" data:"filter"`
		Live   bool    `json:"live" data:"filter"`
	}
	tb, mistakes := tableOf(reflect.TypeFor[Item](), "items")
	if len(mistakes) > 0 {
		t.Fatal(mistakes)
	}
	var d sqlite
	cols := `"id", "created_at", "updated_at", "name", "note", "price", "status", "live"`

	// Sort, search and filters together. An empty filter value is no filter.
	l, err := tb.list(d, Query{Page: 3, PerPage: 10, Sort: "-price", Q: `50%_off\`, Filter: map[string]string{"status": "live", "price": "12.5", "live": ""}})
	if err != nil {
		t.Fatal(err)
	}
	where := ` WHERE "price" = ? AND "status" = ? AND ("name" LIKE ? ESCAPE '\' OR "note" LIKE ? ESCAPE '\')`
	term := `%50\%\_off\\%`
	if want := `SELECT count(*) FROM "items"` + where; l.count != want {
		t.Errorf("count\n%s\nwant\n%s", l.count, want)
	}
	if want := []any{12.5, "live", term, term}; !reflect.DeepEqual(l.args, want) {
		t.Errorf("count args %#v, want %#v", l.args, want)
	}
	if want := `SELECT ` + cols + ` FROM "items"` + where + ` ORDER BY "price" DESC, "id" DESC LIMIT ? OFFSET ?`; l.rows != want {
		t.Errorf("rows\n%s\nwant\n%s", l.rows, want)
	}
	if want := []any{12.5, "live", term, term, int64(10), int64(20)}; !reflect.DeepEqual(l.rowArgs, want) {
		t.Errorf("row args %#v, want %#v", l.rowArgs, want)
	}
	if l.page != 3 || l.perPage != 10 {
		t.Errorf("page %d, per page %d", l.page, l.perPage)
	}

	// The defaults: page 1, 20 a page, the newest first.
	l, err = tb.list(d, Query{})
	if err != nil {
		t.Fatal(err)
	}
	if l.count != `SELECT count(*) FROM "items"` || len(l.args) != 0 {
		t.Errorf("count %s %v", l.count, l.args)
	}
	if want := `SELECT ` + cols + ` FROM "items" ORDER BY "id" DESC LIMIT ? OFFSET ?`; l.rows != want {
		t.Errorf("rows\n%s\nwant\n%s", l.rows, want)
	}
	if want := []any{int64(20), int64(0)}; !reflect.DeepEqual(l.rowArgs, want) || l.page != 1 || l.perPage != 20 {
		t.Errorf("row args %#v, page %d, per page %d", l.rowArgs, l.page, l.perPage)
	}

	// Ascending by a field, ties by id; a call from Go asking for more than
	// 100 gets 100.
	l, _ = tb.list(d, Query{Sort: "name", PerPage: 500})
	if !strings.HasSuffix(l.rows, ` ORDER BY "name" ASC, "id" ASC LIMIT ? OFFSET ?`) || l.perPage != 100 {
		t.Errorf("rows %s, per page %d", l.rows, l.perPage)
	}

	// A page whose offset no integer holds is past every row: only the count
	// is asked.
	l, err = tb.list(d, Query{Page: math.MaxInt, PerPage: 100})
	if err != nil || l.rows != "" || l.count == "" || l.page != math.MaxInt {
		t.Errorf("a page past any offset: %+v, %v", l, err)
	}
	if l, _ = tb.list(d, Query{Page: math.MaxInt64/100 + 1, PerPage: 100}); l.rows == "" {
		t.Error("the last page whose offset an int64 holds was not asked")
	}
}
