package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// apiTypes is the typed client weft gen writes from the app's description.
var apiTypes = filepath.Join("web", ".weft-types", "api.ts")

// describeWithin is how long the app's binary has to print its description.
var describeWithin = 30 * time.Second

var (
	// errNoDescription is an app that failed to build, exited with an
	// error, or printed something that is not a description; its own output
	// says why.
	errNoDescription = errors.New("the app did not describe itself")
	// errDescribeTimeout is an app still running after describeWithin, as
	// one whose main.go serves without a warp.App does.
	errDescribeTimeout = errors.New("the app did not describe itself in time")
)

// description is what an app built on warp prints when run with
// WARP_DESCRIBE=1: warp's Description, with only the fields weft reads. A
// field it does not know is ignored, so an app built with a newer warp still
// generates.
type description struct {
	Routes []routeInfo `json:"routes"`
	Types  []typeInfo  `json:"types"`
}

type routeInfo struct {
	Module     string      `json:"module"`
	Name       string      `json:"name"`
	Method     string      `json:"method"`
	Path       string      `json:"path"`
	Permission string      `json:"permission"`
	Typed      bool        `json:"typed"`
	PathParams []paramInfo `json:"path_params"`
	Query      []paramInfo `json:"query"`
	Body       string      `json:"body"`
	Partial    bool        `json:"partial"`
	Output     *outputInfo `json:"output"`
}

type paramInfo struct {
	Name string `json:"name"`
	TS   string `json:"ts"`
}

type outputInfo struct {
	TS   string `json:"ts"`
	List bool   `json:"list"`
}

type typeInfo struct {
	Name   string      `json:"name"`
	Go     string      `json:"go"`
	Fields []fieldInfo `json:"fields"`
}

type fieldInfo struct {
	Name     string     `json:"name"`
	TS       string     `json:"ts"`
	Optional bool       `json:"optional"`
	Readonly bool       `json:"readonly"`
	Rules    []ruleInfo `json:"rules"`
}

type ruleInfo struct {
	Name string `json:"name"`
	Arg  string `json:"arg"`
}

// parseDescription reads what an app printed: one JSON object with a
// "routes" list, and nothing else.
func parseDescription(b []byte) (description, error) {
	var probe struct {
		Routes json.RawMessage `json:"routes"`
	}
	if err := json.Unmarshal(b, &probe); err != nil {
		return description{}, err
	}
	if !bytes.HasPrefix(bytes.TrimSpace(probe.Routes), []byte("[")) {
		return description{}, errors.New(`it has no "routes" list`)
	}
	var d description
	err := json.Unmarshal(b, &d)
	return d, err
}

func genCmd(args []string, stdout, stderr io.Writer) int {
	appDir, code, ok := appCommand("gen", args, stdout, stderr)
	if !ok {
		return code
	}
	ctx, stop := signal.NotifyContext(context.Background(), stopSignals...)
	defer stop()
	wrote, err := genTypes(ctx, "gen", appDir, stderr)
	if err != nil {
		return genFailed(stderr, "gen", err)
	}
	if wrote {
		fmt.Fprintf(stdout, "Wrote %s\n", filepath.ToSlash(apiTypes))
	} else {
		fmt.Fprintf(stdout, "%s is up to date\n", filepath.ToSlash(apiTypes))
	}
	return 0
}

// appCommand is the start of a command that takes no argument and runs in an
// app: it parses args, finds the app and passes ensure. When ok is false the
// command returns code.
func appCommand(command string, args []string, stdout, stderr io.Writer) (appDir string, code int, ok bool) {
	usage := "usage is weft " + command + "."
	fl := flag.NewFlagSet(command, flag.ContinueOnError)
	fl.SetOutput(io.Discard)
	if code, done := flagsFailed(fl.Parse(args), command, usage, stdout, stderr); done {
		return "", code, false
	}
	if fl.NArg() > 0 {
		fmt.Fprintf(stderr, "weft %s: %q is not a flag; %s\n", command, fl.Arg(0), usage)
		return "", 2, false
	}
	appDir, err := findApp(".")
	if err != nil {
		fmt.Fprintf(stderr, "weft %s: %s\n", command, errNoApp)
		return "", 1, false
	}
	if _, err := ensure(appDir, stdout, stderr); err != nil {
		return "", fail(stderr, command, err), false
	}
	return appDir, 0, true
}

