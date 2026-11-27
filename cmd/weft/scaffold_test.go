package main

import (
	"bytes"
	"go/format"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

func scaffold(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	var out, errb bytes.Buffer
	code := run(append([]string{"scaffold"}, args...), &out, &errb)
	return code, out.String(), errb.String()
}

// The pluralizer is a table and four rules, the same every time; the last
// word of a snake_case name is the one made plural.
func TestPluralOf(t *testing.T) {
	for in, want := range map[string]string{
		"product": "products", "person": "people", "child": "children", "category": "categories",
		"status": "statuses", "address": "addresses", "box": "boxes", "bus": "buses", "quiz": "quizzes",
		"leaf": "leaves", "knife": "knives", "life": "lives", "series": "series", "news": "news",
		"equipment": "equipment", "day": "days", "key": "keys", "story": "stories", "church": "churches",
		"wish": "wishes", "buzz": "buzzes", "line_item": "line_items", "sales_person": "sales_people",
		"order_category": "order_categories", "note": "notes", "y": "ys",
	} {
		if got := pluralOf(in); got != want {
			t.Errorf("pluralOf(%q) = %q, want %q", in, got, want)
		}
	}
}

// The spec's example: each field's Go type and its three tags, the data tag
// from the type alone.
func TestScaffoldParsesFields(t *testing.T) {
	s, err := parseScaffold([]string{"product", "name:string:required:max=120:unique", "price:float:min=0", "sale:float:nullable", "status:enum=draft,live:required"})
	if err != nil {
		t.Fatalf("parseScaffold: %v", err.msg)
	}
	if s.Name != "product" || s.Plural != "products" || s.API {
		t.Errorf("spec = %+v", s)
	}
	want := []struct{ goType, tag string }{
		{"string", `json:"name" warp:"required,max=120,unique" data:"sort,search"`},
		{"float64", `json:"price" warp:"min=0" data:"sort"`},
		{"*float64", `json:"sale" data:"sort"`},
		{"string", `json:"status" warp:"required,enum=draft|live" data:"sort,filter"`},
	}
	if len(s.Fields) != len(want) {
		t.Fatalf("fields = %+v", s.Fields)
	}
	for i, f := range s.Fields {
		if f.goType() != want[i].goType || f.tag() != want[i].tag {
			t.Errorf("field %s: %s `%s`, want %s `%s`", f.Name, f.goType(), f.tag(), want[i].goType, want[i].tag)
		}
	}
}

// Every type, nullable or not, and a field with no type, which is a string.
func TestScaffoldTypes(t *testing.T) {
	for arg, want := range map[string]struct{ goType, tag string }{
		"title":                       {"string", `json:"title" data:"sort,search"`},
		"title:string:nullable":       {"*string", `json:"title" data:"sort,search"`},
		"body:text":                   {"string", `json:"body" data:"search"`},
		"body:text:nullable:max=5000": {"*string", `json:"body" warp:"max=5000" data:"search"`},
		"body:text:max=5000:nullable": {"*string", `json:"body" warp:"max=5000" data:"search"`},
		"mail:email:required":         {"string", `json:"mail" warp:"required,email" data:"sort,search"`},
		"site:url:nullable":           {"*string", `json:"site" warp:"url" data:"sort,search"`},
		"count:int:min=-3":            {"int64", `json:"count" warp:"min=-3" data:"sort"`},
		"count:int:nullable":          {"*int64", `json:"count" data:"sort"`},
		"score:float:max=9.5":         {"float64", `json:"score" warp:"max=9.5" data:"sort"`},
		"live:bool":                   {"bool", `json:"live" data:"sort,filter"`},
		"live:bool:nullable":          {"*bool", `json:"live" data:"sort,filter"`},
		"on:date":                     {"time.Time", `json:"on" data:"sort"`},
		"at:datetime:nullable":        {"*time.Time", `json:"at" data:"sort"`},
		"kind:enum=a,b,c":             {"string", `json:"kind" warp:"enum=a|b|c" data:"sort,filter"`},
		"kind:enum=a,b:nullable":      {"*string", `json:"kind" warp:"enum=a|b" data:"sort,filter"`},
		"code:string:unique":          {"string", `json:"code" warp:"unique" data:"sort,search"`},
		"n:int:unique:min=1":          {"int64", `json:"n" warp:"min=1,unique" data:"sort"`},
	} {
		s, err := parseScaffold([]string{"thing", arg})
		if err != nil {
			t.Errorf("%s: %s", arg, err.msg)
			continue
		}
		if f := s.Fields[0]; f.goType() != want.goType || f.tag() != want.tag {
			t.Errorf("%s: %s `%s`, want %s `%s`", arg, f.goType(), f.tag(), want.goType, want.tag)
		}
	}
}

// Every refusal is one sentence naming what is wrong, with the code, and is
// said before anything is looked at: here no app is anywhere.
func TestScaffoldRefuses(t *testing.T) {
	t.Chdir(t.TempDir())
	usage := "usage is weft scaffold <name> <field[:type[:modifier...]]>... [--plural <word>] [--api]."
	types := "use string, text, email, url, int, float, bool, date, datetime or enum=a,b"
	for _, c := range []struct {
		args []string
		code int
		want string
	}{
		{nil, 2, "weft scaffold: give a resource name and at least one field; " + usage},
		{[]string{"product"}, 2, "weft scaffold: give at least one field after product, such as name:string; " + usage},
		{[]string{"Product", "name"}, 2, `weft scaffold: "Product" is not a resource name; use a lowercase letter, then lowercase letters, digits and _, such as line_item.`},
		{[]string{"product", "name", "--plural", "Goods"}, 2, `weft scaffold: --plural "Goods" is not a name; use a lowercase letter, then lowercase letters, digits and _, such as line_items.`},
		{[]string{"data", "name"}, 2, `weft scaffold: "data" is a name Weft uses in an app (data, auth, warp, app, web, test, main); choose another.`},
		{[]string{"thing", "name", "--plural", "main"}, 2, `weft scaffold: "main" is a name Weft uses in an app (data, auth, warp, app, web, test, main); choose another.`},
		{[]string{"thing", "name", "--plural", "type"}, 2, `weft scaffold: "type" is a word Go keeps for itself, so it cannot name the package app/type; give another with --plural.`},
		{[]string{"user", "name"}, 2, `weft scaffold: "user" would make the type User, which Weft's own modules or the generated client use already; choose another name.`},
		{[]string{"thing", "name", "--plural", "users"}, 2, `weft scaffold: "users" is the table auth keeps its users in; give another with --plural.`},
		{[]string{"product", "status:enum(draft|live)"}, 2, "weft scaffold: write enum choices separated by commas after enum=, as in status:enum=draft,live."},
		{[]string{"product", "status:enum=draft|live"}, 2, "weft scaffold: write enum choices separated by commas after enum=, as in status:enum=draft,live."},
		{[]string{"product", "Name"}, 2, `weft scaffold: "Name" is not a field name; use a lowercase letter, then lowercase letters, digits and _, such as published_at.`},
		{[]string{"product", "id:int"}, 2, `weft scaffold: field "id" comes from data.Record, which every model embeds; leave it out.`},
		{[]string{"product", "created_at:datetime"}, 2, `weft scaffold: field "created_at" comes from data.Record, which every model embeds; leave it out.`},
		{[]string{"product", "updated_at"}, 2, `weft scaffold: field "updated_at" comes from data.Record, which every model embeds; leave it out.`},
		{[]string{"product", "name", "price:float", "name:text"}, 2, `weft scaffold: field "name" is given twice; give each field once.`},
		{[]string{"product", "site_url", "site_u_r_l"}, 2, `weft scaffold: fields "site_url" and "site_u_r_l" would both be the Go field SiteURL; rename one.`},
		{[]string{"product", "actions:string"}, 2, `weft scaffold: field "actions" is the key DataTable gives its column of row actions; rename it.`},
		{[]string{"product", "constructor"}, 2, `weft scaffold: field "constructor" has the name of a property every JavaScript object has, which the generated client cannot check; rename it.`},
		{[]string{"product", "price:money"}, 2, `weft scaffold: field "price" has the type "money", which weft scaffold does not know; ` + types + "."},
		{[]string{"product", "price:"}, 2, `weft scaffold: field "price" has the type "", which weft scaffold does not know; ` + types + "."},
		{[]string{"product", "price:float:big"}, 2, `weft scaffold: field "price" has the modifier "big"; use required, nullable, unique, min=N or max=N.`},
		{[]string{"product", "price:float:required:required"}, 2, `weft scaffold: field "price" has the modifier "required" twice; give it once.`},
		{[]string{"product", "live:bool:required"}, 2, `weft scaffold: field "live" is a bool, which is always true or false, so it cannot be required; remove :required.`},
		{[]string{"product", "sale:float:nullable:required"}, 2, `weft scaffold: field "sale" is nullable, which may be empty, so it cannot be required; remove :nullable or :required.`},
		{[]string{"product", "sale:float:required:nullable"}, 2, `weft scaffold: field "sale" is nullable, which may be empty, so it cannot be required; remove :nullable or :required.`},
		{[]string{"product", "live:bool:nullable:required"}, 2, `weft scaffold: field "live" is a bool, which is always true or false, so it cannot be required; remove :required.`},
		{[]string{"product", "sale:float:nullable:nullable"}, 2, `weft scaffold: field "sale" has the modifier "nullable" twice; give it once.`},
		// ? is a pattern in zsh: one form, a modifier.
		{[]string{"product", "sale:float?"}, 2, "weft scaffold: write a field that may be null with the modifier nullable, as in sale:float:nullable."},
		{[]string{"product", "sale?:float"}, 2, "weft scaffold: write a field that may be null with the modifier nullable, as in sale:float:nullable."},
		{[]string{"product", "kind:enum=a,b?"}, 2, "weft scaffold: write a field that may be null with the modifier nullable, as in sale:float:nullable."},
		{[]string{"product", "live:bool:unique"}, 2, `weft scaffold: field "live" is a bool, and unique is for string, email, url, enum, int and float; remove :unique.`},
		{[]string{"product", "body:text:unique"}, 2, `weft scaffold: field "body" is a text, and unique is for string, email, url, enum, int and float; remove :unique.`},
		{[]string{"product", "on:date:unique"}, 2, `weft scaffold: field "on" is a date, and unique is for string, email, url, enum, int and float; remove :unique.`},
		{[]string{"product", "at:datetime:unique"}, 2, `weft scaffold: field "at" is a datetime, and unique is for string, email, url, enum, int and float; remove :unique.`},
		{[]string{"product", "live:bool:min=1"}, 2, `weft scaffold: field "live" is a bool, and min= is for string, text, email, url, int and float; remove :min=1.`},
		{[]string{"product", "on:date:max=3"}, 2, `weft scaffold: field "on" is a date, and max= is for string, text, email, url, int and float; remove :max=3.`},
		{[]string{"product", "kind:enum=a,b:min=1"}, 2, `weft scaffold: field "kind" is an enum, and min= is for string, text, email, url, int and float; remove :min=1.`},
		{[]string{"product", "name:string:max=-1"}, 2, `weft scaffold: field "name" has max=-1; for text, write a whole number of characters, 0 or more, as in max=120.`},
		{[]string{"product", "name:string:max=1.5"}, 2, `weft scaffold: field "name" has max=1.5; for text, write a whole number of characters, 0 or more, as in max=120.`},
		{[]string{"product", "name:string:max=0"}, 2, `weft scaffold: field "name" has max=0, which holds no text; write max=1 or more.`},
		{[]string{"product", "count:int:min=1.5"}, 2, `weft scaffold: field "count" has min=1.5; for an int, write a whole number, as in min=0.`},
		{[]string{"product", "price:float:min=cheap"}, 2, `weft scaffold: field "price" has min=cheap; for a float, write a number, as in min=0.5.`},
		{[]string{"product", "price:float:min=5:max=1"}, 2, `weft scaffold: field "price" has min=5 above max=1; make min the smaller.`},
		{[]string{"product", "n:int:unique:min=5:max=5"}, 2, `weft scaffold: field "n" is unique between min=5 and max=5, which leaves too few values for two records; widen them.`},
		{[]string{"product", "mail:email:max=2"}, 2, `weft scaffold: field "mail" is an email with max=2, which no address fits; raise max.`},
		{[]string{"product", "kind:enum="}, 2, `weft scaffold: field "kind" has enum= with no choices; write them separated by commas, as in kind:enum=draft,live.`},
		{[]string{"product", "kind:enum=a,,b"}, 2, `weft scaffold: field "kind" has an empty choice in enum=a,,b; write them separated by single commas, as in kind:enum=draft,live.`},
		{[]string{"product", "kind:enum=a,b,a"}, 2, `weft scaffold: field "kind" has the choice "a" twice; give each choice once.`},
		{[]string{"product", "kind:enum=a b,c"}, 2, `weft scaffold: field "kind" has the choice "a b"; write each choice with letters, digits, _, - and . only.`},
		{[]string{"product", "name", "--nope"}, 2, "weft scaffold: flag provided but not defined: -nope; " + usage},
		{[]string{"product", "name", "--plural"}, 2, "weft scaffold: flag needs an argument: -plural; " + usage},
		// Types of later milestones: exit 1, and the milestone named.
		{[]string{"product", "maker:belongs_to"}, 1, `weft scaffold: field "maker" has type "belongs_to", and relations land in Weft 0.3.0 (W3); use a scalar type for now.`},
		{[]string{"product", "parts:has_many"}, 1, `weft scaffold: field "parts" has type "has_many", and relations land in Weft 0.3.0 (W3); use a scalar type for now.`},
		{[]string{"product", "tags:many_to_many"}, 1, `weft scaffold: field "tags" has type "many_to_many", and relations land in Weft 0.3.0 (W3); use a scalar type for now.`},
		{[]string{"product", "photo:image"}, 1, `weft scaffold: field "photo" has type "image", and file attachments land in Weft 0.4.0 (W6); use a scalar type for now.`},
		{[]string{"product", "manual:file:nullable"}, 1, `weft scaffold: field "manual" has type "file", and file attachments land in Weft 0.4.0 (W6); use a scalar type for now.`},
		{[]string{"product", "extra:json"}, 1, `weft scaffold: field "extra" has type "json", which lands in Weft 0.6.0 (W7); use a scalar type for now.`},
		{[]string{"product", "title:translated"}, 1, `weft scaffold: field "title" has type "translated", which lands in Weft 0.6.0 (W7); use a scalar type for now.`},
	} {
		code, out, errs := scaffold(t, c.args...)
		if code != c.code || errs != c.want+"\n" || out != "" {
			t.Errorf("weft scaffold %q: exit %d, stdout %q, stderr %q; want exit %d, %q", c.args, code, out, errs, c.code, c.want)
		}
	}
	if entries, _ := os.ReadDir("."); len(entries) != 0 {
		t.Errorf("a refused weft scaffold wrote %v", entries)
	}
}

func TestScaffoldHelp(t *testing.T) {
	code, out, errs := scaffold(t, "--help")
	if code != 0 || errs != "" || !strings.HasPrefix(out, "Usage is weft scaffold <name>") {
		t.Errorf("exit %d, stdout %q, stderr %q", code, out, errs)
	}
}

// --plural is taken as it is, and --api anywhere among the fields.
func TestScaffoldFlags(t *testing.T) {
	s, err := parseScaffold([]string{"--api", "person", "name", "--plural", "staff", "age:int"})
	if err != nil {
		t.Fatal(err.msg)
	}
	if s.Name != "person" || s.Plural != "staff" || !s.API || len(s.Fields) != 2 || s.Fields[1].Name != "age" {
		t.Errorf("spec = %+v", s)
	}
	if s, _ := parseScaffold([]string{"person", "name"}); s.Plural != "people" {
		t.Errorf("plural of person = %q", s.Plural)
	}
}

// templateModules is app/modules.go as weft create writes it for myapp.
const templateModules = `// Package app lists the modules of the app.
package app

import (
	"github.com/flakerimi/weft/warp"
	"github.com/flakerimi/weft/warp/auth"
	"github.com/flakerimi/weft/warp/data"

	"myapp/app/hello"
)

// Modules is the one list of the app's modules, in the order they start: the
// database first, since the login keeps its users in it. weft scaffold adds
// each resource it makes at the end.
func Modules() []warp.Module {
	return []warp.Module{
		data.Module(),
		auth.Module(),
		hello.Module(), // the public route
	}
}
`

// fakeApp writes the files weft scaffold reads into a new folder: a
// package.json with a "weft" object, go.mod, app/modules.go and, with
// session, web/session.ts. It returns the folder.
func fakeApp(t *testing.T, modules string, session bool) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "myapp")
	files := map[string]string{
		"package.json":        `{"name": "myapp", "weft": {"version": "0.0.0-dev", "theme": "neutral"}}` + "\n",
		"go.mod":              "module myapp\n\ngo 1.26.0\n",
		"app/modules.go":      modules,
		"web/pages/index.vue": "<template><p>hi</p></template>\n",
	}
	if session {
		files["web/session.ts"] = "export {}\n"
	}
	for name, text := range files {
		writeFile(t, dir, name, text)
	}
	return dir
}

