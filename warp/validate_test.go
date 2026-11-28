package warp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"
)

func fieldsOf(t *testing.T, v any) map[string]string {
	t.Helper()
	err := Validate(v)
	if err == nil {
		return nil
	}
	var e *Error
	if !errors.As(err, &e) {
		t.Fatalf("Validate returned %T, want *Error", err)
	}
	return e.Fields
}

func TestValidateMessages(t *testing.T) {
	type S struct {
		Name string `json:"name" warp:"required,max=5"`
	}
	type N struct {
		Age float64 `json:"age" warp:"min=0.5,max=120"`
	}
	type I struct {
		Qty int `json:"qty" warp:"required,max=10"`
	}
	type L struct {
		Tags []string `json:"tags" warp:"min=2,max=3"`
	}
	type E struct {
		Email string `json:"email" warp:"email"`
	}
	type U struct {
		Site string `json:"site" warp:"url"`
	}
	type En struct {
		Status string `json:"status" warp:"enum=draft|live|gone"`
	}
	type EnNum struct {
		Tier int `json:"tier" warp:"enum=1|2|5"`
	}
	type EnFloat struct {
		Rate float64 `json:"rate" warp:"enum=0.25|0.5|1"`
	}
	type EnFloat32 struct {
		Rate float32 `json:"rate" warp:"enum=0.1|0.5|1"`
	}
	type Mn struct {
		Name string `json:"name" warp:"min=3"`
	}
	type Q struct {
		Page uint `query:"page" warp:"min=1,max=9"`
	}
	type P struct {
		ID string `path:"id" warp:"required"`
	}
	type B struct {
		On bool `json:"on" warp:"required"`
	}
	type T struct {
		At time.Time `json:"at" warp:"required"`
	}
	tests := []struct {
		name string
		v    any
		key  string
		want string
	}{
		{"max string", S{Name: "abcdef"}, "name", "Use at most 5 characters."},
		{"required string", S{}, "name", "This is required."},
		{"min string", Mn{Name: "ab"}, "name", "Use at least 3 characters."},
		{"min decimal", N{Age: 0.2}, "age", "Enter 0.5 or more."},
		{"max number", N{Age: 121}, "age", "Enter 120 or less."},
		{"required int zero", I{}, "qty", "This is required."},
		{"max int", I{Qty: 11}, "qty", "Enter 10 or less."},
		{"min slice", L{Tags: []string{"a"}}, "tags", "Choose at least 2."},
		{"max slice", L{Tags: []string{"a", "b", "c", "d"}}, "tags", "Choose at most 3."},
		{"email", E{Email: "nope"}, "email", "Enter an email address."},
		{"email display name", E{Email: "Ada <ada@example.com>"}, "email", "Enter an email address."},
		{"url no scheme", U{Site: "example.com"}, "site", "Enter a web address that starts with http:// or https://."},
		{"url ftp", U{Site: "ftp://example.com"}, "site", "Enter a web address that starts with http:// or https://."},
		{"url no host", U{Site: "https://"}, "site", "Enter a web address that starts with http:// or https://."},
		{"enum", En{Status: "other"}, "status", "Choose one of: draft, live, gone."},
		{"enum number", EnNum{Tier: 3}, "tier", "Choose one of: 1, 2, 5."},
		{"enum float", EnFloat{Rate: 0.75}, "rate", "Choose one of: 0.25, 0.5, 1."},
		{"enum float32", EnFloat32{Rate: 0.2}, "rate", "Choose one of: 0.1, 0.5, 1."},
		{"query key", Q{Page: 10}, "page", "Enter 9 or less."},
		{"path key", P{}, "id", "This is required."},
		{"required bool", B{}, "on", "This is required."},
		{"required time", T{}, "at", "This is required."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fieldsOf(t, tt.v)
			if got[tt.key] != tt.want || len(got) != 1 {
				t.Fatalf("fields = %v, want only %s: %s", got, tt.key, tt.want)
			}
		})
	}
	ok := []any{
		S{Name: "abcde"}, N{Age: 0.5}, I{Qty: 3}, L{Tags: []string{"a", "b"}},
		E{Email: "ada@example.com"}, U{Site: "https://example.com/x"},
		En{Status: "live"}, EnNum{Tier: 2}, EnNum{}, EnFloat{Rate: 0.5}, EnFloat32{Rate: 0.1}, B{On: true}, T{At: time.Now()},
	}
	for _, v := range ok {
		if got := fieldsOf(t, v); got != nil {
			t.Errorf("%T: unexpected %v", v, got)
		}
	}
}

func TestValidateCountsCharacters(t *testing.T) {
	type S struct {
		Name string `json:"name" warp:"max=4"`
	}
	if got := fieldsOf(t, S{Name: "çikë"}); got != nil {
		t.Fatalf("got %v", got)
	}
	if got := fieldsOf(t, S{Name: "çikëx"}); got["name"] != "Use at most 4 characters." {
		t.Fatalf("got %v", got)
	}
}

func TestValidateOptional(t *testing.T) {
	type S struct {
		Email string  `json:"email" warp:"email"`
		Qty   int     `json:"qty" warp:"min=1"`
		Ptr   *string `json:"ptr" warp:"email"`
		Site  string  `json:"site" warp:"url"`
		Tags  []int   `json:"tags" warp:"min=1"`
		Kind  string  `json:"kind" warp:"enum=a|b"`
	}
	if got := fieldsOf(t, S{}); got != nil {
		t.Fatalf("got %v", got)
	}
	type R struct {
		Ptr *string `json:"ptr" warp:"required"`
	}
	if got := fieldsOf(t, R{}); got["ptr"] != "This is required." {
		t.Fatalf("got %v", got)
	}
	bad := "nope"
	if got := fieldsOf(t, S{Ptr: &bad}); got["ptr"] != "Enter an email address." {
		t.Fatalf("pointer rules must apply to the value: %v", got)
	}
}