// genFailed says why weft gen's work failed, under the command's name, and
// returns 1.
func genFailed(stderr io.Writer, command string, err error) int {
	switch {
	case errors.Is(err, errDescribeTimeout):
		fmt.Fprintf(stderr, "weft %s: the app did not describe itself within %d s; check that main.go serves a warp.App, whose Serve prints the description under WARP_DESCRIBE=1, and run weft %s again.\n", command, int(describeWithin.Seconds()), command)
	case errors.Is(err, errNoDescription):
		fmt.Fprintf(stderr, "weft %s: the app did not describe itself; fix what its output above says and run weft %s again.\n", command, command)
	default:
		fail(stderr, command, err)
	}
	return 1
}

// devBinary is where weft dev, weft gen and weft build build the app's Go
// server with the dev tag.
func devBinary(appDir string) string { return filepath.Join(appDir, ".weft", "dev", "app") }

// genTypes builds the app with the dev tag, as weft dev does, and writes its
// typed client; it says whether the file changed.
func genTypes(ctx context.Context, command, appDir string, stderr io.Writer) (bool, error) {
	d, err := buildAndDescribe(ctx, command, appDir, stderr)
	if err != nil {
		return false, err
	}
	return writeAPI(appDir, d)
}

// buildAndDescribe builds the app with the dev tag and returns its description.
func buildAndDescribe(ctx context.Context, command, appDir string, stderr io.Writer) (description, error) {
	bin := devBinary(appDir)
	if !goBuild(ctx, command, appDir, bin, stderr) {
		if ctx.Err() != nil {
			return description{}, ctx.Err()
		}
		return description{}, errNoDescription
	}
	return describeApp(ctx, appDir, bin, describeWithin, stderr)
}

// regenerate describes the app built at bin and writes its typed client, for
// weft dev, which has just built it.
func regenerate(ctx context.Context, appDir, bin string, within time.Duration, stderr io.Writer) (bool, error) {
	d, err := describeApp(ctx, appDir, bin, within, stderr)
	if err != nil {
		return false, err
	}
	return writeAPI(appDir, d)
}

// describeApp runs bin in the app's folder with the environment and
// WARP_DESCRIBE=1, for at most within, and reads the description from its
// standard output. Its standard error passes through; its standard output
// does too when it is not a description, so the app's own words come
// before weft's sentence.
func describeApp(ctx context.Context, appDir, bin string, within time.Duration, stderr io.Writer) (description, error) {
	run, cancel := context.WithTimeout(ctx, within)
	defer cancel()
	cmd := exec.CommandContext(run, bin)
	cmd.Dir = appDir
	cmd.Env = append(os.Environ(), "WARP_DESCRIBE=1")
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, stderr
	cmd.WaitDelay = time.Second
	err := cmd.Run()
	switch {
	case ctx.Err() != nil:
		return description{}, ctx.Err()
	case errors.Is(run.Err(), context.DeadlineExceeded):
		stderr.Write(out.Bytes())
		return description{}, errDescribeTimeout
	case err != nil:
		stderr.Write(out.Bytes())
		return description{}, errNoDescription
	}
	d, err := parseDescription(out.Bytes())
	if err != nil {
		stderr.Write(out.Bytes())
		return description{}, errNoDescription
	}
	return d, nil
}

// writeAPI writes the client generated from d to the app's apiTypes, unless
// the file holds those bytes already, so the page dev server does not reload
// for nothing; it says whether it wrote. The file is written beside and
// renamed into place, so it is never read half-written.
func writeAPI(appDir string, d description) (bool, error) {
	src, err := generate(d)
	if err != nil {
		return false, err
	}
	file := filepath.Join(appDir, apiTypes)
	if old, err := os.ReadFile(file); err == nil && bytes.Equal(old, src) {
		return false, nil
	}
	dir := filepath.Dir(file)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return false, fmt.Errorf("cannot make %s (%v); make the app's folder writable", dir, err)
	}
	tmp, err := os.CreateTemp(dir, ".api.ts.*")
	if err != nil {
		return false, fmt.Errorf("cannot write in %s (%v); make it writable", dir, err)
	}
	_, err = tmp.Write(src)
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err == nil {
		err = os.Chmod(tmp.Name(), 0o644)
	}
	if err == nil {
		err = os.Rename(tmp.Name(), file)
	}
	if err != nil {
		os.Remove(tmp.Name())
		return false, fmt.Errorf("cannot write %s (%v); make it writable", file, err)
	}
	return true, nil
}

