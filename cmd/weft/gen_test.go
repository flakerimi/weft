package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// fixture is the description in testdata/describe.json, which a small app
// built on warp printed (see the task's report for how it was made).
func fixture(t *testing.T) description {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "describe.json"))
	if err != nil {
		t.Fatal(err)
	}
	d, err := parseDescription(b)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func mustGenerate(t *testing.T, d description) string {
	t.Helper()
	b, err := generate(d)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// firstDifference names the first line where got and want differ.
func firstDifference(got, want string) string {
	g, w := strings.Split(got, "\n"), strings.Split(want, "\n")
	for i := 0; i < len(g) || i < len(w); i++ {
		var gl, wl string
		if i < len(g) {
			gl = g[i]
		}
		if i < len(w) {
			wl = w[i]
		}
		if gl != wl {
			return "line " + strconv.Itoa(i+1) + ":\n got " + gl + "\nwant " + wl
		}
	}
	return ""
}

func TestGenerateGolden(t *testing.T) {
	got := mustGenerate(t, fixture(t))
	want, err := os.ReadFile(filepath.Join("testdata", "api.ts"))
	if err != nil {
		t.Fatal(err)
	}
	if got != string(want) {
		out := filepath.Join(t.TempDir(), "api.ts")
		os.WriteFile(out, []byte(got), 0o644)
		t.Fatalf("generate differs from testdata/api.ts (written to %s); first difference at %s", out, firstDifference(got, string(want)))
	}
}

// The same description gives the same bytes, decoded afresh or not.
func TestGenerateIsStable(t *testing.T) {
	first := mustGenerate(t, fixture(t))
	for range 20 {
		if again := mustGenerate(t, fixture(t)); again != first {
			t.Fatalf("two generations differ at %s", firstDifference(again, first))
		}
	}
	if !strings.HasSuffix(first, "}\n") || strings.HasSuffix(first, "\n\n") {
		t.Errorf("the file does not end with one newline: %q", first[len(first)-20:])
	}
}

// route is a typed route of module with the given name, for the name tests.
func route(module, name string) routeInfo {
	return routeInfo{Module: module, Name: name, Method: "GET", Path: "/api/" + name, Typed: true}
}

func TestGenerateNameClash(t *testing.T) {
	for _, c := range []struct {
		name string
		d    description
		want string
	}{
		{"routes", description{Routes: []routeInfo{route("x", "post_a_b"), route("x", "post_a__b")}},
			`routes "post_a_b" and "post_a__b" of module "x" both become postAB; give one a warp.Name.`},
		{"routes outside a module", description{Routes: []routeInfo{route("", "a_b"), route("", "a-b")}},
			`routes "a_b" and "a-b" outside any module both become aB; give one a warp.Name.`},
		{"modules", description{Routes: []routeInfo{route("shop_items", "list"), route("shop-items", "show")}},
			`modules "shop_items" and "shop-items" both become api.shopItems; rename one.`},
		{"app", description{Routes: []routeInfo{route("", "health"), route("app", "show")}},
			`the routes outside any module and module "app" both become api.app; rename the module.`},
		{"route starting with a digit", description{Routes: []routeInfo{route("x", "2fa")}},
			`route "2fa" of module "x" becomes 2fa, which TypeScript cannot use as a name since it starts with a digit; give it a warp.Name that starts with a letter.`},
		{"route with a space", description{Routes: []routeInfo{route("x", "list all")}},
			`route "list all" of module "x" becomes list all, which TypeScript cannot use as a name since it has a space; give it a warp.Name of letters, digits and _.`},
		{"route with a slash", description{Routes: []routeInfo{route("", "a/b")}},
			`route "a/b" outside any module becomes a/b, which TypeScript cannot use as a name since it has "/"; give it a warp.Name of letters, digits and _.`},
		{"route of separators", description{Routes: []routeInfo{route("x", "_-_")}},
			`route "_-_" of module "x" becomes nothing, since it is only _, - and .; give it a warp.Name of letters, digits and _.`},
		{"module starting with a digit", description{Routes: []routeInfo{route("3d", "list")}},
			`module "3d" becomes api.3d, which TypeScript cannot use as a name since it starts with a digit; rename the module so it starts with a letter.`},
		{"module with a space", description{Routes: []routeInfo{route("my shop", "list")}},
			`module "my shop" becomes api.my shop, which TypeScript cannot use as a name since it has a space; rename the module to letters, digits and _.`},
		{"type the file uses", description{Types: []typeInfo{{Name: "Record", Go: "example.com/app/data.Record", Fields: []fieldInfo{}}}},
			`the type Record (example.com/app/data.Record) has a name the generated file uses itself; rename the Go type.`},
		{"type named a reserved word", description{Types: []typeInfo{{Name: "class", Go: "example.com/app.class", Fields: []fieldInfo{}}}},
			`the type class (example.com/app.class) has a name TypeScript reserves; rename the Go type.`},
		{"type named a type operator", description{Types: []typeInfo{{Name: "keyof", Go: "example.com/app.keyof", Fields: []fieldInfo{}}}},
			`the type keyof (example.com/app.keyof) has a name TypeScript reserves; rename the Go type.`},
		{"optional field named as every object's property", description{Types: []typeInfo{{Name: "Thing", Go: "example.com/app.Thing", Fields: []fieldInfo{{Name: "constructor", TS: "string", Optional: true}}}}},
			`the optional field Thing.constructor has the name of a property every JavaScript object has, which TypeScript then cannot check; make it required or give it another json name.`},
		{"query parameter named as every object's property", description{Routes: []routeInfo{{Module: "x", Name: "list", Method: "GET", Path: "/api/x", Typed: true, Query: []paramInfo{{Name: "toString", TS: "string"}}}}},
			`route "list" of module "x" has the query parameter toString, the name of a property every JavaScript object has, which TypeScript then cannot check; give it another query name.`},
		{"field named as every object's property in a partial body", description{
			Routes: []routeInfo{{Module: "x", Name: "change", Method: "PATCH", Path: "/api/x", Typed: true, Body: "Thing", Partial: true}},
			Types:  []typeInfo{{Name: "Thing", Go: "example.com/app.Thing", Fields: []fieldInfo{{Name: "toString", TS: "string"}}}}},
			`the field Thing.toString has the name of a property every JavaScript object has, which TypeScript then cannot check where a route takes Thing as a partial body; give it another json name.`},
		{"path parameters that become one", description{Routes: []routeInfo{{Module: "x", Name: "show", Method: "GET", Path: "/api/x/{class}/{class_}", Typed: true}}},
			`route "show" of module "x" has the path parameters {class} and {class_}, which both become class_ in TypeScript; rename one.`},
	} {
		t.Run(c.name, func(t *testing.T) {
			// weft gen prints it with fail, which ends it with a period.
			_, err := generate(c.d)
			if err == nil || err.Error()+"." != c.want {
				t.Fatalf("generate = %v, want %q", err, c.want)
			}
		})
	}
	// A route added with Handle has no function, so its name is not checked.
	d := description{Routes: []routeInfo{route("x", "list"), {Module: "x", Name: "x_export.csv", Path: "/api/x/export.csv"}}}
	if _, err := generate(d); err != nil {
		t.Fatalf("generate = %v for a Handle route", err)
	}
}

func TestCamelNames(t *testing.T) {
	for in, want := range map[string]string{
		"hello":               "hello",
		"post_hello_greet":    "postHelloGreet",
		"products_export.csv": "productsExportCsv",
		"delete_old-items":    "deleteOldItems",
		"listAll":             "listAll",
		"_x_":                 "x",
	} {
		if got := camel(in); got != want {
			t.Errorf("camel(%q) = %q, want %q", in, got, want)
		}
	}
}

// Every name the language reserves, checked against what tsc refused when
// each was written as a generated interface's name.
func TestReservedTypeNames(t *testing.T) {
	refused := strings.Fields(`any await bigint boolean break case catch class const continue debugger default delete do else enum
		export extends false finally for function if implements import in infer instanceof interface keyof let never new null number
		object package private protected public readonly return static string super switch symbol this throw true try typeof
		undefined unique unknown var void while with yield`)
	for _, w := range refused {
		if !reservedTypeNames[w] {
			t.Errorf("%s is not refused as a type name", w)
		}
	}
	if len(reservedTypeNames) != len(refused) {
		t.Errorf("%d reserved type names, want %d", len(reservedTypeNames), len(refused))
	}
}

// The parts of the output the fixture does not reach: a key that is not a
// name, a path parameter that is a reserved word or would hide a name the
// function uses, a list of nullable answers, a scalar answer, {$}, and no
// route at all.
func TestGenerateEdges(t *testing.T) {
	d := description{
		Routes: []routeInfo{
			{Module: "x", Name: "show", Method: "GET", Path: "/api/x/{class}/{search}/{$}", Typed: true,
				PathParams: []paramInfo{{Name: "search", TS: "string"}, {Name: "class", TS: "number"}},
				Query:      []paramInfo{{Name: "sort-by", TS: "string"}},
				Output:     &outputInfo{TS: "Thing | null", List: true}},
			{Module: "x", Name: "count", Method: "GET", Path: "/api/x/count", Typed: true, Output: &outputInfo{TS: "number"}},
			{Module: "x", Name: "put_x", Method: "PUT", Path: "/api/x/{body}", Typed: true, Body: "Thing",
				PathParams: []paramInfo{{Name: "body", TS: "string"}}, Output: &outputInfo{TS: "Thing"}},
		},
		Types: []typeInfo{{Name: "Thing", Go: "example.com/x.Thing", Fields: []fieldInfo{
			{Name: "x-y", TS: "string", Rules: []ruleInfo{{Name: "required"}, {Name: "unique"}, {Name: "later", Arg: "1"}}},
			{Name: "it's", TS: "number", Rules: []ruleInfo{{Name: "min", Arg: "1.5"}, {Name: "enum", Arg: "a'b|c"}}},
			{Name: "__proto__", TS: "string"},
		}}},
	}
	got := mustGenerate(t, d)
	for _, want := range []string{
		"  'x-y': string\n  'it\\'s': number\n",
		"    async show(class_: number, search_: string, query?: { 'sort-by'?: string }): Promise<(Thing | null)[]> {\n" +
			"      return send('GET', `/api/x/${segment('class', class_)}/${segment('search', search_)}/${search(query)}`)\n",
		"    async count(): Promise<number> {\n      return send('GET', '/api/x/count')\n",
		"    async putX(body_: string, body: Thing): Promise<Thing> {\n      return send('PUT', `/api/x/${segment('body', body_)}`, { body })\n",
		"    'x-y': { kind: 'string', required: true },\n    'it\\'s': { kind: 'number', min: 1.5, enum: ['a\\'b', 'c'] },\n",
		"  __proto__: string\n",
		"    ['__proto__']: { kind: 'string' },\n",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("the output has no\n%s\nin\n%s", want, got)
		}
	}

	empty := mustGenerate(t, description{})
	if !strings.Contains(empty, "\nexport const api = {}\n") || !strings.HasSuffix(empty, "\nexport const schema = {}\n") {
		t.Errorf("an app with no typed route:\n%s", empty)
	}
}