func TestValidateRequiredBlank(t *testing.T) {
	type S struct {
		Name string `json:"name" warp:"required"`
	}
	if got := fieldsOf(t, S{Name: "   "}); got["name"] != "This is required." {
		t.Fatalf("got %v", got)
	}
}

func TestValidateFirstRuleOnly(t *testing.T) {
	type S struct {
		Email string `json:"email" warp:"required,email"`
	}
	if got := fieldsOf(t, S{}); got["email"] != "This is required." {
		t.Fatalf("got %v", got)
	}
}

func TestValidateNested(t *testing.T) {
	type Address struct {
		City string `json:"city" warp:"required"`
	}
	type Line struct {
		SKU string `json:"sku" warp:"required"`
	}
	type Order struct {
		Address Address  `json:"address"`
		Lines   []Line   `json:"lines"`
		Ship    *Address `json:"ship"`
	}
	got := fieldsOf(t, &Order{Lines: []Line{{SKU: "a"}, {}}})
	want := map[string]string{"address.city": "This is required.", "lines.1.sku": "This is required."}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestValidateSummary(t *testing.T) {
	type S struct {
		A string `json:"a" warp:"required"`
		B string `json:"b" warp:"required"`
	}
	err := Validate(S{})
	var e *Error
	if !errors.As(err, &e) || e.Status != http.StatusUnprocessableEntity || e.Message != "2 fields need attention." {
		t.Fatalf("got %#v", err)
	}
	if err := Validate(S{A: "x", B: "y"}); err != nil {
		t.Fatalf("got %v", err)
	}
	var nilS *S
	if Validate(nilS) != nil || Validate(5) != nil || Validate(nil) != nil {
		t.Fatal("a nil pointer or a non-struct must give nil")
	}
}

func TestValidateKeysAndSkips(t *testing.T) {
	type Base struct {
		ID string `json:"id,omitempty" warp:"required"`
	}
	type S struct {
		Base
		Hidden  string `json:"-" warp:"required"`
		Bare    string `warp:"required"`
		private string `query:"private" warp:"required"`
	}
	_ = S{}.private
	got := fieldsOf(t, S{})
	want := map[string]string{"id": "This is required."}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	type Ptr struct {
		*Base
	}
	if got := fieldsOf(t, Ptr{}); got != nil {
		t.Fatalf("a nil embedded pointer is skipped: %v", got)
	}
}

func TestValidateTimeIsLeaf(t *testing.T) {
	type S struct {
		At time.Time `json:"at"`
	}
	if got := fieldsOf(t, S{}); got != nil {
		t.Fatalf("got %v", got)
	}
}

func TestCheckRules(t *testing.T) {
	type Product struct {
		Price  float64 `json:"price" warp:"emial"`
		Mail   float64 `json:"mail" warp:"email"`
		Name   string  `json:"name" warp:"min=abc,max=-1"`
		Status string  `json:"status" warp:"enum="`
		Flag   bool    `json:"flag" warp:"min=1"`
		Site   int     `json:"site" warp:"url"`
		Kind   bool    `json:"kind" warp:"enum=a|b"`
		Bool2  bool    `json:"bool2" warp:"enum=a,enum=b"`
		Tier   int     `json:"tier" warp:"enum=1|two"`
		Half   int     `json:"half" warp:"enum=1|1.5"`
		Small  uint8   `json:"small" warp:"enum=1|300"`
		Neg8   uint8   `json:"neg8" warp:"enum=1|-3"`
		Neg    uint    `json:"neg" warp:"enum=0|-3"`
		BigU64 uint64  `json:"bigu64" warp:"enum=1|18446744073709551616"`
		Neg0I  int     `json:"neg0i" warp:"enum=-0|1"`
		PlusI  int     `json:"plusi" warp:"enum=+5|1"`
		Lead0I int     `json:"lead0i" warp:"enum=05|1"`
		NegL0I int     `json:"negl0i" warp:"enum=-05|1"`
		Neg0U  uint8   `json:"neg0u" warp:"enum=-0|1"`
		PlusU  uint8   `json:"plusu" warp:"enum=+5|1"`
		Lead0U uint8   `json:"lead0u" warp:"enum=05|1"`
		NegL0U uint8   `json:"negl0u" warp:"enum=-05|1"`
		Hex    float64 `json:"hex" warp:"enum=0x1p4|2"`
		Inex32 float32 `json:"inex32" warp:"enum=1|16777217"`
		Score  int     `json:"score" warp:"enum=1_0|2"`
		Twice  string  `json:"twice" warp:"enum=a|b,enum=b|c"`
		Tags   []int   `json:"tags" warp:"unique"`
		Dec    string  `json:"dec" warp:"min=1.5"`
		Bare   string  `json:"bare" warp:"min"`
	}
	got := checkRules(reflect.TypeOf(Product{}))
	want := []string{
		`Product.Price has the rule "emial", which warp does not know; use required, email, url, min=, max=, enum=, date, unique or readonly.`,
		`Product.Mail has the rule "email", which is for text; remove it or make the field a string.`,
		`Product.Name has "min=abc"; write a number after min=.`,
		`Product.Name has "max=-1"; write a number after max=.`,
		`Product.Status has "enum=" with nothing to choose from; write enum=a|b.`,
		`Product.Flag has the rule "min", which is for text, numbers and lists; remove it.`,
		`Product.Site has the rule "url", which is for text; remove it or make the field a string.`,
		`Product.Kind has the rule "enum", which is for text and numbers; remove it or make the field a string or a number.`,
		`Product.Bool2 has the rule "enum", which is for text and numbers; remove it or make the field a string or a number.`,
		`Product.Tier has "enum=1|two"; write whole numbers separated by | after enum=.`,
		`Product.Half has "enum=1|1.5"; write whole numbers separated by | after enum=.`,
		`Product.Small has "enum=1|300"; write whole numbers from 0 to 255 separated by | after enum=.`,
		`Product.Neg8 has "enum=1|-3"; write whole numbers from 0 to 255 separated by | after enum=.`,
		`Product.Neg has "enum=0|-3"; write whole numbers from 0 to 18446744073709551615 separated by | after enum=.`,
		`Product.BigU64 has "enum=1|18446744073709551616"; write whole numbers from 0 to 18446744073709551615 separated by | after enum=.`,
		`Product.Neg0I has "enum=-0|1"; write whole numbers separated by | after enum=.`,
		`Product.PlusI has "enum=+5|1"; write whole numbers separated by | after enum=.`,
		`Product.Lead0I has "enum=05|1"; write whole numbers separated by | after enum=.`,
		`Product.NegL0I has "enum=-05|1"; write whole numbers separated by | after enum=.`,
		`Product.Neg0U has "enum=-0|1"; write whole numbers separated by | after enum=.`,
		`Product.PlusU has "enum=+5|1"; write whole numbers separated by | after enum=.`,
		`Product.Lead0U has "enum=05|1"; write whole numbers separated by | after enum=.`,
		`Product.NegL0U has "enum=-05|1"; write whole numbers separated by | after enum=.`,
		`Product.Hex has "enum=0x1p4|2"; write numbers separated by | after enum=.`,
		`Product.Inex32 has "enum=1|16777217"; write numbers that a float32 can hold exactly separated by | after enum=.`,
		`Product.Score has "enum=1_0|2"; write whole numbers separated by | after enum=.`,
		`Product.Twice has the rule "enum" more than once; list every value in one enum=a|b.`,
		`Product.Tags has the rule "unique", which is for text and numbers; remove it.`,
		`Product.Dec has "min=1.5"; write a number after min=.`,
		`Product.Bare has "min"; write a number after min=.`,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got:\n%q\nwant:\n%q", got, want)
	}
}

func TestDateRule(t *testing.T) {
	type Event struct {
		Day   time.Time  `json:"day" warp:"date"`
		Maybe *time.Time `json:"maybe" warp:"date"`
		Due   time.Time  `json:"due" warp:"required,date"`
	}
	const alone = "Send a date alone, such as 2026-03-29T00:00:00Z."
	for _, c := range []struct {
		body string
		want map[string]string
	}{
		{`{"day":"2026-03-29T00:00:00Z","due":"2026-03-29T00:00:00Z"}`, nil},
		{`{"day":"2026-03-29T00:00:00+02:00","due":"2026-03-29T00:00:00Z"}`, map[string]string{"day": alone}},
		{`{"day":"2026-03-29T00:00:01Z","due":"2026-03-29T00:00:00Z"}`, map[string]string{"day": alone}},
		{`{"day":"2026-03-29T00:00:00.5Z","due":"2026-03-29T00:00:00Z"}`, map[string]string{"day": alone}},
		// Midnight UTC written in another zone is still not a date alone.
		{`{"day":"2026-03-29T02:00:00+02:00","due":"2026-03-29T00:00:00Z"}`, map[string]string{"day": alone}},
		// The zero time is empty, as for every rule.
		{`{"day":"0001-01-01T00:00:00Z","due":"2026-03-29T00:00:00Z"}`, nil},
		{`{"due":"2026-03-29T00:00:00Z"}`, nil},
		{`{"maybe":"2026-03-29T00:00:00Z","due":"2026-03-29T00:00:00Z"}`, nil},
		{`{"maybe":"2026-03-29T12:00:00Z","due":"2026-03-29T00:00:00Z"}`, map[string]string{"maybe": alone}},
		{`{"maybe":null,"due":"2026-03-29T00:00:00Z"}`, nil},
		{`{"day":"2026-03-29T00:00:00Z"}`, map[string]string{"due": "This is required."}},
		{`{"due":"2026-03-29T10:00:00Z"}`, map[string]string{"due": alone}},
	} {
		var e Event
		if err := json.Unmarshal([]byte(c.body), &e); err != nil {
			t.Fatal(err)
		}
		if got := fieldsOf(t, e); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: %v, want %v", c.body, got, c.want)
		}
	}
	if got := checkRules(reflect.TypeOf(Event{})); len(got) != 0 {
		t.Errorf("date on a time.Time and a *time.Time is a mistake: %q", got)
	}
}