// generatedHeader starts the file; clientCode is the part of it that does
// not depend on the app: the error class and the function every route sends
// its request with.
const generatedHeader = `// Generated by weft gen from the description the app's Go code prints; do not edit it.
// weft dev writes it again after each Go change, and weft build before the pages.
`

const clientCode = `
/** A request the API answered with an error: its HTTP status, and the error shape's code, message and fields. */
export class ApiError extends Error {
  status: number
  code: string
  fields: Record<string, string>

  constructor(status: number, code: string, message: string, fields: Record<string, string>) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
    this.fields = fields
  }
}

let base = ''

/** Sends the requests to url, for an app whose pages are served from another origin than its API; by default they go to the page's own. */
export function setBase(url: string): void {
  base = url.replace(/\/+$/, '')
}

/** The query string for query: undefined and null are left out, a list is repeated, and a map is written name[key]=value. */
function search(query: Record<string, string | number | boolean | string[] | Record<string, string> | null | undefined> = {}): string {
  const params = new URLSearchParams()
  for (const [name, value] of Object.entries(query)) {
    if (value === undefined || value === null) continue
    if (Array.isArray(value)) {
      for (const item of value) params.append(name, item)
    } else if (typeof value === 'object') {
      for (const [key, item] of Object.entries(value)) params.append(` + "`${name}[${key}]`" + `, item)
    } else {
      params.append(name, String(value))
    }
  }
  const text = params.toString()
  return text === '' ? '' : ` + "`?${text}`" + `
}

/** Sends one request, with body as JSON when given, and reads the answer as JSON unless the route answers nothing. */
async function send<T>(method: string, path: string, options: { body?: unknown; none?: boolean } = {}): Promise<T> {
  const init: { method: string; credentials: 'same-origin'; headers?: Record<string, string>; body?: string } = { method, credentials: 'same-origin' }
  if ('body' in options) {
    init.headers = { 'Content-Type': 'application/json' }
    init.body = JSON.stringify(options.body)
  }
  const res = await fetch(base + path, init)
  if (!res.ok) throw await failure(res)
  if (options.none || res.status === 204) return undefined as T
  return (await res.json()) as T
}

/** The ApiError for an answer that is not a success: from the error shape, or from the status when the body is not that shape. */
async function failure(res: Response): Promise<ApiError> {
  const shape: unknown = await res.json().catch(() => undefined)
  const error = typeof shape === 'object' && shape !== null ? (shape as { error?: unknown }).error : undefined
  if (typeof error === 'object' && error !== null) {
    const { code, message, fields } = error as { code?: unknown; message?: unknown; fields?: unknown }
    if (typeof code === 'string' && typeof message === 'string') {
      return new ApiError(res.status, code, message, typeof fields === 'object' && fields !== null ? (fields as Record<string, string>) : {})
    }
  }
  return new ApiError(res.status, 'http', res.statusText || String(res.status), {})
}
`

const fieldRulesCode = `
/** The rules of one field, which @weft/ui's Form checks in the browser as the server does, and the kind of value it holds, so a form knows what control to draw. */
export interface FieldRules {
  kind: 'string' | 'number' | 'boolean' | 'list' | 'record' | 'object' | 'unknown'
  required?: boolean
  min?: number
  max?: number
  email?: boolean
  url?: boolean
  enum?: string[]
}
`

// ownTypeNames are the type names the generated file declares or uses
// itself, which an app's type may not have: TypeScript's own type keywords,
// and the global types the file names.
var ownTypeNames = map[string]bool{
	"ApiError": true, "FieldRules": true, "Partial": true, "Promise": true, "Record": true, "Response": true,
	"any": true, "unknown": true, "never": true, "string": true, "number": true, "boolean": true, "object": true,
	"symbol": true, "bigint": true, "undefined": true, "null": true, "void": true,
}