// parseDescription takes what an app prints and refuses anything else.
func TestParseDescription(t *testing.T) {
	for _, bad := range []string{"", "hello\n", "{}", `{"routes": []} trailing`, "[]"} {
		if _, err := parseDescription([]byte(bad)); err == nil {
			t.Errorf("parseDescription(%q) = nil error", bad)
		}
	}
	// A field it does not know, from a newer warp, is ignored.
	d, err := parseDescription([]byte(`{"routes":[{"module":"m","name":"n","typed":true,"later":1}],"types":[],"later":{}}`))
	if err != nil || len(d.Routes) != 1 || d.Routes[0].Name != "n" {
		t.Fatalf("parseDescription = %+v, %v", d, err)
	}
}

func TestRoutesPrints(t *testing.T) {
	want := `METHOD  PATH                               MODULE    NAME                 PERMISSION
GET     /api/health                        -         health               -
GET     /api/hello                         hello     hello                -
POST    /api/hello/greet                   hello     post_hello_greet     -
GET     /api/hello/{slug}/files/{rest...}  hello     file                 -
DELETE  /api/hello/{slug}                  hello     delete               -
GET     /api/products                      products  list                 products:read
GET     /api/products/{id}                 products  show                 products:read
POST    /api/products                      products  create               products:create
PATCH   /api/products/{id}                 products  update               products:edit
DELETE  /api/products/{id}                 products  remove               products:delete
GET     /api/products/{id}/parts           products  products_parts       -
POST    /api/products/{id}/notes           products  post_products_notes  -
GET     /api/products/export.csv           products  products_export.csv  products:read
`
	if got := formatRoutes(fixture(t)); got != want {
		t.Errorf("formatRoutes =\n%s\nwant\n%s", got, want)
	}
	// A Handle route that takes every method.
	d := description{Routes: []routeInfo{{Module: "x", Name: "files", Path: "/api/files/"}}}
	if got, want := formatRoutes(d), "METHOD  PATH         MODULE  NAME   PERMISSION\nANY     /api/files/  x       files  -\n"; got != want {
		t.Errorf("formatRoutes = %q, want %q", got, want)
	}
}