func TestDateRuleMistake(t *testing.T) {
	type Product struct {
		Name  string      `json:"name" warp:"date"`
		Days  []time.Time `json:"days" warp:"date"`
		Count *int        `json:"count" warp:"required,date"`
	}
	got := checkRules(reflect.TypeOf(Product{}))
	want := []string{
		`Product.Name has the rule "date", which is for a time.Time; remove it or make the field a time.`,
		`Product.Days has the rule "date", which is for a time.Time; remove it or make the field a time.`,
		`Product.Count has the rule "date", which is for a time.Time; remove it or make the field a time.`,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got:\n%q\nwant:\n%q", got, want)
	}
	app := New(Config{})
	Post(app, "/products", func(context.Context, Product) (None, error) { return None{}, nil })
	if err := app.Err(); err == nil || !strings.Contains(err.Error(), "warp: POST /products: "+want[0]) {
		t.Errorf("Err = %v, want %q", err, want[0])
	}
}

func TestCheckRulesCorrect(t *testing.T) {
	type Good struct {
		Name  string   `json:"name" warp:"required,min=1,max=20,unique"`
		Age   float64  `json:"age" warp:"min=0.5,max=9.5,unique"`
		Email *string  `json:"email" warp:"email"`
		Tags  []string `json:"tags" warp:"min=1"`
		Free  string   `json:"free"`
		Skip  bool     `warp:"min=1"`
	}
	if got := checkRules(reflect.TypeOf(&Good{})); len(got) != 0 {
		t.Fatalf("got %v", got)
	}
	if got := checkRules(reflect.TypeOf(3)); got != nil {
		t.Fatalf("got %v", got)
	}
}