// paramReserved are the names a path parameter cannot keep in TypeScript:
// JavaScript's reserved words, and the names a route's function uses inside
// it, which a parameter of that name would hide.
var paramReserved = map[string]bool{
	"body": true, "query": true, "send": true, "search": true, "encodeURIComponent": true, "String": true,
	"arguments": true, "eval": true, "undefined": true, "await": true, "break": true, "case": true, "catch": true,
	"class": true, "const": true, "continue": true, "debugger": true, "default": true, "delete": true, "do": true,
	"else": true, "enum": true, "export": true, "extends": true, "false": true, "finally": true, "for": true,
	"function": true, "if": true, "implements": true, "import": true, "in": true, "instanceof": true,
	"interface": true, "let": true, "new": true, "null": true, "package": true, "private": true, "protected": true,
	"public": true, "return": true, "static": true, "super": true, "switch": true, "this": true, "throw": true,
	"true": true, "try": true, "typeof": true, "var": true, "void": true, "while": true, "with": true, "yield": true,
}

// apiModule is one module's routes as api has them.
type apiModule struct {
	name   string // as the description has it; "" for the routes outside one
	key    string // its name in api
	routes []routeInfo
	keys   []string
}

// generate writes the TypeScript client of an app from its description: an
// interface per type, api.<module>.<name>() for each typed route, and the
// rules of each type a route takes as its body. The same description gives
// the same bytes. A module or route name TypeScript cannot use, two that
// become one, and a type named as one the file uses itself are errors.
func generate(d description) ([]byte, error) {
	for _, t := range d.Types {
		if ownTypeNames[t.Name] {
			return nil, fmt.Errorf("the type %s (%s) has a name the generated file uses itself; rename the Go type", t.Name, t.Go)
		}
	}
	modules, err := apiModules(d.Routes)
	if err != nil {
		return nil, err
	}

	var b strings.Builder
	b.WriteString(generatedHeader)
	b.WriteString(clientCode)

	seenGo := map[string]bool{}
	for _, t := range d.Types {
		b.WriteString("\n")
		if seenGo[t.Go] {
			fmt.Fprintf(&b, "/** %s as a route takes it: without its readonly fields. */\n", comment(t.Go))
		} else {
			fmt.Fprintf(&b, "/** %s */\n", comment(t.Go))
		}
		seenGo[t.Go] = true
		if len(t.Fields) == 0 {
			fmt.Fprintf(&b, "export interface %s {}\n", t.Name)
			continue
		}
		fmt.Fprintf(&b, "export interface %s {\n", t.Name)
		for _, f := range t.Fields {
			opt := ""
			if f.Optional {
				opt = "?"
			}
			fmt.Fprintf(&b, "  %s%s: %s\n", key(f.Name), opt, f.TS)
		}
		b.WriteString("}\n")
	}

	b.WriteString("\n/** The app's API: a function for each route, by module and name. Each throws an ApiError when the answer is not a success. */\n")
	if len(modules) == 0 {
		b.WriteString("export const api = {}\n")
	} else {
		b.WriteString("export const api = {\n")
		for _, m := range modules {
			fmt.Fprintf(&b, "  %s: {\n", m.key)
			for i, r := range m.routes {
				writeRoute(&b, m.keys[i], r)
			}
			b.WriteString("  },\n")
		}
		b.WriteString("}\n")
	}

	b.WriteString(fieldRulesCode)
	writeSchema(&b, d)
	return []byte(b.String()), nil
}