func TestGenOutsideApp(t *testing.T) {
	t.Chdir(t.TempDir())
	for _, c := range []string{"gen", "routes"} {
		code, out, errs := runCmd(c)
		if want := "weft " + c + ": no Weft app here; run it in a folder made by weft create.\n"; code != 1 || errs != want || out != "" {
			t.Errorf("%s: exit %d, stdout %q, stderr %q, want %q", c, code, out, errs, want)
		}
		code, _, errs = runCmd(c, "extra")
		if code != 2 || !strings.HasPrefix(errs, "weft "+c+": ") {
			t.Errorf("%s extra: exit %d, stderr %q", c, code, errs)
		}
	}
}

// helperApp builds a program that, by the MODE in its environment, prints
// what an app might print where its description should be.
func helperApp(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	src := `package main

import (
	"os"
	"strings"
)

func main() {
	switch os.Getenv("MODE") {
	case "big":
		chunk := []byte(strings.Repeat("x", 1<<16))
		for range 9 << 4 {
			os.Stdout.Write(chunk)
		}
	case "text":
		os.Stdout.WriteString("just text\n" + strings.Repeat("y", 1000))
	case "json":
		os.Stdout.WriteString("{\"routes\":[],\"types\":[]}\n")
	case "fail":
		os.Stdout.WriteString("partial\n")
		os.Stderr.WriteString("boom\n")
		os.Exit(3)
	}
}
`
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module helper\n\ngo 1.26\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "main.go"), []byte(src), 0o644)
	bin := filepath.Join(dir, "helper")
	cmd := exec.Command("go", "build", "-o", bin, ".")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	return bin
}

