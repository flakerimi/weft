package warp

import (
	"errors"
	"net/http"
	"reflect"
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
		E{Email: "ada@example.com"}, E{Email: "<ada@example.com>"}, U{Site: "https://example.com/x"},
		En{Status: "live"}, B{On: true}, T{At: time.Now()},
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
		Kind   int     `json:"kind" warp:"enum=a|b"`
		Tags   []int   `json:"tags" warp:"unique"`
		Dec    string  `json:"dec" warp:"min=1.5"`
		Bare   string  `json:"bare" warp:"min"`
	}
	got := checkRules(reflect.TypeOf(Product{}))
	want := []string{
		`Product.Price has the rule "emial", which warp does not know; use required, email, url, min=, max=, enum= or unique.`,
		`Product.Mail has the rule "email", which is for text; remove it or make the field a string.`,
		`Product.Name has "min=abc"; write a number after min=.`,
		`Product.Name has "max=-1"; write a number after max=.`,
		`Product.Status has "enum=" with nothing to choose from; write enum=a|b.`,
		`Product.Flag has the rule "min", which is for text, numbers and lists; remove it.`,
		`Product.Site has the rule "url", which is for text; remove it or make the field a string.`,
		`Product.Kind has the rule "enum", which is for text; remove it or make the field a string.`,
		`Product.Tags has the rule "unique", which is for text and numbers; remove it.`,
		`Product.Dec has "min=1.5"; write a number after min=.`,
		`Product.Bare has "min"; write a number after min=.`,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got:\n%q\nwant:\n%q", got, want)
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
	want := []string{`node.Name has the rule "bogus", which warp does not know; use required, email, url, min=, max=, enum= or unique.`}
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
		`Base.ID has the rule "nope", which warp does not know; use required, email, url, min=, max=, enum= or unique.`,
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
		`Inner.X has the rule "inner", which warp does not know; use required, email, url, min=, max=, enum= or unique.`,
		`Outer.B has the rule "outer", which warp does not know; use required, email, url, min=, max=, enum= or unique.`,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q", got)
	}
}