type node struct {
	Name     string  `json:"name" warp:"bogus"`
	Children []node  `json:"children"`
	Parent   *node   `json:"parent"`
	Peers    []*node `json:"peers"`
}

func TestCheckRulesSelfReferencing(t *testing.T) {
	got := checkRules(reflect.TypeOf(node{}))
	want := []string{`node.Name has the rule "bogus", which warp does not know; use required, email, url, min=, max=, enum=, date, unique or readonly.`}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q", got)
	}
	n := node{Children: []node{{Name: "x"}}}
	if err := Validate(n); err != nil {
		t.Fatal(err)
	}
}

func TestCheckRulesNestedAndEmbedded(t *testing.T) {
	type Base struct {
		ID string `json:"id" warp:"nope"`
	}
	type Line struct {
		SKU int `json:"sku" warp:"url"`
	}
	type Order struct {
		Base
		Lines []Line `json:"lines"`
		More  []Line `json:"more"`
	}
	got := checkRules(reflect.TypeOf(Order{}))
	want := []string{
		`Base.ID has the rule "nope", which warp does not know; use required, email, url, min=, max=, enum=, date, unique or readonly.`,
		`Line.SKU has the rule "url", which is for text; remove it or make the field a string.`,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q", got)
	}
}

func TestValidateRequiredPointer(t *testing.T) {
	type S struct {
		N *int     `json:"n" warp:"required"`
		B *bool    `json:"b" warp:"required"`
		S *string  `json:"s" warp:"required"`
		M *string  `json:"m" warp:"required,email"`
		F *float64 `json:"f" warp:"required,min=1"`
	}
	zero, off, empty, bad, half := 0, false, "", "nope", 0.5
	got := fieldsOf(t, S{N: &zero, B: &off, S: &empty, M: &bad, F: &half})
	want := map[string]string{"m": "Enter an email address.", "f": "Enter 1 or more."}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	got = fieldsOf(t, S{})
	if len(got) != 5 || got["n"] != "This is required." {
		t.Fatalf("a nil pointer must fail required: %v", got)
	}
}