// apiModules groups the typed routes by module, in the order each module
// first appears, and names each module and route as api has them.
func apiModules(routes []routeInfo) ([]*apiModule, error) {
	var modules []*apiModule
	byName := map[string]*apiModule{}
	byKey := map[string]*apiModule{}
	for _, r := range routes {
		if !r.Typed {
			continue
		}
		m := byName[r.Module]
		if m == nil {
			m = &apiModule{name: r.Module, key: "app"}
			if r.Module != "" {
				m.key = camel(r.Module)
				if !isIdent(m.key) {
					return nil, fmt.Errorf("module %q becomes api.%s, which TypeScript cannot use as a name; rename the module so it starts with a letter", r.Module, m.key)
				}
			}
			if other := byKey[m.key]; other != nil {
				first, second := other, m
				if second.name == "" {
					first, second = second, first
				}
				if first.name == "" {
					return nil, fmt.Errorf("the routes outside any module and module %q both become api.app; rename the module", second.name)
				}
				return nil, fmt.Errorf("modules %q and %q both become api.%s; rename one", other.name, m.name, m.key)
			}
			byName[r.Module], byKey[m.key] = m, m
			modules = append(modules, m)
		}
		where := fmt.Sprintf("of module %q", r.Module)
		if r.Module == "" {
			where = "outside any module"
		}
		k := camel(r.Name)
		if !isIdent(k) {
			return nil, fmt.Errorf("route %q %s becomes %s, which TypeScript cannot use as a name; give it a warp.Name that starts with a letter", r.Name, where, k)
		}
		if i := slices.Index(m.keys, k); i >= 0 {
			return nil, fmt.Errorf("routes %q and %q %s both become %s; give one a warp.Name", m.routes[i].Name, r.Name, where, k)
		}
		m.routes = append(m.routes, r)
		m.keys = append(m.keys, k)
	}
	return modules, nil
}

// writeRoute writes one route's function: its path parameters in path
// order, then its body, then its query as one object.
func writeRoute(b *strings.Builder, name string, r routeInfo) {
	types := map[string]string{}
	for _, p := range r.PathParams {
		types[p.Name] = p.TS
	}
	var params []string
	var path strings.Builder // a template literal's text
	hasParams := false
	rest := r.Path
	for {
		open := strings.IndexByte(rest, '{')
		end := strings.IndexByte(rest[max(open, 0):], '}')
		if open < 0 || end < 0 {
			path.WriteString(templateText(rest))
			break
		}
		end += open
		path.WriteString(templateText(rest[:open]))
		wild := rest[open+1 : end]
		rest = rest[end+1:]
		if wild == "$" {
			continue
		}
		many := strings.HasSuffix(wild, "...")
		wild = strings.TrimSuffix(wild, "...")
		ident := wild
		if paramReserved[ident] || !isIdent(ident) {
			ident += "_"
		}
		ts := types[wild]
		if ts == "" {
			ts = "string"
		}
		params = append(params, ident+": "+ts)
		hasParams = true
		if many {
			fmt.Fprintf(&path, "${String(%s).split('/').map(encodeURIComponent).join('/')}", ident)
		} else {
			fmt.Fprintf(&path, "${encodeURIComponent(%s)}", ident)
		}
	}
	if r.Body != "" {
		if r.Partial {
			params = append(params, "body: Partial<"+r.Body+">")
		} else {
			params = append(params, "body: "+r.Body)
		}
	}
	if len(r.Query) > 0 {
		fields := make([]string, len(r.Query))
		for i, q := range r.Query {
			fields[i] = key(q.Name) + "?: " + q.TS
		}
		params = append(params, "query?: { "+strings.Join(fields, "; ")+" }")
		path.WriteString("${search(query)}")
		hasParams = true
	}

	out := "void"
	if r.Output != nil {
		out = r.Output.TS
		if r.Output.List {
			if strings.Contains(out, " | ") {
				out = "(" + out + ")"
			}
			out += "[]"
		}
	}
	pathExpr := quote(r.Path)
	if hasParams {
		pathExpr = "`" + path.String() + "`"
	}
	var options []string
	if r.Body != "" {
		options = append(options, "body")
	}
	if r.Output == nil {
		options = append(options, "none: true")
	}
	call := fmt.Sprintf("send(%s, %s", quote(r.Method), pathExpr)
	if len(options) > 0 {
		call += ", { " + strings.Join(options, ", ") + " }"
	}
	call += ")"

	doc := r.Method + " " + r.Path
	if r.Permission != "" {
		doc += " (" + r.Permission + ")"
	}
	fmt.Fprintf(b, "    /** %s */\n", comment(doc))
	fmt.Fprintf(b, "    %s(%s): Promise<%s> {\n", name, strings.Join(params, ", "), out)
	fmt.Fprintf(b, "      return %s\n", call)
	b.WriteString("    },\n")
}