// describeApp reads at most 8 MiB, shows at most 200 bytes of an output that
// is not a description, and says what it began with.
func TestDescribeAppOutput(t *testing.T) {
	bin := helperApp(t)
	describe := func(mode string) (description, error, string) {
		t.Helper()
		t.Setenv("MODE", mode)
		var stderr bytes.Buffer
		d, err := describeApp(context.Background(), filepath.Dir(bin), bin, 30*time.Second, &stderr)
		return d, err, stderr.String()
	}
	if d, err, errs := describe("json"); err != nil || d.Routes == nil || errs != "" {
		t.Errorf("json: %+v, %v, stderr %q", d, err, errs)
	}
	_, err, errs := describe("big")
	if want := "the app printed more than 8 MB where its description should be; make sure nothing but Warp writes to standard output when WARP_DESCRIBE is set"; err == nil || err.Error() != want || len(errs) > 0 {
		t.Errorf("big: %v, %d bytes on stderr, want %q", err, len(errs), want)
	}
	text := "just text\n" + strings.Repeat("y", 1000)
	_, err, errs = describe("text")
	want := `the app's standard output was not its description (it starts with ` + strconv.Quote(text[:60]) + `); make sure nothing else prints there when WARP_DESCRIBE is set`
	if err == nil || err.Error() != want || errs != text[:200]+"\n" {
		t.Errorf("text: %v, stderr %q, want %q and the first 200 bytes", err, errs, want)
	}
	if _, err, errs := describe(""); !errors.Is(err, errNoDescription) || errs != "" {
		t.Errorf("nothing printed: %v, stderr %q", err, errs)
	}
	if _, err, errs := describe("fail"); !errors.Is(err, errNoDescription) || !strings.Contains(errs, "boom") || !strings.Contains(errs, "partial") {
		t.Errorf("fail: %v, stderr %q", err, errs)
	}
}

// Stopped by Ctrl-C, weft gen and weft routes say nothing and exit 130.
func TestGenFailedQuietWhenStopped(t *testing.T) {
	var stderr bytes.Buffer
	if code := genFailed(&stderr, "gen", fmt.Errorf("describing: %w", context.Canceled)); code != 130 || stderr.Len() != 0 {
		t.Errorf("genFailed = %d, stderr %q", code, stderr.String())
	}
}

// repoTool is a command from the repository's node_modules, or the test is
// skipped when there is none.
func repoTool(t *testing.T, name string) string {
	t.Helper()
	if testing.Short() {
		t.Skip("runs the repository's TypeScript and Node")
	}
	bin, err := filepath.Abs(filepath.Join("..", "..", "packages", "ui", "node_modules", ".bin", name))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(bin); err != nil {
		t.Skipf("no %s: run pnpm install at the repository's root", bin)
	}
	return bin
}

var fieldRuleRe = regexp.MustCompile(`(?s)export interface FieldRule \{.*?\n\}\n|export type FieldSchema<[^\n]*\n`)

// The golden file passes tsc --strict, its schema is a FieldSchema of
// @weft/ui's Form, and its types refuse a body with a field the Go type does
// not have.
func TestGeneratedCompiles(t *testing.T) {
	tsc := repoTool(t, "tsc")
	dir := t.TempDir()
	golden, err := os.ReadFile(filepath.Join("testdata", "api.ts"))
	if err != nil {
		t.Fatal(err)
	}
	// FieldRule and FieldSchema as @weft/ui declares them, alone, so a
	// change being made to the rest of schema.ts does not fail this test.
	schemaTS, err := os.ReadFile(filepath.Join("..", "..", "packages", "ui", "src", "forms", "schema.ts"))
	if err != nil {
		t.Fatal(err)
	}
	decls := fieldRuleRe.FindAllString(string(schemaTS), -1)
	if len(decls) != 2 {
		t.Fatalf("found %d of FieldRule and FieldSchema in packages/ui/src/forms/schema.ts", len(decls))
	}
	use := `import { api, schema, ApiError, setBase, type ProductInput, type Product } from './api.ts'
import type { FieldSchema } from './rules.ts'

export const product: FieldSchema = schema.ProductInput
export const named: FieldSchema<Extract<keyof ProductInput, string>> = schema.ProductInput
export const note: FieldSchema = schema.Note
export const greeting: FieldSchema = schema.Greeting

export async function use(): Promise<void> {
  setBase('https://api.example.com')
  const p: Product = await api.products.update(1, { name: 'New' })
  const left: void = await api.products.remove(p.id)
  const parts = await api.products.productsParts(p.id)
  const sku: string = parts[0].sku
  await api.products.list({ page: 2, filter: { status: 'live' }, fields: ['name'] })
  // @ts-expect-error: the Go type has no field nam
  await api.products.update(1, { nam: 'New' })
  // @ts-expect-error: a create takes every field
  await api.products.create({ name: 'New' })
  // @ts-expect-error: a filter is a map of text
  await api.products.list({ filter: { status: 1 } })
  try {
    await api.hello.delete('x')
  } catch (e) {
    if (e instanceof ApiError) console.log(e.status, e.code, e.message, e.fields, left, sku)
  }
}
`
	for name, text := range map[string]string{"api.ts": string(golden), "rules.ts": strings.Join(decls, ""), "use.ts": use} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	cmd := exec.Command(tsc, "--noEmit", "--strict", "--target", "es2022", "--lib", "es2022,dom",
		"--module", "esnext", "--moduleResolution", "bundler", "--allowImportingTsExtensions", "--verbatimModuleSyntax",
		"api.ts", "use.ts")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("tsc: %v\n%s", err, out)
	}
}

// tscCheck compiles the files, by name, with tsc --strict in a new folder.
func tscCheck(t *testing.T, tsc string, files map[string]string, roots ...string) {
	t.Helper()
	dir := t.TempDir()
	for name, text := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	args := append([]string{"--noEmit", "--strict", "--target", "es2022", "--lib", "es2022,dom",
		"--module", "esnext", "--moduleResolution", "bundler", "--allowImportingTsExtensions", "--verbatimModuleSyntax"}, roots...)
	cmd := exec.Command(tsc, args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("tsc: %v\n%s", err, out)
	}
}