// tree is every file under dir with its bytes, and every folder.
func tree(t *testing.T, dir string) map[string]string {
	t.Helper()
	out := map[string]string{}
	filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			t.Fatal(err)
		}
		rel, _ := filepath.Rel(dir, p)
		if d.IsDir() {
			out[rel+"/"] = ""
			return nil
		}
		b, _ := os.ReadFile(p)
		out[rel] = string(b)
		return nil
	})
	return out
}

// Before writing anything weft scaffold checks every path it would write and
// app/modules.go; a refusal leaves the app's files exactly as they were.
func TestScaffoldPreflight(t *testing.T) {
	product := []string{"product", "name:string:required", "price:float"}
	for _, c := range []struct {
		name    string
		modules string
		make    string // a file that is there already
		args    []string
		want    string
	}{
		{"the package is there", templateModules, "app/products/old.go", nil, "weft scaffold: app/products already exists; remove it by hand or choose another resource name."},
		{"the pages are there", templateModules, "web/pages/products/index.vue", nil, "weft scaffold: web/pages/products already exists; move it away, choose another resource name, or give --api to write only the Go."},
		{"the table is there", templateModules, "web/components/ProductsTable.vue", nil, "weft scaffold: web/components/ProductsTable.vue already exists; move it away, choose another resource name, or give --api to write only the Go."},
		{"the view is there", templateModules, "web/components/ProductView.vue", nil, "weft scaffold: web/components/ProductView.vue already exists; move it away, choose another resource name, or give --api to write only the Go."},
		{"no Modules", "package app\n", "", nil, "weft scaffold: app/modules.go has no func Modules() []warp.Module returning a []warp.Module{...} list; write the modules as weft create does, then run weft scaffold again."},
		{"Modules built another way", "package app\n\nimport \"github.com/flakerimi/weft/warp\"\n\nfunc Modules() []warp.Module {\n\tvar m []warp.Module\n\treturn m\n}\n", "", nil, "weft scaffold: app/modules.go has no func Modules() []warp.Module returning a []warp.Module{...} list; write the modules as weft create does, then run weft scaffold again."},
		{"modules.go does not parse", "package app\n\nfunc Modules( {\n", "", nil, "weft scaffold: app/modules.go does not parse (app/modules.go:3:15: expected ')', found '{'); fix it, then run weft scaffold again."},
		{"no data module", strings.Replace(templateModules, "\t\tdata.Module(),\n", "", 1), "", nil, "weft scaffold: app/modules.go does not list data.Module(), which keeps the records; add it first in Modules(), import a database driver in main.go as weft create's template does, then run weft scaffold again."},
		{"a package of that name is imported", strings.Replace(templateModules, "\"myapp/app/hello\"", "\"myapp/app/hello\"\n\t\"example.com/products\"", 1), "", nil, "weft scaffold: app/modules.go imports a package named products already; choose another resource name, or give another with --plural."},
	} {
		t.Run(c.name, func(t *testing.T) {
			app := fakeApp(t, c.modules, true)
			if c.make != "" {
				writeFile(t, app, c.make, "keep\n")
			}
			t.Chdir(app)
			before := tree(t, app)
			code, out, errs := scaffold(t, append(product, c.args...)...)
			if code != 1 || errs != c.want+"\n" || out != "" {
				t.Errorf("exit %d, stdout %q, stderr %q; want exit 1, %q", code, out, errs, c.want)
			}
			if after := tree(t, app); !mapsEqual(before, after) {
				t.Errorf("the refusal changed the app:\nbefore %v\nafter  %v", keysOf(before), keysOf(after))
			}
		})
	}
	// app/modules.go missing is its own sentence.
	app := fakeApp(t, templateModules, true)
	os.Remove(filepath.Join(app, "app", "modules.go"))
	t.Chdir(app)
	if code, _, errs := scaffold(t, product...); code != 1 || errs != "weft scaffold: app/modules.go is missing; make it as weft create does, with a func Modules() []warp.Module, then run weft scaffold again.\n" {
		t.Errorf("no app/modules.go: exit %d, stderr %q", code, errs)
	}
	// Outside an app, the sentence every command says.
	t.Chdir(t.TempDir())
	if code, _, errs := scaffold(t, product...); code != 1 || errs != "weft scaffold: "+errNoApp+"\n" {
		t.Errorf("outside an app: exit %d, stderr %q", code, errs)
	}
}

func mapsEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if w, ok := b[k]; !ok || w != v {
			return false
		}
	}
	return true
}

func keysOf(m map[string]string) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}

// With --api the pages may be there: only the Go is planned.
func TestScaffoldAPIOnlyPlansNoPages(t *testing.T) {
	app := fakeApp(t, templateModules, true)
	writeFile(t, app, "web/pages/products/index.vue", "keep\n")
	s, perr := parseScaffold([]string{"product", "name", "--api"})
	if perr != nil {
		t.Fatal(perr.msg)
	}
	p, err := planScaffold(app, s)
	if err != nil {
		t.Fatal(err)
	}
	var paths []string
	for _, f := range p.files {
		paths = append(paths, f.path)
	}
	if want := []string{"app/products/model.go", "app/products/routes.go", "app/products/products_test.go", "app/modules.go"}; !slices.Equal(paths, want) {
		t.Errorf("--api plans %v, want %v", paths, want)
	}
}

// planned returns the plan of args in a fake app made from modules.
func planned(t *testing.T, modules string, session bool, args ...string) (map[string]string, *scaffoldPlan) {
	t.Helper()
	app := fakeApp(t, modules, session)
	s, perr := parseScaffold(args)
	if perr != nil {
		t.Fatal(perr.msg)
	}
	p, err := planScaffold(app, s)
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]string{}
	for _, f := range p.files {
		files[f.path] = string(f.data)
	}
	return files, p
}