// writeSchema writes the rules of each type some route takes as its body, in
// the description's order of types.
func writeSchema(b *strings.Builder, d description) {
	bodies := map[string]bool{}
	for _, r := range d.Routes {
		if r.Typed && r.Body != "" {
			bodies[r.Body] = true
		}
	}
	var types []typeInfo
	for _, t := range d.Types {
		if bodies[t.Name] {
			types = append(types, t)
			delete(bodies, t.Name)
		}
	}
	b.WriteString("\n/** The rules of each type a route takes as its body, by the type's name and then the field's. */\n")
	if len(types) == 0 {
		b.WriteString("export const schema = {}\n")
		return
	}
	b.WriteString("export const schema: {\n")
	for _, t := range types {
		fmt.Fprintf(b, "  %s: { [K in keyof %s]-?: FieldRules }\n", key(t.Name), t.Name)
	}
	b.WriteString("} = {\n")
	for _, t := range types {
		fmt.Fprintf(b, "  %s: {\n", key(t.Name))
		for _, f := range t.Fields {
			if f.Readonly {
				continue
			}
			fmt.Fprintf(b, "    %s: { %s },\n", key(f.Name), strings.Join(fieldRules(f), ", "))
		}
		b.WriteString("  },\n")
	}
	b.WriteString("}\n")
}

// fieldRules is a field's entry in schema: its kind, then the rules Form
// checks, in the tag's order. unique, which only the database can check,
// and a rule weft does not know are left out.
func fieldRules(f fieldInfo) []string {
	entries := []string{"kind: " + quote(kind(f.TS))}
	for _, r := range f.Rules {
		switch r.Name {
		case "required", "email", "url":
			entries = append(entries, r.Name+": true")
		case "min", "max":
			if n, err := strconv.ParseFloat(r.Arg, 64); err == nil {
				entries = append(entries, r.Name+": "+strconv.FormatFloat(n, 'f', -1, 64))
			}
		case "enum":
			values := strings.Split(r.Arg, "|")
			for i, v := range values {
				values[i] = quote(v)
			}
			entries = append(entries, "enum: ["+strings.Join(values, ", ")+"]")
		}
	}
	return entries
}

// kind is the kind of value a field of TypeScript type ts holds, for a form
// to choose its control.
func kind(ts string) string {
	ts = strings.TrimSuffix(ts, " | null")
	switch {
	case ts == "string", ts == "number", ts == "boolean", ts == "unknown":
		return ts
	case strings.HasSuffix(ts, "[]"):
		return "list"
	case strings.HasPrefix(ts, "Record<"):
		return "record"
	}
	return "object"
}

// camel writes a module or route name in camelCase: the parts between _, -
// and . joined, each after the first with a capital.
func camel(name string) string {
	parts := strings.FieldsFunc(name, func(r rune) bool { return r == '_' || r == '-' || r == '.' })
	for i := 1; i < len(parts); i++ {
		r, size := utf8.DecodeRuneInString(parts[i])
		parts[i] = string(unicode.ToUpper(r)) + parts[i][size:]
	}
	return strings.Join(parts, "")
}

// isIdent reports whether s is a name in JavaScript: a letter, _ or $, then
// letters, digits, _ and $. A reserved word is one too, as a property name.
func isIdent(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		if !(unicode.IsLetter(r) || r == '_' || r == '$' || i > 0 && unicode.IsDigit(r)) {
			return false
		}
	}
	return true
}

// key is a property name: as it is when it is a name, quoted otherwise.
func key(s string) string {
	if isIdent(s) {
		return s
	}
	return quote(s)
}

// quote writes s as a TypeScript string in single quotes.
func quote(s string) string {
	var b strings.Builder
	b.WriteByte('\'')
	for _, r := range s {
		switch {
		case r == '\\' || r == '\'':
			b.WriteByte('\\')
			b.WriteRune(r)
		case r == '\n':
			b.WriteString(`\n`)
		case r < 0x20 || r == 0x2028 || r == 0x2029:
			fmt.Fprintf(&b, `\u%04x`, r)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('\'')
	return b.String()
}

// templateText writes s as the text of a template literal.
func templateText(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, "`", "\\`")
	return strings.ReplaceAll(s, "${", `\${`)
}

// comment makes s safe inside a /** */ comment.
func comment(s string) string { return strings.ReplaceAll(s, "*/", `*\/`) }