func TestValidateBlankIsNotEmpty(t *testing.T) {
	type S struct {
		Mail string `json:"mail" warp:"email"`
		Kind string `json:"kind" warp:"enum=a|b"`
		Site string `json:"site" warp:"url"`
		Two  string `json:"two" warp:"min=2"`
		Five string `json:"five" warp:"min=5"`
	}
	got := fieldsOf(t, S{Mail: "   ", Kind: "   ", Site: "   ", Two: "   ", Five: "   "})
	want := map[string]string{
		"mail": "Enter an email address.",
		"kind": "Choose one of: a, b.",
		"site": "Enter a web address that starts with http:// or https://.",
		"five": "Use at least 5 characters.",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestCheckRulesDeclarationOrder(t *testing.T) {
	type Inner struct {
		X string `json:"x" warp:"inner"`
	}
	type Outer struct {
		A  Inner  `json:"a"`
		B  string `json:"b" warp:"outer"`
		Cs []Inner
	}
	got := checkRules(reflect.TypeOf(Outer{}))
	want := []string{
		`Inner.X has the rule "inner", which warp does not know; use required, email, url, min=, max=, enum=, date, unique or readonly.`,
		`Outer.B has the rule "outer", which warp does not know; use required, email, url, min=, max=, enum=, date, unique or readonly.`,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q", got)
	}
}

func TestValidateDescendsIntoMapValues(t *testing.T) {
	type Line struct {
		SKU string `json:"sku" warp:"required"`
	}
	type Order struct {
		ByName  map[string]Line         `json:"by_name"`
		ByPtr   map[string]*Line        `json:"by_ptr"`
		Lists   map[string][]Line       `json:"lists"`
		Nested  map[string]map[int]Line `json:"nested"`
		Nothing map[string]*Line        `json:"nothing"`
		Any     any                     `json:"any"`
		AnyPtr  any                     `json:"any_ptr"`
		AnyList any                     `json:"any_list"`
		Anys    map[string]any          `json:"anys"`
		Grid    [][]Line                `json:"grid"`
	}
	got := fieldsOf(t, Order{
		ByName:  map[string]Line{"x": {}, "y": {SKU: "ok"}},
		ByPtr:   map[string]*Line{"p": {}},
		Lists:   map[string][]Line{"l": {{SKU: "ok"}, {}}},
		Nested:  map[string]map[int]Line{"a": {7: {}}},
		Nothing: map[string]*Line{"n": nil},
		Any:     Line{},
		AnyPtr:  &Line{},
		AnyList: []Line{{}}, // an interface holding a list is a leaf
		Anys:    map[string]any{"q": &Line{}, "r": 3, "s": []any{Line{}}},
		Grid:    [][]Line{{{SKU: "ok"}}, {{}, {SKU: "ok"}}},
	})
	want := map[string]string{
		"by_name.x.sku":  "This is required.",
		"by_ptr.p.sku":   "This is required.",
		"lists.l.1.sku":  "This is required.",
		"nested.a.7.sku": "This is required.",
		"any.sku":        "This is required.",
		"any_ptr.sku":    "This is required.",
		"anys.q.sku":     "This is required.",
		"grid.1.0.sku":   "This is required.",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("fields %v, want %v", got, want)
	}
}

func TestRulesReachMapValues(t *testing.T) {
	type Line struct {
		SKU string `json:"sku" warp:"emal"`
	}
	type Order struct {
		ByName map[string][]*Line `json:"by_name"`
	}
	got := checkRules(reflect.TypeOf(Order{}))
	want := []string{`Line.SKU has the rule "emal", which warp does not know; use ` + ruleList + "."}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("checkRules = %q, want %q", got, want)
	}
}

func TestSelfReferenceThroughMap(t *testing.T) {
	type Tree struct {
		Name string          `json:"name" warp:"required,emal"`
		Kids map[string]Tree `json:"kids"`
	}
	got := checkRules(reflect.TypeOf(Tree{}))
	if want := []string{`Tree.Name has the rule "emal", which warp does not know; use ` + ruleList + "."}; !reflect.DeepEqual(got, want) {
		t.Fatalf("checkRules = %q, want %q", got, want)
	}
	type Good struct {
		Name string          `json:"name" warp:"required"`
		Kids map[string]Good `json:"kids"`
	}
	fields := fieldsOf(t, Good{Name: "root", Kids: map[string]Good{"a": {Name: "a", Kids: map[string]Good{"b": {}}}}})
	if want := map[string]string{"kids.a.kids.b.name": "This is required."}; !reflect.DeepEqual(fields, want) {
		t.Fatalf("fields %v, want %v", fields, want)
	}
}

// nestedLists is JSON of lists nested depth deep, each holding perLevel
// numbers.
func nestedLists(depth, perLevel int) []byte {
	var b bytes.Buffer
	for range depth {
		b.WriteByte('[')
		for j := range perLevel {
			fmt.Fprintf(&b, "%d,", j)
		}
	}
	b.WriteString("0")
	for range depth {
		b.WriteByte(']')
	}
	return b.Bytes()
}

func TestValidateDecodedAnyCostsNothing(t *testing.T) {
	type Doc struct {
		Name string `json:"name" warp:"required"`
		Data any    `json:"data"`
	}
	var doc Doc
	body := `{"name":"n","data":` + string(nestedLists(2000, 50)) + `}`
	if err := json.Unmarshal([]byte(body), &doc); err != nil {
		t.Fatal(err)
	}
	if err := Validate(doc); err != nil {
		t.Fatal(err)
	}
	if allocs := testing.AllocsPerRun(5, func() { Validate(&doc) }); allocs >= 50 {
		t.Fatalf("Validate on decoded lists 2000 deep: %.0f allocations a run, want fewer than 50", allocs)
	}
}

func TestValidateLongListsCostNothing(t *testing.T) {
	type Doc struct {
		Tags  []string           `json:"tags" warp:"max=200000"`
		Grid  [][]int            `json:"grid"`
		Score map[string]float64 `json:"score"`
		Any   []any              `json:"any"`
		Bag   map[string]any     `json:"bag"`
	}
	doc := Doc{Tags: make([]string, 100000), Grid: make([][]int, 100000), Score: map[string]float64{}, Any: make([]any, 100000), Bag: map[string]any{}}
	for i := range 100000 {
		doc.Tags[i] = "t"
		doc.Grid[i] = []int{i}
		doc.Any[i] = []any{"x", float64(i)}
		doc.Score[strconv.Itoa(i)] = float64(i)
		doc.Bag[strconv.Itoa(i)] = map[string]any{"k": "v"}
	}
	if allocs := testing.AllocsPerRun(5, func() { Validate(&doc) }); allocs >= 50 {
		t.Fatalf("Validate on lists of 100000: %.0f allocations a run, want fewer than 50", allocs)
	}
}

func BenchmarkValidateDecodedAny(b *testing.B) {
	var doc struct {
		Data any `json:"data"`
	}
	if err := json.Unmarshal(append(append([]byte(`{"data":`), nestedLists(2000, 50)...), '}'), &doc); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for b.Loop() {
		Validate(&doc)
	}
}

func BenchmarkValidateLongStrings(b *testing.B) {
	doc := struct {
		Tags []string `json:"tags"`
		Any  []any    `json:"any"`
	}{Tags: make([]string, 100000), Any: make([]any, 100000)}
	for i := range doc.Any {
		doc.Any[i] = "x"
	}
	b.ReportAllocs()
	for b.Loop() {
		Validate(&doc)
	}
}

type deepNode struct {
	Name string     `json:"name" warp:"max=10"`
	Kids []deepNode `json:"k"`
}

type bareNode struct {
	Kids []bareNode `json:"children_of_this_comment_thread"`
}

func deepNodes(depth int, bottom string) deepNode {
	n := deepNode{Name: bottom}
	for range depth {
		n = deepNode{Kids: []deepNode{n}}
	}
	return n
}

func bareNodes(depth int) bareNode {
	var n bareNode
	for range depth {
		n = bareNode{Kids: []bareNode{n}}
	}
	return n
}

func TestValidateDeepPassingCostsNothing(t *testing.T) {
	ok := deepNodes(4000, "short")
	if err := Validate(&ok); err != nil {
		t.Fatal(err)
	}
	if allocs := testing.AllocsPerRun(5, func() { Validate(&ok) }); allocs >= 50 {
		t.Fatalf("Validate on nodes 4000 deep: %.0f allocations a run, want fewer than 50", allocs)
	}
	bare := bareNodes(4000)
	if allocs := testing.AllocsPerRun(5, func() { Validate(&bare) }); allocs >= 50 {
		t.Fatalf("Validate on rule-less nodes 4000 deep: %.0f allocations a run, want fewer than 50", allocs)
	}
}

func TestValidateDeepFailureNamesTheWholePath(t *testing.T) {
	bad := deepNodes(250, "far too long")
	want := map[string]string{strings.Repeat("k.0.", 250) + "name": "Use at most 10 characters."}
	if got := fieldsOf(t, &bad); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %d fields, want the one at the bottom", len(got))
	}
	// 4000 deep, the key is over 1 KiB, so it is not listed.
	bad = deepNodes(4000, "far too long")
	if e := validationOf(t, &bad); e.Message != "More fields need attention than can be listed." || e.Fields != nil {
		t.Fatalf("message %q, %d fields", e.Message, len(e.Fields))
	}
	if allocs := testing.AllocsPerRun(5, func() { Validate(&bad) }); allocs >= 100 {
		t.Fatalf("Validate with one failure 4000 deep: %.0f allocations a run, want fewer than 100", allocs)
	}
}