// A generated file whose types, fields, modules, routes and parameters use
// every awkward name TypeScript allows compiles, and its functions can be
// called by those names.
func TestGenerateAwkwardNamesCompile(t *testing.T) {
	tsc := repoTool(t, "tsc")
	var fields []fieldInfo
	for _, n := range []string{"class", "delete", "new", "__proto__", "prototype", "length", "name", "x-y", "it's",
		"1st", "$", "_", "ünï", "kind", "required", "a b", "type", "keyof"} {
		fields = append(fields, fieldInfo{Name: n, TS: "string", Rules: []ruleInfo{{Name: "max", Arg: "3"}}})
	}
	var types []typeInfo
	names := []string{"Error", "String", "Object", "Array", "Date", "Map", "Symbol", "Function", "JSON", "URLSearchParams",
		"type", "module", "namespace", "declare", "abstract", "is", "as", "of", "async", "get", "set", "constructor", "eval",
		"arguments", "_x", "Ü", "Extract", "segment", "search", "send", "base"}
	for _, n := range names {
		types = append(types, typeInfo{Name: n, Go: "example.com/x." + n, Fields: []fieldInfo{{Name: "v", TS: n + " | null", Optional: true}}})
	}
	types = append(types, typeInfo{Name: "Thing", Go: "example.com/x.Thing", Fields: fields})
	params := []paramInfo{{Name: "new", TS: "string"}, {Name: "default", TS: "number"}, {Name: "body", TS: "string"},
		{Name: "query", TS: "string"}, {Name: "segment", TS: "string"}, {Name: "segments", TS: "string"}, {Name: "refuse", TS: "string"},
		{Name: "search", TS: "string"}, {Name: "send", TS: "string"}, {Name: "String", TS: "string"}, {Name: "rest", TS: "string"}}
	path := "/api/a"
	for _, p := range params[:len(params)-1] {
		path += "/{" + p.Name + "}"
	}
	path += "/{rest...}"
	d := description{
		Routes: []routeInfo{
			{Module: "constructor", Name: "delete", Method: "PUT", Path: path, Typed: true, PathParams: params, Body: "Thing", Partial: true,
				Query:  []paramInfo{{Name: "class", TS: "string"}, {Name: "__proto__", TS: "string"}, {Name: "x-y", TS: "string[]"}},
				Output: &outputInfo{TS: "type"}},
			{Module: "constructor", Name: "new", Method: "GET", Path: "/api/new", Typed: true, Output: &outputInfo{TS: "Error | null", List: true}},
			{Module: "toString", Name: "__proto__", Method: "GET", Path: "/api/proto", Typed: true, Output: &outputInfo{TS: "Map"}},
			{Module: "default", Name: "hasOwnProperty", Method: "POST", Path: "/api/own", Typed: true, Body: "type"},
			{Module: "", Name: "valueOf", Method: "GET", Path: "/api/value/{$}", Typed: true, Output: &outputInfo{TS: "Ü"}},
		},
		Types: types,
	}
	src := mustGenerate(t, d)
	use := `import { api, schema, type Thing } from './api.ts'

export async function use(t: Thing): Promise<void> {
  const ty = await api.constructor.delete('n', 1, 'b', 'q', 's', 'ss', 'r', 'se', 'sd', 'st', 'a/b', { class: 'c' }, { class: 'c', ['__proto__']: 'p', 'x-y': ['a'] })
  const errs = await api.constructor.new()
  const m = await api.toString.proto()
  await api.default.hasOwnProperty({ v: null })
  const u = await api.app.valueOf()
  console.log(ty.v, errs.length, m.v, u.v, t.__proto__, t['x-y'], t.class, schema.Thing.__proto__.kind, schema.Thing['1st'].max, schema.type.v.kind)
}
`
	if strings.Contains(src, "\n    __proto__: {") || strings.Contains(src, "\n  __proto__: {") {
		t.Errorf("__proto__ is written as a plain key in an object, where it sets the prototype:\n%s", src)
	}
	tscCheck(t, tsc, map[string]string{"api.ts": src, "use.ts": use}, "api.ts", "use.ts")
}

// A required field named as an Object member compiles, in an output-only type
// and in a body type, and is written; an optional one, and any in a type a
// route takes as a partial body, are refused (see TestGenerateNameClash).
func TestGenerateRequiredObjectMemberFieldsCompile(t *testing.T) {
	tsc := repoTool(t, "tsc")
	member := func() []fieldInfo {
		var fs []fieldInfo
		for _, n := range []string{"constructor", "toString", "toLocaleString", "valueOf", "hasOwnProperty", "isPrototypeOf", "propertyIsEnumerable"} {
			fs = append(fs, fieldInfo{Name: n, TS: "string", Rules: []ruleInfo{{Name: "max", Arg: "3"}}})
		}
		return fs
	}
	d := description{
		Routes: []routeInfo{
			{Module: "x", Name: "show", Method: "GET", Path: "/api/x", Typed: true, Output: &outputInfo{TS: "Shown"}},
			{Module: "x", Name: "make", Method: "POST", Path: "/api/x", Typed: true, Body: "Made"},
		},
		Types: []typeInfo{
			{Name: "Shown", Go: "example.com/x.Shown", Fields: member()},
			{Name: "Made", Go: "example.com/x.Made", Fields: member()},
		},
	}
	src := mustGenerate(t, d)
	use := `import { api, schema, type Shown } from './api.ts'

export async function use(s: Shown): Promise<void> {
  const full = { constructor: 'a', toString: 'b', toLocaleString: 'c', valueOf: 'd', hasOwnProperty: 'e', isPrototypeOf: 'f', propertyIsEnumerable: 'g' }
  await api.x.make(full)
  const shown = await api.x.show()
  console.log(s.constructor, shown.valueOf, schema.Made.toString.max)
}
`
	tscCheck(t, tsc, map[string]string{"api.ts": src, "use.ts": use}, "api.ts", "use.ts")
}

