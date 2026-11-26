package dbtest

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/flakerimi/weft/warp"
	"github.com/flakerimi/weft/warp/data"
)

// Item is a product whose fields take part in lists.
type Item struct {
	data.Record
	Name   string  `json:"name" warp:"required,unique,max=120" data:"sort,search"`
	Note   *string `json:"note" data:"search"`
	Price  float64 `json:"price" warp:"min=0" data:"sort"`
	Status string  `json:"status" warp:"enum=draft|live" data:"filter"`
}

// makeItems makes 25 items: Item 01 to Item 25, priced 1 to 25 with item 5's
// price 20 as well, every third live, the even ones with a note.
func makeItems(t *testing.T, s *data.Store[Item]) {
	t.Helper()
	for i := 1; i <= 25; i++ {
		it := Item{Name: fmt.Sprintf("Item %02d", i), Price: float64(i), Status: "draft"}
		if i == 5 {
			it.Price = 20
		}
		if i%3 == 0 {
			it.Status = "live"
		}
		if i%2 == 0 {
			note := fmt.Sprintf("note %d", i)
			it.Note = &note
		}
		if err := s.Create(ctx, &it); err != nil {
			t.Fatal(err)
		}
	}
}

func idsOf[T any](l data.List[T]) []int64 {
	out := []int64{}
	for _, it := range l.Items {
		out = append(out, reflect.ValueOf(it).FieldByName("ID").Int())
	}
	return out
}

func listOf(t *testing.T, s *data.Store[Item], q data.Query) data.List[Item] {
	t.Helper()
	l, err := s.List(ctx, q)
	if err != nil {
		t.Fatalf("List(%+v) = %v", q, err)
	}
	return l
}

func TestListPagesSortsSearchesFilters(t *testing.T) {
	_, s := shop[Item](t, dbFile(t))
	makeItems(t, s)

	cases := []struct {
		name  string
		q     data.Query
		ids   []int64
		total int
	}{
		{"page 1 of 10", data.Query{PerPage: 10}, []int64{25, 24, 23, 22, 21, 20, 19, 18, 17, 16}, 25},
		{"page 3 of 10", data.Query{Page: 3, PerPage: 10}, []int64{5, 4, 3, 2, 1}, 25},
		{"past the end", data.Query{Page: 4, PerPage: 10}, []int64{}, 25},
		{"by id", data.Query{Sort: "id", PerPage: 3}, []int64{1, 2, 3}, 25},
		{"by price, ties by id", data.Query{Sort: "price", Page: 5, PerPage: 4}, []int64{18, 19, 5, 20}, 25},
		{"by price, descending", data.Query{Sort: "-price", PerPage: 3}, []int64{25, 24, 23}, 25},
		{"by name, descending", data.Query{Sort: "-name", PerPage: 2}, []int64{25, 24}, 25},
		{"by created_at", data.Query{Sort: "created_at", PerPage: 2}, []int64{1, 2}, 25},
		{"by updated_at, descending", data.Query{Sort: "-updated_at", PerPage: 2}, []int64{25, 24}, 25},
		{"search a name", data.Query{Q: "item 1", PerPage: 100}, []int64{19, 18, 17, 16, 15, 14, 13, 12, 11, 10}, 10},
		{"search in either field", data.Query{Q: "4", Sort: "id"}, []int64{4, 14, 24}, 3},
		{"search the note alone", data.Query{Q: "NOTE 2", Sort: "id"}, []int64{2, 20, 22, 24}, 4},
		{"filter", data.Query{Filter: map[string]string{"status": "live"}, Sort: "id"}, []int64{3, 6, 9, 12, 15, 18, 21, 24}, 8},
		{"an empty filter is none", data.Query{Filter: map[string]string{"status": ""}, PerPage: 1}, []int64{25}, 25},
		{"filter matching nothing", data.Query{Filter: map[string]string{"status": "gone"}}, []int64{}, 0},
		{"all together", data.Query{Q: "item 1", Filter: map[string]string{"status": "live"}, Sort: "-price", Page: 2, PerPage: 2}, []int64{12}, 3},
	}
	for _, c := range cases {
		l := listOf(t, s, c.q)
		if got := idsOf(l); !reflect.DeepEqual(got, c.ids) || l.Total != c.total {
			t.Errorf("%s: ids %v, total %d; want %v, %d", c.name, got, l.Total, c.ids, c.total)
		}
	}

	// The items are whole records.
	l := listOf(t, s, data.Query{Sort: "id", PerPage: 2})
	first, _ := s.Get(ctx, 1)
	if !reflect.DeepEqual(l.Items[0], first) || l.Items[1].Note == nil || *l.Items[1].Note != "note 2" {
		t.Fatalf("items %+v, want %+v first", l.Items, first)
	}
}