func TestValidateFlatListCostsNothing(t *testing.T) {
	type Line struct {
		SKU string `json:"sku" warp:"required"`
	}
	doc := struct {
		Lines []Line `json:"lines"`
	}{Lines: make([]Line, 100000)}
	for i := range doc.Lines {
		doc.Lines[i].SKU = "a"
	}
	if allocs := testing.AllocsPerRun(5, func() { Validate(&doc) }); allocs >= 50 {
		t.Fatalf("Validate on 100000 valid lines: %.0f allocations a run, want fewer than 50", allocs)
	}
}

func BenchmarkValidateDeepNodes(b *testing.B) {
	short := deepNodes(4990, "")
	long := bareNodes(4990)
	b.Run("k", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			Validate(&short)
		}
	})
	b.Run("children_of_this_comment_thread", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			Validate(&long)
		}
	})
}

type limitLine struct {
	SKU string `json:"sku" warp:"required"`
}

type limitOrder struct {
	Lines []limitLine `json:"lines"`
}

type limitIn struct {
	ByName map[string]limitOrder `json:"by_name"`
	Lines  []limitLine           `json:"lines"`
}

// validation answers that list fields, by what Validate returned.
func validationOf(t *testing.T, v any) *Error {
	t.Helper()
	var e *Error
	if !errors.As(Validate(v), &e) {
		t.Fatalf("Validate(%T) did not return an *Error", v)
	}
	return e
}

func failingLines(n int) []limitLine { return make([]limitLine, n) }

func TestValidateListsAtMost100Fields(t *testing.T) {
	e := validationOf(t, limitIn{Lines: failingLines(150)})
	if want := "More fields need attention than are listed; the first 100 are listed."; e.Message != want {
		t.Fatalf("message %q, want %q", e.Message, want)
	}
	want := map[string]string{}
	for i := range 100 {
		want["lines."+strconv.Itoa(i)+".sku"] = "This is required."
	}
	if !reflect.DeepEqual(e.Fields, want) {
		t.Fatalf("%d fields listed, want lines.0.sku to lines.99.sku", len(e.Fields))
	}
	// Exactly 100 is not more than are listed.
	if e := validationOf(t, limitIn{Lines: failingLines(100)}); e.Message != "100 fields need attention." || len(e.Fields) != 100 {
		t.Fatalf("100 failing: message %q, %d fields", e.Message, len(e.Fields))
	}
}

func TestValidateListsTheSameMapEntries(t *testing.T) {
	in := limitIn{ByName: map[string]limitOrder{}}
	for i := range 300 {
		in.ByName[fmt.Sprintf("k%03d", i)] = limitOrder{Lines: failingLines(1)}
	}
	first := validationOf(t, in).Fields
	if len(first) != 100 {
		t.Fatalf("%d fields listed, want 100", len(first))
	}
	for i := range 100 {
		if k := fmt.Sprintf("by_name.k%03d.lines.0.sku", i); first[k] == "" {
			t.Fatalf("%s is not listed; the first 100 keys in sorted order are", k)
		}
	}
	for range 20 {
		if got := validationOf(t, in).Fields; !reflect.DeepEqual(got, first) {
			t.Fatal("another run listed other fields")
		}
	}
}

func TestValidateKeyBudget(t *testing.T) {
	// A key over 1 KiB is never listed.
	long := limitIn{ByName: map[string]limitOrder{strings.Repeat("k", 1<<10): {Lines: failingLines(3)}}}
	e := validationOf(t, long)
	if e.Message != "More fields need attention than can be listed." || e.Fields != nil {
		t.Fatalf("message %q, %d fields", e.Message, len(e.Fields))
	}
	w := writeErr(t, e)
	if strings.Contains(w.Body.String(), `"fields"`) || w.Code != 422 {
		t.Fatalf("status %d, body %s", w.Code, w.Body)
	}
	// Up to 16 KiB of keys in all: by_name.<1000 bytes>.lines.0.sku is 1018
	// bytes, so 16 fit and the 17th does not.
	in := limitIn{ByName: map[string]limitOrder{}}
	for i := range 20 {
		in.ByName[fmt.Sprintf("%02d", i)+strings.Repeat("k", 998)] = limitOrder{Lines: failingLines(1)}
	}
	e = validationOf(t, in)
	if e.Message != "More fields need attention than are listed; the first 16 are listed." || len(e.Fields) != 16 {
		t.Fatalf("message %q, %d fields", e.Message, len(e.Fields))
	}
	// A key of exactly 1 KiB is listed, and a short one after a long one is too.
	edge := limitIn{ByName: map[string]limitOrder{
		strings.Repeat("a", 1024-len("by_name..lines.0.sku")): {Lines: failingLines(1)},
		"b" + strings.Repeat("z", 1024):                       {Lines: failingLines(1)},
		"c":                                                   {Lines: failingLines(1)},
	}}
	e = validationOf(t, edge)
	if e.Message != "More fields need attention than are listed; the first 2 are listed." || len(e.Fields) != 2 || e.Fields["by_name.c.lines.0.sku"] == "" {
		t.Fatalf("message %q, fields %d", e.Message, len(e.Fields))
	}
}