// request is what the test server was sent.
type request struct {
	Method, URI, ContentType, Body string
}

// The generated client sends what the brief says and reads the answers: the
// query with lists repeated and filter[key], path parameters encoded, JSON
// bodies with their Content-Type and none without, the error shape as an
// ApiError, another error as one with code http, and no body read for a
// route that answers nothing.
func TestGeneratedClientRuns(t *testing.T) {
	repoTool(t, "tsc") // the same skip: the repository is installed
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("no node on PATH")
	}
	var mu sync.Mutex
	var got []request
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		mu.Lock()
		got = append(got, request{r.Method, r.RequestURI, r.Header.Get("Content-Type"), string(body)})
		mu.Unlock()
		switch r.Method + " " + r.URL.Path {
		case "GET /api/products":
			w.Write([]byte(`{"items":[],"total":0,"page":2,"per_page":20}`))
		case "PATCH /api/products/7":
			w.Write([]byte(`{"id":7,"name":"New"}`))
		case "DELETE /api/products/7":
			w.WriteHeader(http.StatusNoContent)
		case "POST /api/products":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnprocessableEntity)
			w.Write([]byte(`{"error":{"code":"invalid","message":"Check the fields.","fields":{"name":"This is already taken."}}}`))
		case "GET /api/hello":
			http.Error(w, "boom", http.StatusInternalServerError)
		case "GET /api/hello/a b/files/x/y z":
			w.Write([]byte(`{"message":"file"}`))
		case "POST /api/products/7/notes":
			// Not JSON: a route that answers nothing does not read it.
			w.Write([]byte("not json"))
		case "GET /api/health":
			// A success with no body, on a route that answers something.
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	dir := t.TempDir()
	golden, err := os.ReadFile(filepath.Join("testdata", "api.ts"))
	if err != nil {
		t.Fatal(err)
	}
	// A port nothing listens on, for a request that cannot be sent.
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	closed := l.Addr().String()
	l.Close()
	script := `import { api, setBase, ApiError } from './api.ts'
setBase(process.argv[2] + '/')
// Every failure is a rejection with an ApiError, never a throw.
const caught = async (call) => {
  let promise
  try {
    promise = call()
  } catch (e) {
    return 'threw instead of rejecting: ' + e
  }
  try {
    await promise
    return 'resolved'
  } catch (e) {
    return { api: e instanceof ApiError, status: e.status, code: e.code, message: e.message, fields: e.fields, cause: e.cause !== undefined }
  }
}
const out = {
  list: await api.products.list({ page: 2, q: 'a b', fields: ['name', 'price'], full: false, filter: { status: 'live' }, sort: undefined, per_page: null }),
  update: await api.products.update(7, { name: 'New' }),
  remove: String(await api.products.remove(7)),
  create: await caught(() => api.products.create({ name: 'x' })),
  hello: await caught(() => api.hello.hello()),
  file: await api.hello.file('a b', 'x/y z', { loud: true }),
  note: String(await api.products.postProductsNotes(7, { text: 'hi' })),
  dots: [
    await caught(() => api.hello.delete('..')),
    await caught(() => api.hello.delete('.')),
    await caught(() => api.hello.delete('')),
    await caught(() => api.hello.delete('...')),
  ],
  rest: [
    await caught(() => api.hello.file('x', 'a/../../b')),
    await caught(() => api.hello.file('x', 'a//b')),
    await caught(() => api.hello.file('x', 'a/')),
  ],
  filterNames: [
    await caught(() => api.products.list({ filter: { '': 'x' } })),
    await caught(() => api.products.list({ filter: { 'a]': 'x' } })),
    await caught(() => api.products.list({ filter: { 'a[b': 'x' } })),
  ],
  skipped: await api.products.list({ fields: [null, 'a', undefined], filter: { status: undefined, kind: '', tag: null, ok: 'x' } }),
  notJSON: await caught(() => api.app.health()),
}
setBase('http://' + process.argv[3])
const network = await caught(() => api.app.health())
out.network = { ...network, message: network.message === '' ? '' : 'a message' }
console.log(JSON.stringify(out))
`
	os.WriteFile(filepath.Join(dir, "api.ts"), golden, 0o644)
	os.WriteFile(filepath.Join(dir, "run.mjs"), []byte(script), 0o644)
	cmd := exec.Command(node, "run.mjs", srv.URL, closed)
	cmd.Dir = dir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("node: %v\n%s", err, stderr.String())
	}
	var res map[string]any
	if err := json.Unmarshal(out, &res); err != nil {
		t.Fatalf("node printed %q: %v", out, err)
	}
	refused := func(name string) map[string]any {
		return map[string]any{"api": true, "status": 0.0, "code": "bad_request", "message": "The " + name + ` in the address cannot be empty, "." or "..".`, "fields": map[string]any{}, "cause": false}
	}
	badFilter := map[string]any{"api": true, "status": 0.0, "code": "bad_request", "message": "A filter name cannot be empty or contain [ or ].", "fields": map[string]any{}, "cause": false}
	wantRes := map[string]any{
		"list":        map[string]any{"items": []any{}, "total": 0.0, "page": 2.0, "per_page": 20.0},
		"update":      map[string]any{"id": 7.0, "name": "New"},
		"remove":      "undefined",
		"create":      map[string]any{"api": true, "status": 422.0, "code": "invalid", "message": "Check the fields.", "fields": map[string]any{"name": "This is already taken."}, "cause": false},
		"hello":       map[string]any{"api": true, "status": 500.0, "code": "http", "message": "Internal Server Error", "fields": map[string]any{}, "cause": false},
		"file":        map[string]any{"message": "file"},
		"note":        "undefined",
		"dots":        []any{refused("slug"), refused("slug"), refused("slug"), refused("slug")},
		"rest":        []any{refused("rest"), refused("rest"), refused("rest")},
		"filterNames": []any{badFilter, badFilter, badFilter},
		"skipped":     map[string]any{"items": []any{}, "total": 0.0, "page": 2.0, "per_page": 20.0},
		"notJSON":     map[string]any{"api": true, "status": 200.0, "code": "http", "message": "The answer was not JSON.", "fields": map[string]any{}, "cause": true},
		"network":     map[string]any{"api": true, "status": 0.0, "code": "network", "message": "a message", "fields": map[string]any{}, "cause": true},
	}
	g, _ := json.Marshal(res)
	w, _ := json.Marshal(wantRes)
	if string(g) != string(w) {
		t.Errorf("results\n got %s\nwant %s", g, w)
	}
	wantReq := []request{
		{"GET", "/api/products?page=2&q=a+b&fields=name&fields=price&full=false&filter%5Bstatus%5D=live", "", ""},
		{"PATCH", "/api/products/7", "application/json", `{"name":"New"}`},
		{"DELETE", "/api/products/7", "", ""},
		{"POST", "/api/products", "application/json", `{"name":"x"}`},
		{"GET", "/api/hello", "", ""},
		{"GET", "/api/hello/a%20b/files/x/y%20z?loud=true", "", ""},
		{"POST", "/api/products/7/notes", "application/json", `{"text":"hi"}`},
		// The refused calls sent nothing.
		{"GET", "/api/products?fields=a&filter%5Bok%5D=x", "", ""},
		{"GET", "/api/health", "", ""},
	}
	mu.Lock()
	defer mu.Unlock()
	if len(got) != len(wantReq) {
		t.Fatalf("requests %+v, want %+v", got, wantReq)
	}
	for i := range wantReq {
		if got[i] != wantReq[i] {
			t.Errorf("request %d: %+v, want %+v", i, got[i], wantReq[i])
		}
	}
}