// The Go weft scaffold writes is gofmt's, and says what the spec says.
func TestScaffoldWritesGo(t *testing.T) {
	files, p := planned(t, templateModules, true, "product", "name:string:required:max=120:unique", "price:float:min=0", "sale:float:nullable", "status:enum=draft,live:required", "published_at:datetime:nullable")
	if !p.auth {
		t.Error("the plan does not see auth.Module() in app/modules.go")
	}
	for _, name := range []string{"app/products/model.go", "app/products/routes.go", "app/products/products_test.go", "app/modules.go"} {
		src, ok := files[name]
		if !ok {
			t.Errorf("%s is not planned", name)
			continue
		}
		if formatted, err := format.Source([]byte(src)); err != nil || string(formatted) != src {
			t.Errorf("%s is not gofmt's (%v):\n%s", name, err, src)
		}
	}
	model := files["app/products/model.go"]
	for _, want := range []string{
		"package products\n",
		"\t\"time\"\n",
		"\t\"github.com/flakerimi/weft/warp/data\"\n",
		"type Product struct {\n\tdata.Record\n",
		"\tName        string     `json:\"name\" warp:\"required,max=120,unique\" data:\"sort,search\"`\n",
		"\tPrice       float64    `json:\"price\" warp:\"min=0\" data:\"sort\"`\n",
		"\tSale        *float64   `json:\"sale\" data:\"sort\"`\n",
		"\tStatus      string     `json:\"status\" warp:\"required,enum=draft|live\" data:\"sort,filter\"`\n",
		"\tPublishedAt *time.Time `json:\"published_at\" data:\"sort\"`\n",
	} {
		if !strings.Contains(model, want) {
			t.Errorf("model.go lacks %q:\n%s", want, model)
		}
	}
	routes := files["app/products/routes.go"]
	for _, want := range []string{
		"func (module) Name() string { return \"products\" }\n",
		"\tstore := data.NewStore[Product](data.For(app), \"products\")\n",
		"\tdata.Resource(app, \"/products\", store, data.PerAction(auth.Per(\"products\")))\n",
		"func Module() warp.Module { return module{} }\n",
	} {
		if !strings.Contains(routes, want) {
			t.Errorf("routes.go lacks %q:\n%s", want, routes)
		}
	}
	story := files["app/products/products_test.go"]
	for _, want := range []string{
		"package products_test\n",
		"\t\"myapp/app\"\n\t\"myapp/app/products\"\n",
		"\t_ \"modernc.org/sqlite\"\n",
		"authtest.Owner(t, c)",
		"\"AUTH_SECRET\":",
		"c.Post(\"/api/products\", ",
		"Want(http.StatusCreated)",
		"?sort=-id",
		"?q=",
		"?filter%5Bstatus%5D=live",
		"c.Patch(\"/api/products/\"",
		"c.Delete(\"/api/products/\" + idOf(second)).Want(http.StatusNoContent)",
		"Want(http.StatusNotFound)",
		"Want(http.StatusUnauthorized)",
	} {
		if !strings.Contains(story, want) {
			t.Errorf("products_test.go lacks %q:\n%s", want, story)
		}
	}
	wantModules := strings.Replace(strings.Replace(templateModules,
		"\t\"myapp/app/hello\"\n", "\t\"myapp/app/hello\"\n\t\"myapp/app/products\"\n", 1),
		"\t\thello.Module(), // the public route\n", "\t\thello.Module(), // the public route\n\t\tproducts.Module(),\n", 1)
	if got := files["app/modules.go"]; got != wantModules {
		t.Errorf("app/modules.go =\n%s\nwant\n%s", got, wantModules)
	}
}