func TestValidateTwoFieldsUnchanged(t *testing.T) {
	type S struct {
		A string `json:"a" warp:"required"`
		B string `json:"b" warp:"email"`
	}
	e := validationOf(t, S{B: "nope"})
	if want := map[string]string{"a": "This is required.", "b": "Enter an email address."}; e.Message != "2 fields need attention." || !reflect.DeepEqual(e.Fields, want) {
		t.Fatalf("message %q, fields %v", e.Message, e.Fields)
	}
}

// The two bodies the whole-branch review measured: a small body could make a
// huge 422 (100 MB from the first, 11.6 MB from the second).
func TestValidationAnswersAreBounded(t *testing.T) {
	app := New(Config{})
	Post(app, "/orders", func(ctx context.Context, in limitIn) (Item, error) { return Item{}, nil })
	h := mustRegister(t, app)
	lines := func(n int) string { return strings.TrimSuffix(strings.Repeat("{},", n), ",") }
	for _, c := range []struct {
		name, body string
		allocs     float64
	}{
		{"one 100 KB map key over 1,000 failing lines", `{"by_name":{"` + strings.Repeat("k", 100<<10) + `":{"lines":[` + lines(1000) + `]}}}`, 50},
		{"300,000 failing lines", `{"lines":[` + lines(300000) + `]}`, 200},
	} {
		w := send(t, h, "POST", "/api/orders", c.body)
		if w.Code != 422 || w.Body.Len() >= 32<<10 {
			t.Errorf("%s: status %d, answer %d B, want a 422 under 32 KiB", c.name, w.Code, w.Body.Len())
		}
		var in limitIn
		if err := json.Unmarshal([]byte(c.body), &in); err != nil {
			t.Fatal(err)
		}
		allocs := testing.AllocsPerRun(3, func() { Validate(&in) })
		t.Logf("%s: answer %d B, %.0f allocations", c.name, w.Body.Len(), allocs)
		if allocs > c.allocs {
			t.Errorf("%s: Validate made %.0f allocations, want at most %.0f", c.name, allocs, c.allocs)
		}
	}
}

func TestValidateValidMapCostsLittle(t *testing.T) {
	doc := struct {
		ByName map[string]limitLine `json:"by_name"`
	}{ByName: map[string]limitLine{}}
	for i := range 100000 {
		doc.ByName[strconv.Itoa(i)] = limitLine{SKU: "a"}
	}
	allocs := testing.AllocsPerRun(5, func() { Validate(&doc) })
	t.Logf("a map of 100000 valid entries: %.0f allocations", allocs)
	if allocs >= 50 {
		t.Fatalf("Validate on a map of 100000 valid entries: %.0f allocations a run, want fewer than 50", allocs)
	}
}

// A pointer that is not nil was sent, so every rule applies to what it points
// at, even a zero; a zero that is not behind a pointer is empty, and only
// required checks it.
func TestValidateSentPointerZero(t *testing.T) {
	type S struct {
		N    *int    `json:"n" warp:"min=1"`
		Text *string `json:"text" warp:"min=1"`
		Mail *string `json:"mail" warp:"email"`
		Kind *string `json:"kind" warp:"enum=a|b"`
		Need *int    `json:"need" warp:"required"`
		Zero int     `json:"zero" warp:"min=1"`
		None *int    `json:"none" warp:"min=1"`
	}
	zero, empty := 0, ""
	got := fieldsOf(t, S{N: &zero, Text: &empty, Mail: &empty, Kind: &empty, Need: &zero})
	want := map[string]string{
		"n":    "Enter 1 or more.",
		"text": "Use at least 1 character.",
		"mail": "Enter an email address.",
		"kind": "Choose one of: a, b.",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("fields %v, want %v", got, want)
	}
}

func TestValidateRequiredOnEmptyCollections(t *testing.T) {
	type S struct {
		List  []int          `json:"list" warp:"required"`
		Map   map[string]int `json:"map" warp:"required"`
		Array [2]string      `json:"array" warp:"required"`
	}
	want := map[string]string{"list": "This is required.", "map": "This is required.", "array": "This is required."}
	if got := fieldsOf(t, S{List: []int{}, Map: map[string]int{}}); !reflect.DeepEqual(got, want) {
		t.Fatalf("fields %v, want %v", got, want)
	}
	if got := fieldsOf(t, S{List: []int{0}, Map: map[string]int{"a": 0}, Array: [2]string{"", "x"}}); got != nil {
		t.Fatalf("fields %v, want none", got)
	}
}