// genApp copies the installed app into a new temp folder and runs the test
// in it.
func genApp(t *testing.T) string {
	t.Helper()
	if testing.Short() {
		t.Skip("weft gen builds and runs an app")
	}
	inTempDir(t, "work")
	app := copyApp(t, ".")
	t.Chdir(app)
	return app
}

// In a created app weft gen writes the client, with api.hello.hello; run
// again it leaves the file alone; a page using it type-checks, and stops
// type-checking once the Go field it reads is renamed. weft routes prints
// the route.
func TestGenInApp(t *testing.T) {
	app := genApp(t)
	file := filepath.Join(app, "web", ".weft-types", "api.ts")
	code, out, errs := runCmd("gen")
	if code != 0 || out != "Wrote web/.weft-types/api.ts\n" {
		t.Fatalf("exit %d, stdout %q, stderr %q", code, out, errs)
	}
	src, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"export interface Message {\n  message: string\n}\n", "  hello: {\n    /** GET /api/hello */\n    async hello(): Promise<Message> {\n"} {
		if !strings.Contains(string(src), want) {
			t.Errorf("api.ts has no %q:\n%s", want, src)
		}
	}
	before, _ := os.Stat(file)
	time.Sleep(20 * time.Millisecond)
	code, out, errs = runCmd("gen")
	if code != 0 || out != "web/.weft-types/api.ts is up to date\n" {
		t.Fatalf("again: exit %d, stdout %q, stderr %q", code, out, errs)
	}
	if after, _ := os.Stat(file); !after.ModTime().Equal(before.ModTime()) || !os.SameFile(before, after) {
		t.Error("a second weft gen wrote the file again")
	}

	code, out, errs = runCmd("routes")
	if want := "METHOD  PATH        MODULE  NAME   PERMISSION\nGET     /api/hello  hello   hello  -\n"; code != 0 || out != want {
		t.Errorf("weft routes: exit %d, stdout %q, stderr %q, want %q", code, out, errs, want)
	}

	page := "<script setup vapor lang=\"ts\">\nimport { api } from '../.weft-types/api.ts'\n\nconst text: string = (await api.hello.hello()).message\n</script>\n\n<template>\n  <p>{{ text }}</p>\n</template>\n"
	if err := os.WriteFile(filepath.Join(app, "web", "pages", "typed.vue"), []byte(page), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("pnpm", "typecheck").CombinedOutput(); err != nil {
		t.Fatalf("pnpm typecheck with the client: %v\n%s", err, out)
	}
	routes := filepath.Join(app, "app", "hello", "routes.go")
	goSrc, err := os.ReadFile(routes)
	if err != nil {
		t.Fatal(err)
	}
	renamed := strings.Replace(string(goSrc), "`json:\"message\"`", "`json:\"text\"`", 1)
	if renamed == string(goSrc) {
		t.Fatalf("app/hello/routes.go has no json:\"message\": %s", goSrc)
	}
	os.WriteFile(routes, []byte(renamed), 0o644)
	if code, out, errs := runCmd("gen"); code != 0 {
		t.Fatalf("gen after the rename: exit %d: %s%s", code, out, errs)
	}
	out2, err := exec.Command("pnpm", "typecheck").CombinedOutput()
	if err == nil || !strings.Contains(string(out2), "typed.vue") || !strings.Contains(string(out2), "message") {
		t.Errorf("pnpm typecheck passed a page reading a field the Go type no longer has: %v\n%s", err, out2)
	}
}

