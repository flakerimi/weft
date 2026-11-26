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

const (
	// describeMax is the most a description is read of.
	describeMax = 8 << 20
	// shownMax is how much of an output that is not a description is shown.
	shownMax = 200
)

// outputError is an app whose standard output was not its description.
type outputError struct{ sentence string }

func (e outputError) Error() string { return e.sentence }

// cappedBuffer keeps what is written up to max bytes and calls over, once,
// when more comes. The buffer is a field, not embedded, so io.Copy cannot
// reach past Write through bytes.Buffer's ReadFrom.
type cappedBuffer struct {
	buf  bytes.Buffer
	max  int
	over func()
	full bool
}

func (c *cappedBuffer) Write(p []byte) (int, error) {
	if c.full {
		return len(p), nil
	}
	if c.buf.Len()+len(p) > c.max {
		c.full = true
		c.over()
		return len(p), nil
	}
	return c.buf.Write(p)
}

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
	ctx, stop := signal.NotifyContext(context.Background(), stopSignals...)
	defer stop()
	return gen(ctx, args, stdout, stderr)
}

// gen is weft gen until ctx is done.
func gen(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	appDir, code, ok := appCommand("gen", args, stdout, stderr)
	if !ok {
		return code
	}
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
	return appCommandWith(command, args, stdout, stderr, func(appDir string) error {
		_, err := ensure(appDir, stdout, stderr)
		return err
	})
}

// appCommandWith is appCommand with the step that gets the app ready.
func appCommandWith(command string, args []string, stdout, stderr io.Writer, prepare func(appDir string) error) (appDir string, code int, ok bool) {
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
	if err := prepare(appDir); err != nil {
		return "", fail(stderr, command, err), false
	}
	return appDir, 0, true
}