// With no auth in app/modules.go the routes are open, and the plan says so.
func TestScaffoldWithoutAuth(t *testing.T) {
	modules := strings.Replace(strings.Replace(templateModules, "\t\"github.com/flakerimi/weft/warp/auth\"\n", "", 1), "\t\tauth.Module(),\n", "", 1)
	files, p := planned(t, modules, false, "note", "title:string:required", "--api")
	if p.auth {
		t.Error("the plan sees auth where there is none")
	}
	routes := files["app/notes/routes.go"]
	if strings.Contains(routes, "auth") || !strings.Contains(routes, "\tdata.Resource(app, \"/notes\", store)\n") {
		t.Errorf("routes.go without auth:\n%s", routes)
	}
	story := files["app/notes/notes_test.go"]
	if strings.Contains(story, "auth") || strings.Contains(story, "Unauthorized") {
		t.Errorf("notes_test.go without auth names auth:\n%s", story)
	}
	if !strings.Contains(files["app/modules.go"], "\t\tnotes.Module(),\n\t}\n") {
		t.Errorf("app/modules.go:\n%s", files["app/modules.go"])
	}
}

// The edit of app/modules.go keeps every comment and the author's layout,
// however the list and the imports are written.
func TestScaffoldEditsModules(t *testing.T) {
	for _, c := range []struct{ name, in, want string }{
		{"one line, no import group",
			"package app\n\nimport \"github.com/flakerimi/weft/warp\"\nimport \"github.com/flakerimi/weft/warp/data\" // the records\n\n// Modules lists them.\nfunc Modules() []warp.Module { return []warp.Module{data.Module()} }\n",
			"package app\n\nimport \"github.com/flakerimi/weft/warp\"\nimport (\n\t\"github.com/flakerimi/weft/warp/data\" // the records\n\t\"myapp/app/notes\"\n)\n\n// Modules lists them.\nfunc Modules() []warp.Module { return []warp.Module{data.Module(), notes.Module()} }\n"},
		{"a trailing comma on one line",
			"package app\n\nimport (\n\t\"github.com/flakerimi/weft/warp\"\n\t\"github.com/flakerimi/weft/warp/data\"\n)\n\nfunc Modules() []warp.Module {\n\treturn []warp.Module{data.Module(),}\n}\n",
			"package app\n\nimport (\n\t\"github.com/flakerimi/weft/warp\"\n\t\"github.com/flakerimi/weft/warp/data\"\n\t\"myapp/app/notes\"\n)\n\nfunc Modules() []warp.Module {\n\treturn []warp.Module{data.Module(), notes.Module()}\n}\n"},
		{"a comment after the last module",
			"package app\n\nimport (\n\t\"github.com/flakerimi/weft/warp\"\n\twd \"github.com/flakerimi/weft/warp/data\"\n)\n\nfunc Modules() []warp.Module {\n\treturn []warp.Module{\n\t\twd.Module(),\n\t\t// more go here\n\t}\n}\n",
			"package app\n\nimport (\n\t\"github.com/flakerimi/weft/warp\"\n\twd \"github.com/flakerimi/weft/warp/data\"\n\t\"myapp/app/notes\"\n)\n\nfunc Modules() []warp.Module {\n\treturn []warp.Module{\n\t\twd.Module(),\n\t\t// more go here\n\t\tnotes.Module(),\n\t}\n}\n"},
	} {
		t.Run(c.name, func(t *testing.T) {
			files, _ := planned(t, c.in, false, "note", "title", "--api")
			if got := files["app/modules.go"]; got != c.want {
				t.Errorf("app/modules.go =\n%s\nwant\n%s", got, c.want)
			}
		})
	}
}