func TestListDefaults(t *testing.T) {
	_, s := shop[Item](t, dbFile(t))
	makeItems(t, s)
	l := listOf(t, s, data.Query{})
	if l.Page != 1 || l.PerPage != 20 || l.Total != 25 || len(l.Items) != 20 || l.Items[0].ID != 25 || l.Items[19].ID != 6 {
		t.Fatalf("defaults: page %d, per page %d, total %d, %v", l.Page, l.PerPage, l.Total, idsOf(l))
	}

	// A call from Go may ask for less than 1 or more than 100.
	if l := listOf(t, s, data.Query{Page: -2, PerPage: 500}); l.Page != 1 || l.PerPage != 100 || len(l.Items) != 25 {
		t.Fatalf("page %d, per page %d, %d items", l.Page, l.PerPage, len(l.Items))
	}

	// Over HTTP 0 is the default too, and what the rules refuse is a 422.
	c, s2 := withResource[Item](t)
	makeItems(t, s2)
	var got data.List[Item]
	c.Get("/api/products?page=0&per_page=0&sort=&q=").Want(200).JSON(&got)
	if got.Page != 1 || got.PerPage != 20 || got.Total != 25 || len(got.Items) != 20 || got.Items[0].ID != 25 {
		t.Fatalf("over HTTP: page %d, per page %d, total %d, %v", got.Page, got.PerPage, got.Total, idsOf(got))
	}
	c.Get("/api/products?per_page=100").Want(200).JSON(&got)
	if got.PerPage != 100 || len(got.Items) != 25 {
		t.Fatalf("per_page=100: %d, %d items", got.PerPage, len(got.Items))
	}
	for query, field := range map[string]string{
		"per_page=101":                  "per_page",
		"page=-1":                       "page",
		"q=" + strings.Repeat("a", 201): "q",
	} {
		e := c.Get("/api/products?" + query).Want(422).Error()
		if _, ok := e.Fields[field]; !ok || len(e.Fields) != 1 {
			t.Errorf("%s: %v", query, e.Fields)
		}
	}
	c.Get("/api/products?q=" + strings.Repeat("a", 200)).Want(200)
	if e := c.Get("/api/products?page=two").Want(400).Error(); e.Fields["page"] != "Send a whole number." {
		t.Errorf("page=two: %+v", e)
	}
}

func TestListRefusesUnknownSortAndFilter(t *testing.T) {
	_, s := shop[Item](t, dbFile(t))
	makeItems(t, s)
	cases := []struct {
		q      data.Query
		fields map[string]string
	}{
		{data.Query{Sort: "cost"}, map[string]string{"sort": `products cannot be sorted by "cost"; use one of: created_at, id, name, price, updated_at.`}},
		{data.Query{Sort: "-cost"}, map[string]string{"sort": `products cannot be sorted by "cost"; use one of: created_at, id, name, price, updated_at.`}},
		{data.Query{Sort: "status"}, map[string]string{"sort": `products cannot be sorted by "status"; use one of: created_at, id, name, price, updated_at.`}},
		{data.Query{Sort: "name,price"}, map[string]string{"sort": `products cannot be sorted by "name,price"; use one of: created_at, id, name, price, updated_at.`}},
		{data.Query{Sort: "-"}, map[string]string{"sort": `products cannot be sorted by ""; use one of: created_at, id, name, price, updated_at.`}},
		{data.Query{Filter: map[string]string{"size": "9"}}, map[string]string{"filter[size]": `products cannot be filtered by "size"; use one of: status.`}},
		{data.Query{Filter: map[string]string{"price": ""}}, map[string]string{"filter[price]": `products cannot be filtered by "price"; use one of: status.`}},
		{data.Query{Filter: map[string]string{"id": "1"}}, map[string]string{"filter[id]": `products cannot be filtered by "id"; use one of: status.`}},
		{data.Query{Sort: "cost", Filter: map[string]string{"size": "9", "status": "live"}}, map[string]string{
			"sort":         `products cannot be sorted by "cost"; use one of: created_at, id, name, price, updated_at.`,
			"filter[size]": `products cannot be filtered by "size"; use one of: status.`,
		}},
	}
	for _, c := range cases {
		_, err := s.List(ctx, c.q)
		wantBadRequest(t, fmt.Sprintf("%+v", c.q), err, c.fields)
	}

	// A model with no search or filter field says so.
	_, plain := shop[Product](t, dbFile(t))
	_, err := plain.List(ctx, data.Query{Q: "lamp", Filter: map[string]string{"status": "live"}})
	wantBadRequest(t, "Product", err, map[string]string{
		"q":              `products cannot be searched; no field of Product is tagged data:"search".`,
		"filter[status]": `products cannot be filtered; no field of Product is tagged data:"filter".`,
	})
	// Sorting by the record's fields needs no tag; no text means no search.
	if _, err := plain.List(ctx, data.Query{Sort: "-created_at", Q: ""}); err != nil {
		t.Fatal(err)
	}

	// Over HTTP it is the one error shape.
	c, _ := withResource[Item](t)
	res := c.Get("/api/products?sort=cost&filter[size]=&filter[status]=live").Want(400)
	want := `{"error":{"code":"bad_request","message":"2 fields have a problem.","fields":{` +
		`"filter[size]":"products cannot be filtered by \"size\"; use one of: status.",` +
		`"sort":"products cannot be sorted by \"cost\"; use one of: created_at, id, name, price, updated_at."}}}` + "\n"
	if string(res.Body) != want || res.Header.Get("Content-Type") != "application/json" {
		t.Fatalf("over HTTP %s, want %s", res.Body, want)
	}
}