// An app that does not describe itself, whether it fails to compile, makes a
// registration mistake or never builds a warp.App, gets its own output and
// then one sentence, and the file is left as it was.
func TestGenReportsAppMistake(t *testing.T) {
	app := genApp(t)
	file := filepath.Join(app, "web", ".weft-types", "api.ts")
	if code, out, errs := runCmd("gen"); code != 0 {
		t.Fatalf("exit %d: %s%s", code, out, errs)
	}
	before, _ := os.ReadFile(file)
	const sentence = "weft gen: the app did not describe itself; fix what its output above says and run weft gen again.\n"
	routes := filepath.Join(app, "app", "hello", "routes.go")
	goSrc, _ := os.ReadFile(routes)
	main := filepath.Join(app, "main.go")
	mainSrc, _ := os.ReadFile(main)
	for _, c := range []struct {
		name, file, src, output, sentence string
	}{
		{"a registration mistake", routes, strings.Replace(string(goSrc), `"/hello"`, `"hello"`, 1), `has a path that does not start with /`, sentence},
		{"a compile error", routes, string(goSrc) + "\nfunc broken( {\n", "routes.go:", sentence},
		{"no warp.App", main, "package main\n\nimport \"fmt\"\n\nfunc main() { fmt.Println(\"just text\") }\n", "just text\n",
			"weft gen: the app's standard output was not its description (it starts with \"just text\\n\"); make sure nothing else prints there when WARP_DESCRIBE is set.\n"},
		{"nothing printed", main, "package main\n\nfunc main() {}\n", "", sentence},
	} {
		t.Run(c.name, func(t *testing.T) {
			os.WriteFile(c.file, []byte(c.src), 0o644)
			defer os.WriteFile(routes, goSrc, 0o644)
			defer os.WriteFile(main, mainSrc, 0o644)
			code, out, errs := runCmd("gen")
			if code != 1 || out != "" || !strings.HasSuffix(errs, c.sentence) || !strings.Contains(errs, c.output) {
				t.Errorf("exit %d, stdout %q, stderr %q, want %q and then %q", code, out, errs, c.output, c.sentence)
			}
			if after, _ := os.ReadFile(file); !bytes.Equal(after, before) {
				t.Error("api.ts changed")
			}
		})
	}
	// Stopped before it is done, it says nothing and exits 130.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var out, errb bytes.Buffer
	if code := gen(ctx, nil, &out, &errb); code != 130 || out.Len() != 0 || errb.Len() != 0 {
		t.Errorf("stopped: exit %d, stdout %q, stderr %q", code, out.String(), errb.String())
	}
}

// weft build writes the client before the page build, and a build of an app
// that does not describe itself fails before any page is built.
func TestBuildRunsGen(t *testing.T) {
	app := genApp(t)
	file := filepath.Join(app, "web", ".weft-types", "api.ts")
	if exists(t, file) {
		t.Fatal("the created app has an api.ts already")
	}
	code, out, errs := build(t)
	if code != 0 {
		t.Fatalf("exit %d: %s%s", code, out, errs)
	}
	src, err := os.ReadFile(file)
	if err != nil || !strings.Contains(string(src), "    async hello(): Promise<Message> {\n") {
		t.Fatalf("weft build wrote no client: %v\n%s", err, src)
	}

	app = genApp(t)
	main := "package main\n\nimport \"os\"\n\nfunc main() { os.Stderr.WriteString(\"no app here\\n\"); os.Exit(3) }\n"
	os.WriteFile(filepath.Join(app, "main.go"), []byte(main), 0o644)
	code, out, errs = build(t)
	want := "weft build: the app did not describe itself; fix what its output above says and run weft build again.\n"
	if code != 1 || !strings.HasSuffix(errs, want) || !strings.Contains(errs, "no app here") {
		t.Errorf("exit %d, stdout %q, stderr %q, want it to end %q", code, out, errs, want)
	}
	if exists(t, filepath.Join(app, "web", "dist")) {
		t.Error("the pages were built")
	}
}