// Without --api, the islands and the two pages, against DataTable's props
// and the generated client; the session is waited on when the app has one.
func TestScaffoldWritesPages(t *testing.T) {
	files, _ := planned(t, templateModules, true, "line_item", "name:string:required", "body:text", "qty:int:min=1", "price:float:nullable", "live:bool", "kind:enum=a,b_c", "on:date:nullable", "at:datetime")
	table := files["web/components/LineItemsTable.vue"]
	for _, want := range []string{
		"<script setup vapor lang=\"ts\">",
		"import type { DataTableFilter, EntityFieldConfig, TableColumn } from '@weft/ui'",
		"import { api, ApiError, schema, type LineItem } from '../.weft-types/api.ts'",
		"import { requireUser, signInAgain, useSession } from '../session.ts'",
		"const columns: TableColumn<LineItem>[] = [",
		"{ key: 'name', label: 'Name', sortable: true, rowHeader: true },",
		"{ key: 'qty', label: 'Qty', sortable: true, align: 'end', format: number },",
		"{ key: 'on', label: 'On', sortable: true, format: day },",
		"{ key: 'at', label: 'At', sortable: true, format: moment },",
		"{ key: 'kind', label: 'Kind', items: [{ label: 'A', value: 'a' }, { label: 'B c', value: 'b_c' }] },",
		"{ key: 'live', label: 'Live', items: [{ label: 'Yes', value: 'true' }, { label: 'No', value: 'false' }] },",
		"{ name: 'body', label: 'Body', control: 'textarea' },",
		"{ name: 'qty', label: 'Qty', step: 1 },",
		"<DataTable\n    v-if=\"ready\"",
		":resource=\"api.lineItems\"",
		":schema=\"schema.LineItemInput\"",
		":columns=\"columns\"",
		":filter-defs=\"filterDefs\"",
		":fields=\"fields\"",
		"name=\"line item\"",
		"caption=\"Line items\"",
		"empty-title=\"No line items\"",
		"@error=\"listFailed\"",
		"<template #name-cell=\"{ row }\">\n      <Link :to=\"`/line_items/${row.id}`\">{{ row.name }}</Link>",
		"<template #kind-cell=\"{ row }\">",
		"<template #live-cell=\"{ row }\">",
	} {
		if !strings.Contains(table, want) {
			t.Errorf("LineItemsTable.vue lacks %q:\n%s", want, table)
		}
	}
	if strings.Contains(table, "{ key: 'body'") {
		t.Errorf("LineItemsTable.vue has a column for the text field body:\n%s", table)
	}
	view := files["web/components/LineItemView.vue"]
	for _, want := range []string{
		"<script setup vapor lang=\"ts\">",
		"import { useParams } from '@weft/ui'",
		"import { api, ApiError, type LineItem } from '../.weft-types/api.ts'",
		"import { requireUser, useSession } from '../session.ts'",
		"useParams('/line_items/:id').id",
		"await api.lineItems.show(id)",
		"<Link to=\"/line_items\">Back to the line items</Link>",
	} {
		if !strings.Contains(view, want) {
			t.Errorf("LineItemView.vue lacks %q:\n%s", want, view)
		}
	}
	index := files["web/pages/line_items/index.vue"]
	if !strings.Contains(index, "import LineItemsTable from '../../components/LineItemsTable.vue'") || !strings.Contains(index, "<Island when=\"load\" :component=\"LineItemsTable\" />") {
		t.Errorf("index.vue:\n%s", index)
	}
	one := files["web/pages/line_items/[id].vue"]
	if !strings.Contains(one, "export const runtime = true") || !strings.Contains(one, "<Island when=\"load\" :component=\"LineItemView\" />") {
		t.Errorf("[id].vue:\n%s", one)
	}
	for name, src := range files {
		if strings.HasSuffix(name, ".vue") && strings.Contains(src, "fetch(") {
			t.Errorf("%s calls fetch, not the generated client:\n%s", name, src)
		}
	}

	// An app with no web/session.ts: nothing waits on a session.
	files, _ = planned(t, templateModules, false, "note", "title")
	for _, name := range []string{"web/components/NotesTable.vue", "web/components/NoteView.vue"} {
		if src := files[name]; strings.Contains(src, "session") || strings.Contains(src, "requireUser") {
			t.Errorf("%s waits on a session the app does not have:\n%s", name, src)
		}
	}
	if src := files["web/components/NotesTable.vue"]; !strings.Contains(src, "<DataTable\n    :resource=\"api.notes\"") {
		t.Errorf("NotesTable.vue:\n%s", src)
	}
}