// Nothing read from the query is unbounded, and a long sort is not echoed.
func TestListLimits(t *testing.T) {
	_, s := shop[Item](t, dbFile(t))
	makeItems(t, s)
	unknownSort := func(name string) map[string]string {
		return map[string]string{"sort": fmt.Sprintf(`products cannot be sorted by %q; use one of: created_at, id, name, price, updated_at.`, name)}
	}
	long := map[string]string{"sort": "The sort field is longer than 64 bytes."}
	for _, c := range []struct {
		q      data.Query
		fields map[string]string
	}{
		{data.Query{Sort: strings.Repeat("s", 64)}, unknownSort(strings.Repeat("s", 64))},
		{data.Query{Sort: "-" + strings.Repeat("s", 63)}, unknownSort(strings.Repeat("s", 63))},
		{data.Query{Sort: strings.Repeat("s", 65)}, long},
		{data.Query{Sort: "-" + strings.Repeat("s", 64)}, long},
		{data.Query{Filter: map[string]string{"status": strings.Repeat("v", 201)}}, map[string]string{"filter[status]": "A filter value is longer than 200 bytes."}},
		{data.Query{Q: "lamp\x00"}, map[string]string{"q": "The search text has a character that cannot be searched for."}},
		{data.Query{Q: "\x00"}, map[string]string{"q": "The search text has a character that cannot be searched for."}},
		{data.Query{Filter: map[string]string{"status": "live\x00"}}, map[string]string{"filter[status]": "The value has a character that cannot be filtered on."}},
		// An unknown key is named as such, whatever its value.
		{data.Query{Filter: map[string]string{"size": strings.Repeat("v", 201) + "\x00"}}, map[string]string{"filter[size]": `products cannot be filtered by "size"; use one of: status.`}},
	} {
		_, err := s.List(ctx, c.q)
		wantBadRequest(t, fmt.Sprintf("sort %d bytes, q %q, filter %d", len(c.q.Sort), c.q.Q, len(c.q.Filter["status"])), err, c.fields)
	}
	// 200 bytes is a value like any other, and matches nothing here.
	if l := listOf(t, s, data.Query{Filter: map[string]string{"status": strings.Repeat("v", 200)}}); l.Total != 0 {
		t.Fatalf("a 200-byte filter: total %d", l.Total)
	}

	// Over HTTP: a 100 KB sort is answered in less than 1 KB.
	c, _ := withResource[Item](t)
	res := c.Get("/api/products?sort=" + strings.Repeat("%3C", 100<<10)).Want(400)
	if len(res.Body) >= 1<<10 || res.Error().Fields["sort"] != long["sort"] {
		t.Fatalf("a 100 KB sort answered %d bytes: %.200s", len(res.Body), res.Body)
	}
	if e := c.Get("/api/products?q=%00").Want(400).Error(); e.Fields["q"] == "" {
		t.Fatalf("q=%%00: %+v", e)
	}
	if e := c.Get("/api/products?filter[status]=%00").Want(400).Error(); e.Fields["filter[status]"] == "" {
		t.Fatalf("filter[status]=%%00: %+v", e)
	}
}

// wantBadRequest fails unless err is the 400 naming fields.
func wantBadRequest(t *testing.T, what string, err error, fields map[string]string) {
	t.Helper()
	var e *warp.Error
	if !errors.As(err, &e) {
		t.Fatalf("%s: %v, want a *warp.Error", what, err)
	}
	message := fmt.Sprintf("%d fields have a problem.", len(fields))
	if len(fields) == 1 {
		message = "1 field has a problem."
	}
	if e.Status != 400 || e.Code != "bad_request" || e.Message != message || !reflect.DeepEqual(e.Fields, fields) {
		t.Errorf("%s: %d %s %q %v\nwant 400 bad_request %q %v", what, e.Status, e.Code, e.Message, e.Fields, message, fields)
	}
}