// genFailed says why weft gen's work failed, under the command's name, and
// returns 1; stopped by a signal, as weft dev is, it says nothing and
// returns 130.
func genFailed(stderr io.Writer, command string, err error) int {
	switch {
	case errors.Is(err, context.Canceled):
		return 130
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
// standard output, at most describeMax of it. Its standard error passes
// through; of a standard output that is not a description, the first
// shownMax bytes do too, so the app's own words come before weft's sentence.
func describeApp(ctx context.Context, appDir, bin string, within time.Duration, stderr io.Writer) (description, error) {
	run, cancel := context.WithTimeout(ctx, within)
	defer cancel()
	cmd := exec.CommandContext(run, bin)
	cmd.Dir = appDir
	cmd.Env = append(os.Environ(), "WARP_DESCRIBE=1")
	out := &cappedBuffer{max: describeMax, over: cancel}
	cmd.Stdout, cmd.Stderr = out, stderr
	cmd.WaitDelay = time.Second
	err := cmd.Run()
	show := func() {
		b := out.buf.Bytes()[:min(out.buf.Len(), shownMax)]
		if len(b) > 0 {
			stderr.Write(b)
			if b[len(b)-1] != '\n' {
				io.WriteString(stderr, "\n")
			}
		}
	}
	switch {
	case ctx.Err() != nil:
		return description{}, ctx.Err()
	case out.full:
		return description{}, outputError{"the app printed more than 8 MB where its description should be; make sure nothing but Warp writes to standard output when WARP_DESCRIBE is set"}
	case errors.Is(run.Err(), context.DeadlineExceeded):
		show()
		return description{}, errDescribeTimeout
	case err != nil:
		show()
		return description{}, errNoDescription
	case out.buf.Len() == 0:
		return description{}, errNoDescription
	}
	d, err := parseDescription(out.buf.Bytes())
	if err != nil {
		show()
		start := out.buf.Bytes()[:min(out.buf.Len(), 60)]
		for !utf8.Valid(start) {
			start = start[:len(start)-1]
		}
		return description{}, outputError{fmt.Sprintf("the app's standard output was not its description (it starts with %s); make sure nothing else prints there when WARP_DESCRIBE is set", strconv.Quote(string(start)))}
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
// Requests go to the pages' own origin, under what setBase names; an API on another origin than the pages is not supported yet.
`

const clientCode = `
/** Every rejection of an api function is an ApiError. From the server: its HTTP status and the error shape's code, message and fields. Without an answer: status 0, with code network when the request could not be sent, or bad_request when the arguments would reach another route or be misread. cause holds the error behind it. */
export class ApiError extends Error {
  status: number
  code: string
  fields: Record<string, string>

  constructor(status: number, code: string, message: string, fields: Record<string, string> = {}, cause?: unknown) {
    super(message, cause === undefined ? undefined : { cause })
    this.name = 'ApiError'
    this.status = status
    this.code = code
    this.fields = fields
  }
}

let base = ''

/** Puts url in front of every request's path: a path prefix or an absolute URL of the pages' own origin, or, for a script or a test that calls fetch outside a browser, the API's address. An API on another origin than the pages is not supported yet: a browser sends it no session cookie. */
export function setBase(url: string): void {
  base = url.replace(/\/+$/, '')
}

/** The ApiError for arguments that would reach another route, or that the server would misread; no request is sent. */
function refuse(message: string): ApiError {
  return new ApiError(0, 'bad_request', message)
}

/** One path parameter, encoded. One that is empty, "." or ".." (or only dots) would reach another route, so it is refused. */
function segment(name: string, value: string | number | boolean): string {
  const text = String(value)
  if (/^\.*$/.test(text)) throw refuse(` + "`" + `The ${name} in the address cannot be empty, "." or "..".` + "`" + `)
  return encodeURIComponent(text)
}

/** A {name...} path parameter: its segments, each encoded and checked as segment does. */
function segments(name: string, value: string | number | boolean): string {
  return String(value).split('/').map((part) => segment(name, part)).join('/')
}

/** The query string for query. An undefined or null value is left out, at the top and inside a list or a filter, and so is a filter whose value is ''; a list is repeated, and a filter map is written name[key]=value. */
function search(query: Record<string, string | number | boolean | (string | null | undefined)[] | Record<string, string | null | undefined> | null | undefined> = {}): string {
  const params = new URLSearchParams()
  for (const [name, value] of Object.entries(query)) {
    if (value === undefined || value === null) continue
    if (Array.isArray(value)) {
      for (const item of value) {
        if (item !== undefined && item !== null) params.append(name, item)
      }
    } else if (typeof value === 'object') {
      for (const [key, item] of Object.entries(value)) {
        if (key === '' || /[[\]]/.test(key)) throw refuse('A filter name cannot be empty or contain [ or ].')
        if (item !== undefined && item !== null && item !== '') params.append(` + "`" + `${name}[${key}]` + "`" + `, item)
      }
    } else {
      params.append(name, String(value))
    }
  }
  const text = params.toString()
  return text === '' ? '' : ` + "`" + `?${text}` + "`" + `
}

/** Sends one request, with body as JSON when given, and reads the answer as JSON unless the route answers nothing. */
async function send<T>(method: string, path: string, options: { body?: unknown; none?: boolean } = {}): Promise<T> {
  const init: { method: string; credentials: 'same-origin'; headers?: Record<string, string>; body?: string } = { method, credentials: 'same-origin' }
  if ('body' in options) {
    init.headers = { 'Content-Type': 'application/json' }
    init.body = JSON.stringify(options.body)
  }
  let res: Response
  try {
    res = await fetch(base + path, init)
  } catch (error) {
    throw new ApiError(0, 'network', error instanceof Error ? error.message : String(error), {}, error)
  }
  if (!res.ok) throw await failure(res)
  if (options.none || res.status === 204) return undefined as T
  try {
    return (await res.json()) as T
  } catch (error) {
    throw new ApiError(res.status, 'http', 'The answer was not JSON.', {}, error)
  }
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
  return new ApiError(res.status, 'http', res.statusText || String(res.status))
}
`

const fieldRulesCode = `
/** The rules of one field that @weft/ui's Form checks in the browser, and the kind of value it holds, so a form knows what control to draw. */
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
// itself, which an app's type may not have.
var ownTypeNames = map[string]bool{
	"ApiError": true, "FieldRules": true, "Partial": true, "Promise": true, "Record": true, "Response": true,
}

// reservedTypeNames are the names TypeScript refuses for an interface, or
// misreads where one is used as a type: ECMAScript's reserved words, those
// strict mode reserves, TypeScript's predefined types, and its type
// operators.
var reservedTypeNames = map[string]bool{}

func init() {
	for _, w := range strings.Fields(`await break case catch class const continue debugger default delete do else enum export
		extends false finally for function if import in instanceof new null return super switch this throw true try typeof var
		void while with yield
		implements interface let package private protected public static
		any bigint boolean never number object string symbol undefined unknown
		infer keyof readonly unique`) {
		reservedTypeNames[w] = true
	}
}

// objectMembers are the properties every JavaScript object has. TypeScript
// checks an optional property of one of these names against the object's
// own, so an optional field or a query parameter named so refuses every
// value; a required field of that name compiles, in a type's output or a whole body; a partial body makes every field optional.
var objectMembers = map[string]bool{
	"constructor": true, "toString": true, "toLocaleString": true, "valueOf": true, "hasOwnProperty": true,
	"isPrototypeOf": true, "propertyIsEnumerable": true,
}

// paramReserved are the names a path parameter cannot keep in TypeScript:
// JavaScript's reserved words, and the names a route's function uses inside
// it, which a parameter of that name would hide.
var paramReserved = map[string]bool{
	"body": true, "query": true, "send": true, "search": true, "segment": true, "segments": true, "refuse": true,
	"encodeURIComponent": true, "String": true,
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
	// A partial body makes every field optional, so a field named as an
	// Object member breaks there as an optional one does.
	partial := map[string]bool{}
	for _, r := range d.Routes {
		if r.Typed && r.Partial && r.Body != "" {
			partial[r.Body] = true
		}
	}
	for _, t := range d.Types {
		if ownTypeNames[t.Name] {
			return nil, fmt.Errorf("the type %s (%s) has a name the generated file uses itself; rename the Go type", t.Name, t.Go)
		}
		if reservedTypeNames[t.Name] {
			return nil, fmt.Errorf("the type %s (%s) has a name TypeScript reserves; rename the Go type", t.Name, t.Go)
		}
		for _, f := range t.Fields {
			if partial[t.Name] && objectMembers[f.Name] {
				return nil, fmt.Errorf("the field %s.%s has the name of a property every JavaScript object has, which TypeScript then cannot check where a route takes %s as a partial body; give it another json name", t.Name, f.Name, t.Name)
			}
			if f.Optional && objectMembers[f.Name] {
				return nil, fmt.Errorf("the optional field %s.%s has the name of a property every JavaScript object has, which TypeScript then cannot check; make it required or give it another json name", t.Name, f.Name)
			}
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

	b.WriteString("\n/** The app's API: a function for each route, by module and name. Every failure is an ApiError, including when nothing could be sent. */\n")
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
				if why, digit := notIdent(m.key); why != "" {
					fix := "rename the module to letters, digits and _"
					if digit {
						fix = "rename the module so it starts with a letter"
					}
					if m.key == "" {
						return nil, fmt.Errorf("module %q becomes nothing, since it is only _, - and .; %s", r.Module, fix)
					}
					return nil, fmt.Errorf("module %q becomes api.%s, which TypeScript cannot use as a name since %s; %s", r.Module, m.key, why, fix)
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
		if why, digit := notIdent(k); why != "" {
			fix := "give it a warp.Name of letters, digits and _"
			if digit {
				fix = "give it a warp.Name that starts with a letter"
			}
			if k == "" {
				return nil, fmt.Errorf("route %q %s becomes nothing, since it is only _, - and .; %s", r.Name, where, fix)
			}
			return nil, fmt.Errorf("route %q %s becomes %s, which TypeScript cannot use as a name since %s; %s", r.Name, where, k, why, fix)
		}
		if err := checkParams(r, where); err != nil {
			return nil, err
		}
		for _, q := range r.Query {
			if objectMembers[q.Name] {
				return nil, fmt.Errorf("route %q %s has the query parameter %s, the name of a property every JavaScript object has, which TypeScript then cannot check; give it another query name", r.Name, where, q.Name)
			}
		}
		if i := slices.Index(m.keys, k); i >= 0 {
			return nil, fmt.Errorf("routes %q and %q %s both become %s; give one a warp.Name", m.routes[i].Name, r.Name, where, k)
		}
		m.routes = append(m.routes, r)
		m.keys = append(m.keys, k)
	}
	return modules, nil
}

// wildcards returns the path's wildcards, in order, as {name} or {name...}
// are written: each one's name and whether it takes the rest of the path.
// {$} is left out.
func wildcards(path string) (names []string, rest []bool) {
	for {
		open := strings.IndexByte(path, '{')
		if open < 0 {
			return names, rest
		}
		end := strings.IndexByte(path[open:], '}')
		if end < 0 {
			return names, rest
		}
		wild := path[open+1 : open+end]
		path = path[open+end+1:]
		if wild == "$" {
			continue
		}
		names = append(names, strings.TrimSuffix(wild, "..."))
		rest = append(rest, strings.HasSuffix(wild, "..."))
	}
}

// paramIdent is a path parameter's name in TypeScript: as it is, with _
// after a name it cannot keep.
func paramIdent(name string) string {
	if paramReserved[name] || !isIdent(name) {
		return name + "_"
	}
	return name
}

// checkParams refuses a route two of whose path parameters become one name.
func checkParams(r routeInfo, where string) error {
	names, _ := wildcards(r.Path)
	seen := map[string]string{}
	for _, n := range names {
		id := paramIdent(n)
		if other, ok := seen[id]; ok {
			return fmt.Errorf("route %q %s has the path parameters {%s} and {%s}, which both become %s in TypeScript; rename one", r.Name, where, other, n, id)
		}
		seen[id] = n
	}
	return nil
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
		ident := paramIdent(wild)
		ts := types[wild]
		if ts == "" {
			ts = "string"
		}
		params = append(params, ident+": "+ts)
		hasParams = true
		if many {
			fmt.Fprintf(&path, "${segments(%s, %s)}", quote(wild), ident)
		} else {
			fmt.Fprintf(&path, "${segment(%s, %s)}", quote(wild), ident)
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
	fmt.Fprintf(b, "    async %s(%s): Promise<%s> {\n", name, strings.Join(params, ", "), out)
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
	b.WriteString("\n/** The rules of each type a route takes as its body, by the type's name and then the field's, for a form to check before it sends. unique, and every rule of a nested type, are checked by the server only, and arrive as its field errors. */\n")
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
		fmt.Fprintf(b, "  %s: {\n", literalKey(t.Name))
		for _, f := range t.Fields {
			if f.Readonly {
				continue
			}
			fmt.Fprintf(b, "    %s: { %s },\n", literalKey(f.Name), strings.Join(fieldRules(f), ", "))
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

// notIdent says why s is not a name in JavaScript, "" when it is one, and
// whether that is because it starts with a digit.
func notIdent(s string) (why string, digit bool) {
	if s == "" {
		return "it is empty", false
	}
	for i, r := range s {
		switch {
		case unicode.IsLetter(r) || r == '_' || r == '$':
		case unicode.IsDigit(r) && i == 0:
			return "it starts with a digit", true
		case unicode.IsDigit(r):
		case r == ' ':
			return "it has a space", false
		default:
			return "it has " + strconv.Quote(string(r)), false
		}
	}
	return "", false
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

// literalKey is a property name in an object literal: as key writes it,
// and __proto__, which there would set the object's prototype, as a computed
// key.
func literalKey(s string) string {
	if s == "__proto__" {
		return "['__proto__']"
	}
	return key(s)
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