// A model with no string field links its rows from the id.
func TestScaffoldLinksFromTheID(t *testing.T) {
	files, _ := planned(t, templateModules, true, "reading", "value:float:required")
	table := files["web/components/ReadingsTable.vue"]
	if !strings.Contains(table, "{ key: 'id', label: 'Id', sortable: true, rowHeader: true },") || !strings.Contains(table, "<template #id-cell=\"{ row }\">") || !strings.Contains(table, "label-key=\"id\"") {
		t.Errorf("ReadingsTable.vue:\n%s", table)
	}
}

// scaffoldedApp is a copy of the installed app, the shop's frame.
func scaffoldedApp(t *testing.T) string {
	t.Helper()
	inTempDir(t, "work")
	app := copyApp(t, ".")
	t.Chdir(app)
	return app
}

// goIn runs go with args in dir and fails the test with its output.
func goIn(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("go", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return string(out)
}

// In a created app: weft scaffold writes the resource, its Go vets and its
// story test passes over SQLite, weft gen has run, and the client has the
// five routes and the schema.
func TestScaffoldInACreatedApp(t *testing.T) {
	app := scaffoldedApp(t)
	code, out, errs := scaffold(t, "product", "name:string:required:max=120:unique", "price:float:min=0", "sale:float:nullable", "status:enum=draft,live:required")
	if code != 0 {
		t.Fatalf("exit %d: %s%s", code, out, errs)
	}
	wantOut := "Scaffolded product as products:\n" +
		"  app/products/model.go\n  app/products/routes.go\n  app/products/products_test.go\n  app/modules.go (products.Module() added)\n" +
		"  web/components/ProductsTable.vue\n  web/components/ProductView.vue\n  web/pages/products/index.vue\n  web/pages/products/[id].vue\n"
	if !strings.HasPrefix(out, wantOut) || !strings.HasSuffix(out, "Wrote web/.weft-types/api.ts\n") {
		t.Errorf("stdout = %q, want it to start %q and end with weft gen's line", out, wantOut)
	}
	goIn(t, app, "vet", "-tags", "dev", "./...")
	if test := goIn(t, app, "test", "-count=1", "-tags", "dev", "./..."); !strings.Contains(test, "ok  \tmyapp/app/products") {
		t.Errorf("go test did not run the story:\n%s", test)
	}
	src, err := os.ReadFile(filepath.Join(app, "web", ".weft-types", "api.ts"))
	if err != nil {
		t.Fatal(err)
	}
	api := string(src)
	block := apiBlock(api, "products")
	for _, fn := range []string{"list", "show", "create", "update", "remove"} {
		if !regexp.MustCompile(`\basync ` + fn + `\(`).MatchString(block) {
			t.Errorf("api.products has no %s: %q", fn, block)
		}
	}
	if !regexp.MustCompile(`(?m)^  ProductInput: \{$`).MatchString(api) {
		t.Errorf("the client has no schema.ProductInput:\n%s", api)
	}
	// weft routes shows the permission per action.
	var rout, rerr bytes.Buffer
	if code := run([]string{"routes"}, &rout, &rerr); code != 0 || !strings.Contains(rout.String(), "products:delete") {
		t.Errorf("weft routes: exit %d: %s%s", code, &rout, &rerr)
	}
	// A second scaffold of the same name is refused, and changes nothing.
	before := tree(t, filepath.Join(app, "app"))
	if code, _, errs := scaffold(t, "product", "name"); code != 1 || !strings.Contains(errs, "app/products already exists") {
		t.Errorf("a second weft scaffold product: exit %d, %q", code, errs)
	}
	if after := tree(t, filepath.Join(app, "app")); !mapsEqual(before, after) {
		t.Error("the refused second scaffold changed app/")
	}
}

// Every type at once, with --api: only Go is written, it vets, and the
// story passes, the pointers sent and the times kept.
func TestScaffoldEveryTypeAPI(t *testing.T) {
	app := scaffoldedApp(t)
	code, out, errs := scaffold(t, "entry", "title", "body:text:nullable:max=500", "mail:email:required:unique", "site:url:nullable", "count:int:min=1:max=9", "score:float:nullable:min=-1.5", "live:bool", "seen:bool:nullable", "on:date", "at:datetime:nullable", "kind:enum=a,b,c:required", "mood:enum=up,down:nullable", "--api", "--plural", "journal")
	if code != 0 {
		t.Fatalf("exit %d: %s%s", code, out, errs)
	}
	if !strings.HasPrefix(out, "Scaffolded entry as journal:\n") || strings.Contains(out, "web/") && !strings.Contains(out, "web/.weft-types/api.ts") {
		t.Errorf("stdout = %q", out)
	}
	if exists(t, filepath.Join(app, "web", "pages", "journal")) || exists(t, filepath.Join(app, "web", "components", "JournalTable.vue")) {
		t.Error("--api wrote pages")
	}
	goIn(t, app, "vet", "-tags", "dev", "./...")
	if test := goIn(t, app, "test", "-count=1", "-tags", "dev", "./app/journal"); !strings.Contains(test, "ok  \tmyapp/app/journal") {
		t.Errorf("go test:\n%s", test)
	}
}

// An app whose modules have no auth gets open routes, a one-line note, and
// a story with no login in it, which passes.
func TestScaffoldWithoutAuthInACreatedApp(t *testing.T) {
	app := scaffoldedApp(t)
	file := filepath.Join(app, "app", "modules.go")
	src, _ := os.ReadFile(file)
	open := strings.Replace(strings.Replace(string(src), "\t\"github.com/flakerimi/weft/warp/auth\"\n", "", 1), "\t\tauth.Module(),\n", "", 1)
	os.WriteFile(file, []byte(open), 0o644)
	code, out, errs := scaffold(t, "note", "title:string:required", "--api")
	if code != 0 {
		t.Fatalf("exit %d: %s%s", code, out, errs)
	}
	if note := "weft scaffold: app/modules.go lists no auth module, so the routes of notes are open to anyone; to guard them, add auth.Module() and give data.Resource data.PerAction(auth.Per(\"notes\")).\n"; !strings.Contains(out, note) {
		t.Errorf("stdout = %q, want the note %q", out, note)
	}
	goIn(t, app, "vet", "-tags", "dev", "./...")
	if test := goIn(t, app, "test", "-count=1", "-tags", "dev", "./app/notes"); !strings.Contains(test, "ok  \tmyapp/app/notes") {
		t.Errorf("go test:\n%s", test)
	}
}

// The whole way, as a person goes it: weft create, weft scaffold with the
// pages, weft build, the type check, and the pages driven in a browser.
func TestScaffoldedPagesInBrowser(t *testing.T) {
	t.Skip("needs DataTable and EntityDialog from @weft/ui (W5, branch w5-ui), which are not on main yet")
	app := scaffoldedApp(t)
	if code, out, errs := scaffold(t, "product", "name:string:required:unique", "price:float:min=0", "sale:float:nullable", "status:enum=draft,live:required"); code != 0 {
		t.Fatalf("weft scaffold: exit %d: %s%s", code, out, errs)
	}
	if out, err := exec.Command("pnpm", "typecheck").CombinedOutput(); err != nil {
		t.Fatalf("pnpm typecheck: %v\n%s", err, out)
	}
	if code, out, errs := build(t); code != 0 {
		t.Fatalf("weft build: exit %d: %s%s", code, out, errs)
	}
	url, token := startApp(t, filepath.Join(app, "myapp"), app)
	for path, want := range map[string]string{
		"/products":    `<weft-island data-id="ProductsTable"`,
		"/products/42": `data-weft-route="/products/:id"`,
	} {
		if code, body := get(t, url+path); code != 200 || !strings.Contains(body, want) {
			t.Errorf("GET %s = %d, want 200 with %q: %s", path, code, want, body)
		}
	}
	browser := browserPath()
	if browser == "" {
		t.Skip("no Chrome or Chromium is installed; set WEFT_BROWSER to drive the pages in one")
	}
	// The script is the test's own, kept beside it in cmd/weft/testdata.
	script := filepath.Join(repoCmdWeft, "testdata", "scaffold-browser.mjs")
	cmd := exec.Command("node", script, browser, url, url+"/setup#token="+token, t.TempDir())
	cmd.Dir = app
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("the scaffolded pages in %s: %v\n%s", browser, err, out)
	} else {
		t.Logf("drove the scaffolded pages:\n%s", out)
	}
}

// repoCmdWeft is cmd/weft, found while the test binary still runs there.
var repoCmdWeft = func() string {
	wd, _ := os.Getwd()
	return wd
}()