// Filter values are converted to the field's kind.
func TestListFilterKinds(t *testing.T) {
	type Gauge struct {
		data.Record
		Price float64   `json:"price" data:"filter"`
		Count int       `json:"count" data:"filter"`
		Live  bool      `json:"live" data:"filter"`
		At    time.Time `json:"at" data:"filter"`
		Code  *string   `json:"code" data:"filter"`
	}
	_, s := shop[Gauge](t, dbFile(t))
	at := time.Date(2026, 11, 25, 9, 30, 0, 0, time.UTC)
	code := "A1"
	for _, g := range []Gauge{{Price: 1.5, Count: 3, Live: true, At: at, Code: &code}, {Price: 2, Count: 4}} {
		if err := s.Create(ctx, &g); err != nil {
			t.Fatal(err)
		}
	}
	for filter, want := range map[string][]int64{
		"price=1.5": {1}, "price=2": {2}, "count=4": {2}, "live=true": {1}, "live=0": {2},
		"at=2026-11-25T10:30:00+01:00": {1}, "code=A1": {1}, "code=B2": {},
	} {
		k, v, _ := strings.Cut(filter, "=")
		l, err := s.List(ctx, data.Query{Filter: map[string]string{k: v}})
		if err != nil {
			t.Fatalf("%s: %v", filter, err)
		}
		if got := idsOf(l); !reflect.DeepEqual(got, want) {
			t.Errorf("%s: %v, want %v", filter, got, want)
		}
	}
	_, err := s.List(ctx, data.Query{Filter: map[string]string{"price": "cheap", "count": "1.5", "live": "maybe", "at": "tomorrow", "code": ""}})
	wantBadRequest(t, "kinds", err, map[string]string{
		"filter[price]": "Send a number.",
		"filter[count]": "Send a whole number.",
		"filter[live]":  "Send true or false.",
		"filter[at]":    "Send a time like 2026-11-25T09:30:00Z.",
	})
	_, err = s.List(ctx, data.Query{Filter: map[string]string{"price": "NaN"}})
	wantBadRequest(t, "NaN", err, map[string]string{"filter[price]": "Send a number."})
}

func TestSearchTakesWildcardsLiterally(t *testing.T) {
	_, s := shop[Item](t, dbFile(t))
	for _, name := range []string{"50% off", "500 off", "a_b", "axb", `back\slash`, `backslash`, "Ünïcode lamp"} {
		if err := s.Create(ctx, &Item{Name: name}); err != nil {
			t.Fatal(err)
		}
	}
	for q, want := range map[string][]string{
		"%":          {"50% off"},
		"0% o":       {"50% off"},
		"_":          {"a_b"},
		"a_b":        {"a_b"},
		`\`:          {`back\slash`},
		`k\s`:        {`back\slash`},
		"%%":         {},
		"OFF":        {"50% off", "500 off"},
		"ünïcode":    {}, // LIKE folds ASCII case alone
		"Ünïcode la": {"Ünïcode lamp"},
	} {
		l := listOf(t, s, data.Query{Q: q, Sort: "id"})
		got := []string{}
		for _, it := range l.Items {
			got = append(got, it.Name)
		}
		if !reflect.DeepEqual(got, want) || l.Total != len(want) {
			t.Errorf("q=%q: %q (total %d), want %q", q, got, l.Total, want)
		}
	}
}

func TestListEmptyIsNotNull(t *testing.T) {
	c, s := withResource[Item](t)
	l := listOf(t, s, data.Query{})
	if l.Items == nil || len(l.Items) != 0 || l.Total != 0 || l.Page != 1 || l.PerPage != 20 {
		t.Fatalf("empty list %#v", l)
	}
	want := `{"items":[],"total":0,"page":1,"per_page":20}` + "\n"
	if got := string(c.Get("/api/products").Want(200).Body); got != want {
		t.Fatalf("empty list over HTTP %s, want %s", got, want)
	}

	// A page past any offset an integer holds is empty, with the right total.
	makeItems(t, s)
	want = fmt.Sprintf(`{"items":[],"total":25,"page":%d,"per_page":100}`+"\n", math.MaxInt)
	if got := string(c.Get(fmt.Sprintf("/api/products?page=%d&per_page=100", math.MaxInt)).Want(200).Body); got != want {
		t.Fatalf("a page past any offset %s, want %s", got, want)
	}
	if l := listOf(t, s, data.Query{Page: 4, PerPage: 10, Filter: map[string]string{"status": "live"}}); l.Items == nil || l.Total != 8 {
		t.Fatalf("a page past the end %#v", l)
	}
}
