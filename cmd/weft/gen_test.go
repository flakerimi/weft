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
	"runtime"
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
		"variants_list":       "variantsList",
		"order-items_remove":  "orderItemsRemove",
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
	bin := filepath.Join(dir, "helper"+exeSuffix(runtime.GOOS))
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
		d, err := describeApp(context.Background(), "gen", filepath.Dir(bin), bin, 30*time.Second, &stderr)
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
  // What JSON cannot carry, or would write as null, is refused before sending.
  unsendable: [
    await caught(() => api.products.update(7, { name: 'x', price: NaN })),
    await caught(() => api.products.update(7, { price: -Infinity })),
    await caught(() => api.products.update(7, { lines: [{ qty: NaN }] })),
    await caught(() => api.products.update(7, { tags: ['a', undefined] })),
    await caught(() => api.products.update(7, { price: 10n })),
    await caught(() => { const o = {}; o.self = o; return api.products.update(7, o) }),
    await caught(() => api.products.update(NaN, { name: 'x' })),
    await caught(() => api.hello.delete('\ud800')),
    await caught(() => api.products.list({ page: NaN })),
  ],
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
	plain := func(message string) map[string]any {
		return map[string]any{"api": true, "status": 0.0, "code": "bad_request", "message": message, "fields": map[string]any{}, "cause": false}
	}
	wrong := func(field, says string) map[string]any {
		return map[string]any{"api": true, "status": 0.0, "code": "bad_request", "message": "1 field has the wrong type.", "fields": map[string]any{field: says}, "cause": false}
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
		"unsendable": []any{
			wrong("price", "Send a number."), wrong("price", "Send a number."), wrong("lines.qty", "Send a number."),
			wrong("tags", "Send a value, not null."), wrong("price", "Send a number."),
			map[string]any{"api": true, "status": 0.0, "code": "bad_request", "message": "The body cannot be written as JSON; its cause says why.", "fields": map[string]any{}, "cause": true},
			plain("The id in the address is NaN; give it a value."),
			plain("The slug in the address is not well-formed text."),
			plain("The page in the query is NaN; give it a value."),
		},
		"network": map[string]any{"api": true, "status": 0.0, "code": "network", "message": "a message", "fields": map[string]any{}, "cause": true},
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

// An app with a warp.App that does not describe itself, whether it fails to
// compile, makes a registration mistake, prints something else or nothing,
// gets its own output and then one sentence, and the file is left as it was.
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
		{"a compile error", routes, string(goSrc) + "\nfunc broken( {\n", "routes.go:", "weft gen: go build failed; fix what its output above says and run weft gen again.\n"},
		{"text printed", main, "package main\n\nimport (\n\t\"fmt\"\n\n\t\"github.com/flakerimi/weft/warp\"\n)\n\nfunc main() { _ = warp.New(warp.Config{}); fmt.Println(\"just text\") }\n", "just text\n",
			"weft gen: the app's standard output was not its description (it starts with \"just text\\n\"); make sure nothing else prints there when WARP_DESCRIBE is set.\n"},
		{"nothing printed", main, "package main\n\nimport \"github.com/flakerimi/weft/warp\"\n\nfunc main() { _ = warp.New(warp.Config{}) }\n", "", sentence},
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
// with a warp.App that exits with an error when asked to describe itself
// fails before any page is built.
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
	main := "package main\n\nimport (\n\t\"os\"\n\n\t\"github.com/flakerimi/weft/warp\"\n)\n\nfunc main() { _ = warp.New(warp.Config{}); os.Stderr.WriteString(\"no app here\\n\"); os.Exit(3) }\n"
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

// An app that makes a warp.App and does not describe itself does not stop
// weft build: one that never describes is given a free PORT and
// HOST=127.0.0.1, stopped when its time is up, and the build goes on; one
// that prints something else is said and the build goes on.
func TestBuildGoesOnWithoutDescription(t *testing.T) {
	app := genApp(t)
	file := filepath.Join(app, "web", ".weft-types", "api.ts")
	main := filepath.Join(app, "main.go")
	was := describeWithin
	describeWithin = 3 * time.Second
	t.Cleanup(func() { describeWithin = was })

	never := "package main\n\nimport (\n\t\"io/fs\"\n\t\"log\"\n\t\"net\"\n\t\"net/http\"\n\t\"os\"\n\n\t\"github.com/flakerimi/weft/warp\"\n)\n\nvar dist fs.FS\n\nfunc main() {\n\t_ = warp.New(warp.FromEnv())\n\tos.WriteFile(\"describe-env.txt\", []byte(os.Getenv(\"PORT\")+\" \"+os.Getenv(\"HOST\")), 0o644)\n\tln, err := net.Listen(\"tcp\", net.JoinHostPort(os.Getenv(\"HOST\"), os.Getenv(\"PORT\")))\n\tif err != nil {\n\t\tlog.Fatal(err)\n\t}\n\tlog.Fatal(http.Serve(ln, nil))\n}\n"
	os.WriteFile(main, []byte(never), 0o644)
	before, _ := os.ReadFile(file)
	code, out, errs := build(t)
	want := "weft build: the app did not describe itself within 3 s, so web/.weft-types/api.ts is as it was; make sure main.go calls Serve before doing other work.\n"
	if code != 0 || !strings.Contains(errs, want) || !exists(t, filepath.Join(app, "myapp"+exeSuffix(runtime.GOOS))) {
		t.Fatalf("never describes: exit %d, stdout %q, stderr %q, want %q and a binary", code, out, errs, want)
	}
	describedOn(t, app)
	if after, _ := os.ReadFile(file); !bytes.Equal(after, before) {
		t.Error("api.ts changed")
	}
	os.Remove(filepath.Join(app, "myapp"+exeSuffix(runtime.GOOS)))

	text := "package main\n\nimport (\n\t\"fmt\"\n\t\"io/fs\"\n\n\t\"github.com/flakerimi/weft/warp\"\n)\n\nvar dist fs.FS\n\nfunc main() { _ = warp.New(warp.Config{}); fmt.Println(\"just text\") }\n"
	os.WriteFile(main, []byte(text), 0o644)
	code, out, errs = build(t)
	want = "weft build: the app's standard output was not its description (it starts with \"just text\\n\"); make sure nothing else prints there when WARP_DESCRIBE is set.\nweft build: the app did not describe itself, so web/.weft-types/api.ts is as it was; fix what its output above says.\n"
	if code != 0 || !strings.Contains(errs, want) || !exists(t, filepath.Join(app, "myapp"+exeSuffix(runtime.GOOS))) {
		t.Fatalf("prints text: exit %d, stdout %q, stderr %q, want %q and a binary", code, out, errs, want)
	}
}

// An app that makes no warp.App, as a 0.0.6 app is after weft upgrade's
// two-line change (main.go listens itself, and under the dev tag never
// reaches warp.Serve), is not run to describe itself. Over a client that has
// routes, weft build leaves the file and says so once, and weft gen refuses;
// with no such client, weft build writes the client of an app with no
// routes, says nothing and does not wait, and weft gen and weft routes say
// why there are no routes. An app whose warp.New is in a package under app/
// still describes.
func TestNoWarpAppIsNotDescribed(t *testing.T) {
	app := genApp(t)
	file := filepath.Join(app, "web", ".weft-types", "api.ts")
	if code, out, errs := runCmd("gen"); code != 0 {
		t.Fatalf("the new app: exit %d: %s%s", code, out, errs)
	}
	withRoutes, _ := os.ReadFile(file)
	old := "package main\n\nimport (\n\t\"io/fs\"\n\t\"log\"\n\t\"net/http\"\n\t\"os\"\n\n\t\"github.com/flakerimi/weft/warp\"\n)\n\nvar dist fs.FS\n\nfunc handler() http.Handler {\n\tif dist != nil {\n\t\treturn warp.Serve(dist, http.NotFoundHandler())\n\t}\n\treturn http.NotFoundHandler()\n}\n\nfunc main() {\n\tif os.Getenv(\"WARP_DESCRIBE\") != \"\" {\n\t\tos.WriteFile(\"described.txt\", nil, 0o644)\n\t}\n\tlog.Fatal(http.ListenAndServe(\"127.0.0.1:\"+os.Getenv(\"PORT\"), handler()))\n}\n"
	if err := os.WriteFile(filepath.Join(app, "main.go"), []byte(old), 0o644); err != nil {
		t.Fatal(err)
	}
	kept := func(command string) string {
		return "weft " + command + ": the app has no warp.App, as no package it is built from calls warp.New, so web/.weft-types/api.ts, which has routes, was left as it was; if the app has routes, make sure its main package is built from one that calls warp.New, and if it has none, remove the file.\n"
	}
	code, out, errs := build(t)
	if code != 0 || errs != kept("build") {
		t.Errorf("over a client with routes: exit %d, stdout %q, stderr %q, want %q", code, out, errs, kept("build"))
	}
	code, out, errs = runCmd("gen")
	if code != 1 || errs != kept("gen") {
		t.Errorf("weft gen over a client with routes: exit %d, stdout %q, stderr %q, want %q", code, out, errs, kept("gen"))
	}
	if now, _ := os.ReadFile(file); !bytes.Equal(now, withRoutes) {
		t.Error("the client with routes was changed")
	}
	os.Remove(file)
	os.Remove(filepath.Join(app, "myapp"+exeSuffix(runtime.GOOS)))

	start := time.Now()
	code, out, errs = build(t)
	if code != 0 || errs != "" {
		t.Fatalf("exit %d, stdout %q, stderr %q", code, out, errs)
	}
	if took := time.Since(start); took > describeWithin*2/3 {
		t.Errorf("the build took %v", took)
	}
	if exists(t, filepath.Join(app, "described.txt")) {
		t.Error("the app was run to describe itself")
	}
	src, err := os.ReadFile(file)
	if err != nil || strings.Contains(string(src), "hello") || !strings.Contains(string(src), "export const schema = {}") || !strings.Contains(string(src), "export const api = {") {
		t.Fatalf("weft build wrote no empty client: %v\n%s", err, src)
	}
	if empty := mustGenerate(t, noRoutes); string(src) != empty {
		t.Errorf("the client is not that of an app with no routes:\n%s", firstDifference(string(src), empty))
	}
	os.Remove(filepath.Join(app, "myapp"+exeSuffix(runtime.GOOS)))

	code, out, errs = runCmd("gen")
	if want := "weft gen: the app has no warp.App, as no package it is built from calls warp.New, so there are no routes to write and web/.weft-types/api.ts is the client of an app with none.\n"; code != 0 || out != want || errs != "" {
		t.Errorf("weft gen: exit %d, stdout %q, stderr %q, want %q", code, out, errs, want)
	}
	code, out, errs = runCmd("routes")
	if want := "weft routes: the app has no warp.App, as no package it is built from calls warp.New, so there are no routes to print.\n"; code != 0 || out != want || errs != "" {
		t.Errorf("weft routes: exit %d, stdout %q, stderr %q, want %q", code, out, errs, want)
	}
	if exists(t, filepath.Join(app, "described.txt")) {
		t.Error("the app was run to describe itself")
	}

	// warp.New in a package under app/, which main.go imports.
	server := "package app\n\nimport \"github.com/flakerimi/weft/warp\"\n\n// Server is the app's warp.App with its modules.\nfunc Server() *warp.App {\n\ts := warp.New(warp.FromEnv())\n\ts.Use(Modules()...)\n\treturn s\n}\n"
	main := "package main\n\nimport (\n\t\"io/fs\"\n\t\"log\"\n\n\t\"myapp/app\"\n)\n\nvar dist fs.FS\n\nfunc main() {\n\tif err := app.Server().Serve(dist); err != nil {\n\t\tlog.Fatal(err)\n\t}\n}\n"
	os.WriteFile(filepath.Join(app, "app", "server.go"), []byte(server), 0o644)
	os.WriteFile(filepath.Join(app, "main.go"), []byte(main), 0o644)
	if code, out, errs := build(t); code != 0 {
		t.Fatalf("an App under app/: exit %d: %s%s", code, out, errs)
	}
	if src, _ := os.ReadFile(file); !strings.Contains(string(src), "    async hello(): Promise<Message> {\n") {
		t.Errorf("an App under app/ was not described:\n%s", src)
	}
}

// describedOn checks the PORT and HOST the describing run had, as the app
// wrote them to describe-env.txt: a port Warp's configuration takes, free at
// localhost and not the app's own 8080, and 127.0.0.1.
func describedOn(t *testing.T, app string) {
	t.Helper()
	env, _ := os.ReadFile(filepath.Join(app, "describe-env.txt"))
	port, host, _ := strings.Cut(string(env), " ")
	if n, err := strconv.Atoi(port); err != nil || n < 1 || n > 65535 || n == 8080 || host != "127.0.0.1" {
		t.Errorf("the describing run had PORT and HOST %q, want a free port and 127.0.0.1", env)
	}
}

// An app that starts its modules before it serves is described by Start,
// which starts nothing in a describing run (whose port is one Warp's
// configuration takes), whether it then serves through its own Serve or
// through the package-level warp.Serve. One that serves its App through
// warp.Serve without starting it, whose routes Serve cannot see, stops the
// build with Warp's sentence, not an empty client.
func TestBuildDescribesAnAppThatStartsFirst(t *testing.T) {
	app := genApp(t)
	head := "package main\n\nimport (\n\t\"context\"\n\t\"io/fs\"\n\t\"log\"\n\t\"net/http\"\n\t\"os\"\n\n\t\"github.com/flakerimi/weft/warp\"\n\n\t\"myapp/app\"\n)\n\nvar dist fs.FS\n\nvar _ http.Handler\n\nfunc main() {\n\tos.WriteFile(\"describe-env.txt\", []byte(os.Getenv(\"PORT\")+\" \"+os.Getenv(\"HOST\")), 0o644)\n\tserver := warp.New(warp.FromEnv())\n\tserver.Use(app.Modules()...)\n\tif err := server.Start(context.Background()); err != nil {\n\t\tlog.Fatal(err)\n\t}\n"
	starts := head + "\tif err := server.Serve(dist); err != nil {\n\t\tlog.Fatal(err)\n\t}\n}\n"
	if err := os.WriteFile(filepath.Join(app, "main.go"), []byte(starts), 0o644); err != nil {
		t.Fatal(err)
	}
	code, out, errs := build(t)
	if code != 0 || strings.Contains(errs, "PORT") || strings.Contains(errs, "describe") || !exists(t, filepath.Join(app, "myapp"+exeSuffix(runtime.GOOS))) {
		t.Fatalf("exit %d, stdout %q, stderr %q", code, out, errs)
	}
	describedOn(t, app)
	if src, _ := os.ReadFile(filepath.Join(app, "web", ".weft-types", "api.ts")); !strings.Contains(string(src), "    async hello(): Promise<Message> {\n") {
		t.Errorf("the app was not described:\n%s", src)
	}
	os.Remove(filepath.Join(app, "myapp"+exeSuffix(runtime.GOOS)))

	// Started first, then served through warp.Serve: Start describes it.
	tail := "\tif dist == nil {\n\t\tdist = os.DirFS(\".\")\n\t}\n\tlog.Fatal(http.ListenAndServe(\"127.0.0.1:\"+os.Getenv(\"PORT\"), warp.Serve(dist, server.Handler())))\n}\n"
	os.WriteFile(filepath.Join(app, "main.go"), []byte(head+tail), 0o644)
	if code, out, errs := build(t); code != 0 || strings.Contains(errs, "describe") {
		t.Fatalf("started, then warp.Serve: exit %d, stdout %q, stderr %q", code, out, errs)
	}
	if src, _ := os.ReadFile(filepath.Join(app, "web", ".weft-types", "api.ts")); !strings.Contains(string(src), "    async hello(): Promise<Message> {\n") {
		t.Errorf("started, then warp.Serve: the app was not described:\n%s", src)
	}
	os.Remove(filepath.Join(app, "myapp"+exeSuffix(runtime.GOOS)))

	// Not started, served through warp.Serve: its routes cannot be seen.
	pkg := strings.Replace(head, "\tif err := server.Start(context.Background()); err != nil {\n\t\tlog.Fatal(err)\n\t}\n", "\t_ = context.Background\n", 1) + tail
	os.WriteFile(filepath.Join(app, "main.go"), []byte(pkg), 0o644)
	code, out, errs = build(t)
	want := "warp: Serve was called from the package and the app's routes are on an App; call the app's own Serve so they can be described.\nweft build: the app did not describe itself; fix what its output above says and run weft build again.\n"
	if code != 1 || !strings.HasSuffix(errs, want) {
		t.Errorf("warp.Serve with an App: exit %d, stdout %q, stderr %q, want it to end %q", code, out, errs, want)
	}
}

// weft build type-checks the pages against the client it has just written, so
// a page reading a Go field that was renamed stops the build.
func TestBuildTypeChecks(t *testing.T) {
	app := genApp(t)
	page := "<script setup vapor lang=\"ts\">\nimport { api } from '../.weft-types/api.ts'\n\nasync function text(): Promise<string> {\n  return (await api.hello.hello()).message\n}\n</script>\n\n<template>\n  <p @click=\"text\">typed</p>\n</template>\n"
	if err := os.WriteFile(filepath.Join(app, "web", "pages", "typed.vue"), []byte(page), 0o644); err != nil {
		t.Fatal(err)
	}
	if code, out, errs := build(t); code != 0 {
		t.Fatalf("exit %d: %s%s", code, out, errs)
	}
	os.Remove(filepath.Join(app, "myapp"+exeSuffix(runtime.GOOS)))
	routes := filepath.Join(app, "app", "hello", "routes.go")
	goSrc, _ := os.ReadFile(routes)
	os.WriteFile(routes, []byte(strings.Replace(string(goSrc), "`json:\"message\"`", "`json:\"text\"`", 1)), 0o644)
	code, out, errs := build(t)
	want := "weft build: the app's TypeScript does not type-check; fix what vue-tsc says above and run weft build again.\n"
	if code != 1 || !strings.HasSuffix(errs, want) || !strings.Contains(out+errs, "typed.vue") {
		t.Errorf("exit %d, stdout %q, stderr %q, want it to end %q", code, out, errs, want)
	}
	if exists(t, filepath.Join(app, "myapp"+exeSuffix(runtime.GOOS))) {
		t.Error("the binary was built")
	}
}

// A second data.Resource in a module has its names prefixed with its last
// segment; they generate as one camelCase name each, beside the first's.
func TestGenerateSecondResource(t *testing.T) {
	var routes []routeInfo
	for _, name := range []string{"list", "show", "variants_list", "variants_show", "order-items_list"} {
		routes = append(routes, route("products", name))
	}
	src := mustGenerate(t, description{Routes: routes, Types: []typeInfo{}})
	for _, want := range []string{"async list(", "async show(", "async variantsList(", "async variantsShow(", "async orderItemsList("} {
		if !strings.Contains(src, want) {
			t.Errorf("api.ts has no %q", want)
		}
	}
}

// makesApp asks the Go tool what the app's main package is built from, with
// the dev tag, and finds a call of warp.New, under the name each file
// imports Warp by, in the files built: in the app, in a sibling module by
// replace, through a symlink, in a vendored or a nested module; not in a
// test, a comment, a string, a file a build tag leaves out, or a package the
// main package does not import. An app the Go tool cannot list, or a file
// of a package that imports Warp that does not parse, counts as making one,
// so the describing build reports it.
func TestMakesApp(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("no go")
	}
	repo, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	const imp = "\"github.com/flakerimi/weft/warp\""
	const old = "package main\n\nimport (\n\t\"net/http\"\n\n\t" + imp + "\n)\n\nfunc main() { http.ListenAndServe(\":8080\", warp.Serve(nil, nil)) }\n"
	const makes = "package main\n\nimport " + imp + "\n\nfunc main() { warp.New(warp.FromEnv()).Serve(nil) }\n"
	const wrapper = "package fw\n\nimport " + imp + "\n\n// NewApp makes the app.\nfunc NewApp() *warp.App { return warp.New(warp.Config{}) }\n"
	const usesFW = "package main\n\nimport \"example.com/fw\"\n\nfunc main() { fw.NewApp().Serve(nil) }\n"
	gomod := "module myapp\n\ngo 1.26\n\nrequire github.com/flakerimi/weft v0.0.0\n\nreplace github.com/flakerimi/weft => " + repo + "\n"
	withFW := func(dir string) string {
		return strings.Replace(gomod, "require github.com/flakerimi/weft v0.0.0\n", "require (\n\tgithub.com/flakerimi/weft v0.0.0\n\texample.com/fw v0.0.0\n)\n", 1) + "\nreplace example.com/fw => " + dir + "\n"
	}
	fwMod := "module example.com/fw\n\ngo 1.26\n\nrequire github.com/flakerimi/weft v0.0.0\n\nreplace github.com/flakerimi/weft => " + repo + "\n"
	outside := t.TempDir()
	os.MkdirAll(filepath.Join(outside, "server"), 0o755)
	os.WriteFile(filepath.Join(outside, "server", "server.go"), []byte("package server\n\nimport "+imp+"\n\nvar App = warp.New(warp.Config{})\n"), 0o644)
	sibling := t.TempDir()
	os.WriteFile(filepath.Join(sibling, "go.mod"), []byte(fwMod), 0o644)
	os.WriteFile(filepath.Join(sibling, "fw.go"), []byte(wrapper), 0o644)
	for _, c := range []struct {
		name  string
		mod   string
		files map[string]string
		link  string // a symlink app/server to outside/server
		want  bool
	}{
		{"a 0.0.6 app after the upgrade's change", gomod, map[string]string{"main.go": old, "api/api.go": "package api\n\nfunc Handler() {}\n"}, "", false},
		{"an App in main.go", gomod, map[string]string{"main.go": makes}, "", true},
		{"an App made under app/", gomod, map[string]string{"main.go": "package main\n\nimport \"myapp/app/server\"\n\nfunc main() { server.App.Serve(nil) }\n", "app/server/server.go": "package server\n\nimport " + imp + "\n\nvar App = warp.New(warp.Config{})\n"}, "", true},
		{"Warp under another name", gomod, map[string]string{"main.go": "package main\n\nimport w " + imp + "\n\nfunc main() { w.New(w.Config{}) }\n"}, "", true},
		{"Warp's names imported bare", gomod, map[string]string{"main.go": "package main\n\nimport . " + imp + "\n\nfunc main() { New(Config{}) }\n"}, "", true},
		{"Warp imported twice, once as _", gomod, map[string]string{"main.go": "package main\n\nimport (\n\t" + imp + "\n\t_ " + imp + "\n)\n\nfunc main() { warp.New(warp.Config{}) }\n"}, "", true},
		{"New taken as a value", gomod, map[string]string{"main.go": "package main\n\nimport " + imp + "\n\nvar make = warp.New\n\nfunc main() { _ = make }\n"}, "", true},
		{"a file of Warp's that does not parse", gomod, map[string]string{"main.go": "package main\n\nimport " + imp + "\n\nfunc main() {\n"}, "", true},
		{"an import the Go tool cannot find", gomod, map[string]string{"main.go": "package main\n\nimport \"example.com/gone\"\n\nfunc main() { gone.X() }\n"}, "", true},
		{"only a test", gomod, map[string]string{"main.go": old, "main_test.go": "package main\n\nimport " + imp + "\n\nvar _ = warp.New(warp.Config{})\n"}, "", false},
		{"a comment and a string", gomod, map[string]string{"main.go": "package main\n\n// warp.New(\nvar s = \"warp.New(\"\n\nfunc main() { _ = s }\n"}, "", false},
		{"a file a build tag leaves out", gomod, map[string]string{"main.go": old, "ignored.go": "//go:build ignore\n\npackage main\n\nimport " + imp + "\n\nvar _ = warp.New(warp.Config{})\n", "prod.go": "//go:build !dev\n\npackage main\n\nimport " + imp + "\n\nvar _ = warp.New(warp.Config{})\n"}, "", false},
		{"a file under the dev tag", gomod, map[string]string{"main.go": old, "dev.go": "//go:build dev\n\npackage main\n\nimport " + imp + "\n\nvar _ = warp.New(warp.Config{})\n"}, "", true},
		{"a package main does not import", gomod, map[string]string{"main.go": old, "tools/tools.go": "package tools\n\nimport " + imp + "\n\nvar _ = warp.New(warp.Config{})\n"}, "", false},
		{"a sibling module by replace", withFW(sibling), map[string]string{"main.go": usesFW}, "", true},
		{"a module nested in the app, by replace", withFW("./core"), map[string]string{"main.go": usesFW, "core/go.mod": fwMod, "core/fw.go": wrapper}, "", true},
		{"a symlinked package", gomod, map[string]string{"main.go": "package main\n\nimport \"myapp/server\"\n\nfunc main() { server.App.Serve(nil) }\n"}, "server", true},
	} {
		app := t.TempDir()
		os.WriteFile(filepath.Join(app, "go.mod"), []byte(c.mod), 0o644)
		for p, src := range c.files {
			os.MkdirAll(filepath.Dir(filepath.Join(app, p)), 0o755)
			if err := os.WriteFile(filepath.Join(app, p), []byte(src), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		if c.link != "" {
			if err := os.Symlink(filepath.Join(outside, "server"), filepath.Join(app, c.link)); err != nil {
				t.Fatal(err)
			}
		}
		if got := makesApp(app); got != c.want {
			t.Errorf("%s: makesApp = %v, want %v", c.name, got, c.want)
		}
	}

	// Vendored: the wrapper is a copy under vendor/, which go mod vendor writes.
	app := t.TempDir()
	os.WriteFile(filepath.Join(app, "go.mod"), []byte(withFW(sibling)), 0o644)
	os.WriteFile(filepath.Join(app, "main.go"), []byte(usesFW), 0o644)
	vendor := exec.Command("go", "mod", "vendor")
	vendor.Dir = app
	if out, err := vendor.CombinedOutput(); err != nil {
		t.Fatalf("go mod vendor: %v\n%s", err, out)
	}
	if !makesApp(app) {
		t.Error("vendored: makesApp = false, want true")
	}
}

// With no port free, the sentence names the command that is running, which
// may be weft build, gen or routes as well as weft dev, and so does the
// describing run's.
func TestFreePortNamesTheCommand(t *testing.T) {
	was := portFreeAt
	portFreeAt = func(int) bool { return false }
	t.Cleanup(func() { portFreeAt = was })
	for _, command := range []string{"dev", "build", "gen", "routes"} {
		want := "found no free port on localhost; stop some servers and run weft " + command + " again"
		if _, err := freePort(command); err == nil || err.Error() != want {
			t.Errorf("freePort(%q) = %v, want %q", command, err, want)
		}
	}
	var stderr bytes.Buffer
	if _, err := describeApp(context.Background(), "build", t.TempDir(), "none", time.Second, &stderr); err == nil || !strings.HasSuffix(err.Error(), "run weft build again") {
		t.Errorf("describeApp under weft build = %v", err)
	}
}

// A field that is a pointer in Go, which the description types T | null, is
// nullable in schema, so a Form sends null for it when it is emptied; any
// other field is not.
func TestSchemaSaysNullable(t *testing.T) {
	src := mustGenerate(t, fixture(t))
	for _, want := range []string{
		"    times: { kind: 'number', min: 1, nullable: true },\n",
		"    site: { kind: 'string', url: true, nullable: true },\n",
		"    price: { kind: 'number', min: 0 },\n",
		"  /** null is the field's empty value: a pointer in Go, so an emptied control sends null, which clears it. */\n  nullable?: boolean\n",
	} {
		if !strings.Contains(src, want) {
			t.Errorf("api.ts has no %q", want)
		}
	}
}

// A client has no routes when what it says is an empty api object, whoever
// wrote it: today's empty client, one from an older or a newer weft (other
// header text, CRLF line endings, another body of the helper code), one with
// a comment inside the braces. A client with a route, or a file that is not
// a client at all, has routes as far as weft can tell, so it is kept.
func TestClientHasRoutes(t *testing.T) {
	empty := mustGenerate(t, noRoutes)
	older := strings.Replace(empty, "class ApiError", "class ApiError /* older */", 1)
	if older == empty {
		t.Fatal("the empty client has no class ApiError to change")
	}
	for name, c := range map[string]struct {
		src    string
		routes bool
	}{
		"today's empty client":  {empty, false},
		"other header text":     {"// Written by an older weft; do not edit.\n" + strings.Replace(empty, "/**", "/** (old)", 1), false},
		"CRLF line endings":     {strings.ReplaceAll(empty, "\n", "\r\n"), false},
		"another helper body":   {older, false},
		"a comment in the api":  {strings.Replace(empty, "export const api = {}", "export const api = { /* none */\n  // yet\n}", 1), false},
		"one route":             {mustGenerate(t, fixture(t)), true},
		"not a client":          {"hello\n", true},
		"an empty file":         {"", true},
		"api only in a comment": {"// export const api = {}\nexport const api = { x: 1 }\n", true},
	} {
		if got := clientHasRoutes([]byte(c.src)); got != c.routes {
			t.Errorf("%s: clientHasRoutes = %v, want %v", name, got, c.routes)
		}
	}

	// An App-less app keeps a client with routes, and writes over an older
	// empty one.
	app := t.TempDir()
	file := filepath.Join(app, apiTypes)
	os.MkdirAll(filepath.Dir(file), 0o755)
	os.WriteFile(file, []byte(strings.ReplaceAll(older, "\n", "\r\n")), 0o644)
	if _, err := writeNoRoutes(app); !errors.Is(err, errNoWarpApp) || errors.Is(err, errKeptClient) {
		t.Errorf("an older empty client: %v, want it written over", err)
	}
	if got, _ := os.ReadFile(file); string(got) != empty {
		t.Error("the older empty client was not written over")
	}
	os.WriteFile(file, []byte("hello\n"), 0o644)
	if _, err := writeNoRoutes(app); !errors.Is(err, errKeptClient) {
		t.Errorf("a file that is not a client: %v, want it kept", err)
	}
}