func TestValidateMinMaxOnArrays(t *testing.T) {
	type S struct {
		Few  [2]int `json:"few" warp:"min=3"`
		Many [4]int `json:"many" warp:"max=3"`
	}
	want := map[string]string{"few": "Choose at least 3.", "many": "Choose at most 3."}
	if got := fieldsOf(t, S{Few: [2]int{1}, Many: [4]int{1}}); !reflect.DeepEqual(got, want) {
		t.Fatalf("fields %v, want %v", got, want)
	}
	if got := fieldsOf(t, S{}); got != nil {
		t.Fatalf("a zero array is empty: fields %v", got)
	}
}

func TestKeyOfNoSteps(t *testing.T) {
	if got := key(nil); got != "" {
		t.Fatalf("key(nil) = %q", got)
	}
}

func TestReadonlyIsNeverValidated(t *testing.T) {
	type Person struct {
		Name string `json:"name" warp:"required"`
	}
	type S struct {
		ID    int64    `json:"id" warp:"readonly"`
		Owner Person   `json:"owner" warp:" readonly "`
		Peers []Person `json:"peers" warp:"readonly"`
		Name  string   `json:"name" warp:"required"`
	}
	if got := fieldsOf(t, S{Peers: []Person{{}}}); !reflect.DeepEqual(got, map[string]string{"name": "This is required."}) {
		t.Fatalf("fields %v", got)
	}
	var readonly []string
	for _, spec := range specsOf(reflect.TypeFor[S]()) {
		if spec.readonly {
			readonly = append(readonly, spec.key)
		}
	}
	if want := []string{"id", "owner", "peers"}; !reflect.DeepEqual(readonly, want) {
		t.Fatalf("readonly specs %q, want %q", readonly, want)
	}
}

func TestCheckRulesReadonly(t *testing.T) {
	type Product struct {
		ID    int64  `json:"id" warp:"readonly,required"`
		Code  string `json:"code" warp:"min=1,readonly"`
		Fine  int64  `json:"fine" warp:"readonly"`
		Twice int64  `json:"twice" warp:"readonly,readonly"`
	}
	got := checkRules(reflect.TypeFor[Product]())
	want := []string{
		`Product.ID has the rule "readonly" beside other rules; a readonly field is never read, so remove them.`,
		`Product.Code has the rule "readonly" beside other rules; a readonly field is never read, so remove them.`,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got:\n%q\nwant:\n%q", got, want)
	}
}

// The url rule is the browser's (isWebAddress in packages/ui's
// forms/schema.ts), on the rows of its test.
func TestURLRuleIsTheBrowsers(t *testing.T) {
	type U struct {
		Site string `json:"site" warp:"url"`
	}
	for _, ok := range []string{
		"http://x.com", "HTTPS://X.COM/a?b#c", "http://u:p@x.com", "http://x.com:8080", "http://[::1]:8080/",
		"https://u@p@x.com", "http://x.com?q", "http://x.com#f", "http://1.2.3.4:1",
	} {
		if err := Validate(&U{Site: ok}); err != nil {
			t.Errorf("%q was refused: %v", ok, err)
		}
	}
	for _, bad := range []string{
		"ftp://x", "http://", "//x.com", "http:x.com", "http://a b.com", "http://x.com/a b", "http://:8080",
		"https:///x.com", "http://x.com ", "http://x\t.com", "http://x.com/\na", `http://x.com\a`, "http://x:999999",
		"http://x:", "http://x:8a", "http://u@", "http://x.com\u00a0", "http://x.com\ufeff", "\x00http://x.com", "http://x\x7f.com",
	} {
		if fieldsOf(t, &U{Site: bad})["site"] != "Enter a web address that starts with http:// or https://." {
			t.Errorf("%q was taken", bad)
		}
	}
}

// The email rule is the browser's (EMAIL in packages/ui's forms/schema.ts),
// on the rows both sides' tests read from testdata/email.json.
func TestEmailRuleIsTheBrowsers(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("testdata", "email.json"))
	if err != nil {
		t.Fatal(err)
	}
	var rows struct{ Accept, Refuse []string }
	if err := json.Unmarshal(b, &rows); err != nil {
		t.Fatal(err)
	}
	type E struct {
		Mail string `json:"mail" warp:"email"`
	}
	for _, ok := range rows.Accept {
		if err := Validate(&E{Mail: ok}); err != nil {
			t.Errorf("%q was refused: %v", ok, err)
		}
	}
	for _, bad := range rows.Refuse {
		if fieldsOf(t, &E{Mail: bad})["mail"] != "Enter an email address." {
			t.Errorf("%q was taken", bad)
		}
	}
}

// A count of one character is said in the singular, as the browser's schema
// says it.
func TestValidateSaysOneCharacter(t *testing.T) {
	type One struct {
		A string `json:"a" warp:"min=1"`
		B string `json:"b" warp:"max=1"`
	}
	type Two struct {
		A string `json:"a" warp:"min=2"`
		B string `json:"b" warp:"max=2"`
	}
	if got := fieldsOf(t, One{A: " ", B: "xy"}); !reflect.DeepEqual(got, map[string]string{"b": "Use at most 1 character."}) {
		t.Errorf("one: %v", got)
	}
	if got := fieldsOf(t, &struct {
		A string `json:"a" warp:"min=1"`
	}{A: ""}); got != nil {
		t.Errorf("empty passes min: %v", got)
	}
	if got := fieldsOf(t, Two{A: "x", B: "xyz"}); !reflect.DeepEqual(got, map[string]string{"a": "Use at least 2 characters.", "b": "Use at most 2 characters."}) {
		t.Errorf("two: %v", got)
	}
	type OneMin struct {
		A string `json:"a" warp:"min=2,max=1"`
	}
	if got := fieldsOf(t, OneMin{A: "x"}); got["a"] != "Use at least 2 characters." {
		t.Errorf("min first: %v", got)
	}
}
