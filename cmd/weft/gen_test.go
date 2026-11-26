package main

import (
	"bytes"
	"encoding/json"
	"io"
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
		{"route not a name", description{Routes: []routeInfo{route("x", "2fa")}},
			`route "2fa" of module "x" becomes 2fa, which TypeScript cannot use as a name; give it a warp.Name that starts with a letter.`},
		{"module not a name", description{Routes: []routeInfo{route("3d", "list")}},
			`module "3d" becomes api.3d, which TypeScript cannot use as a name; rename the module so it starts with a letter.`},
		{"type the file uses", description{Types: []typeInfo{{Name: "Record", Go: "example.com/app/data.Record", Fields: []fieldInfo{}}}},
			`the type Record (example.com/app/data.Record) has a name the generated file uses itself; rename the Go type.`},
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
		}}},
	}
	got := mustGenerate(t, d)
	for _, want := range []string{
		"  'x-y': string\n  'it\\'s': number\n",
		"    show(class_: number, search_: string, query?: { 'sort-by'?: string }): Promise<(Thing | null)[]> {\n" +
			"      return send('GET', `/api/x/${encodeURIComponent(class_)}/${encodeURIComponent(search_)}/${search(query)}`)\n",
		"    count(): Promise<number> {\n      return send('GET', '/api/x/count')\n",
		"    putX(body_: string, body: Thing): Promise<Thing> {\n      return send('PUT', `/api/x/${encodeURIComponent(body_)}`, { body })\n",
		"    'x-y': { kind: 'string', required: true },\n    'it\\'s': { kind: 'number', min: 1.5, enum: ['a\\'b', 'c'] },\n",
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
	script := `import { api, setBase, ApiError } from './api.ts'
setBase(process.argv[2] + '/')
const caught = async (call) => {
  try {
    await call()
    return 'resolved'
  } catch (e) {
    return { api: e instanceof ApiError, status: e.status, code: e.code, message: e.message, fields: e.fields }
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
}
console.log(JSON.stringify(out))
`
	os.WriteFile(filepath.Join(dir, "api.ts"), golden, 0o644)
	os.WriteFile(filepath.Join(dir, "run.mjs"), []byte(script), 0o644)
	cmd := exec.Command(node, "run.mjs", srv.URL)
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
	wantRes := map[string]any{
		"list":   map[string]any{"items": []any{}, "total": 0.0, "page": 2.0, "per_page": 20.0},
		"update": map[string]any{"id": 7.0, "name": "New"},
		"remove": "undefined",
		"create": map[string]any{"api": true, "status": 422.0, "code": "invalid", "message": "Check the fields.", "fields": map[string]any{"name": "This is already taken."}},
		"hello":  map[string]any{"api": true, "status": 500.0, "code": "http", "message": "Internal Server Error", "fields": map[string]any{}},
		"file":   map[string]any{"message": "file"},
		"note":   "undefined",
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
	for _, want := range []string{"export interface Message {\n  message: string\n}\n", "  hello: {\n    /** GET /api/hello */\n    hello(): Promise<Message> {\n"} {
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
		name, file, src, output string
	}{
		{"a registration mistake", routes, strings.Replace(string(goSrc), `"/hello"`, `"hello"`, 1), `has a path that does not start with /`},
		{"a compile error", routes, string(goSrc) + "\nfunc broken( {\n", "routes.go:"},
		{"no warp.App", main, "package main\n\nimport \"fmt\"\n\nfunc main() { fmt.Println(\"just text\") }\n", "just text"},
		{"nothing printed", main, "package main\n\nfunc main() {}\n", ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			os.WriteFile(c.file, []byte(c.src), 0o644)
			defer os.WriteFile(routes, goSrc, 0o644)
			defer os.WriteFile(main, mainSrc, 0o644)
			code, out, errs := runCmd("gen")
			if code != 1 || out != "" || !strings.HasSuffix(errs, sentence) || !strings.Contains(errs, c.output) {
				t.Errorf("exit %d, stdout %q, stderr %q, want %q and then %q", code, out, errs, c.output, sentence)
			}
			if after, _ := os.ReadFile(file); !bytes.Equal(after, before) {
				t.Error("api.ts changed")
			}
		})
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
	if err != nil || !strings.Contains(string(src), "    hello(): Promise<Message> {\n") {
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
