---
title: Go package
description: Everything the warp package exports, with its signature, and the root package's Version.
order: 37
---

Look up here everything the `warp` Go package exports today, with its signature and what it does. The packages beside it list theirs at the end of their guides: [data](../guides/data.md#everything-it-exports), [auth and authtest](../guides/login.md#everything-it-exports) and [warptest](../guides/warp.md#everything-warptest-exports). Import it as `github.com/flakerimi/weft/warp`. It uses only the standard library. [Warp, the Go side](../guides/warp.md) explains the app and [Typed routes](../guides/routes.md) its routes and errors, and [Serving from Go](../guides/go-serving.md) what `Serve` sends.

## App

```go
func New(cfg Config) *App
func (a *App) Config() Config
func (a *App) Use(modules ...Module)
func (a *App) Handle(pattern string, h http.Handler, opts ...RouteOption)
func (a *App) Wrap(mw func(http.Handler) http.Handler)
func (a *App) Err() error
func (a *App) Routes() []Route
func (a *App) Handler() http.Handler
func (a *App) Serve(dist fs.FS, opts ...ServeOption) error
func (a *App) Run(ctx context.Context, ln net.Listener, dist fs.FS, opts ...ServeOption) error
func (a *App) Start(ctx context.Context) error
func (a *App) Stop(ctx context.Context) error
func (a *App) Describe() (Description, error)
func (a *App) Mistake(format string, args ...any)
func (a *App) Module() string
func (a *App) ClientIP(r *http.Request) string
func (a *App) ClientKey(r *http.Request) string
func (a *App) IsHTTPS(r *http.Request) bool
```

- `New` makes an app configured by `cfg`, with no modules. The zero `Config` works: port 8080, a 1 MB body limit.
- `Config` returns the configuration, with its defaults filled in.
- `Use` calls each module's `Register` at once, in order. Two modules with one name, a `Register` that fails, and a `Use`, `Handle` or `Wrap` after the app started are mistakes; one made after the start is also logged, since `Run` has checked `Err` by then.
- `Handle` mounts `h` at `pattern`: a method, a space and a path without `/api` (`"GET /export.csv"`), or a path alone for every method.
- `Wrap` puts `mw` around every API request; the first given is the outermost.
- `Err` returns every mistake so far, the configuration's first, joined; nil when there is none. Each is one sentence; a registration mistake starts `warp: `, a configuration one does not.
- `Routes` returns every route, in the order registered. `Describe` returns them with every struct type they take and answer, each field with its JSON name, TypeScript type and rules, which `weft gen` reads; a type it cannot write as TypeScript, such as two types of one name, is a mistake it returns. Its types:

```go
type Description struct {
	Routes []RouteInfo `json:"routes"`
	Types  []TypeInfo  `json:"types"`
}
type RouteInfo struct {
	Module     string      `json:"module"`
	Name       string      `json:"name"`
	Method     string      `json:"method"`
	Path       string      `json:"path"` // with /api in front, as registered: /api/products/{id}
	Status     int         `json:"status"`
	Permission string      `json:"permission"`
	Typed      bool        `json:"typed"`
	PathParams []ParamInfo `json:"path_params"`
	Query      []ParamInfo `json:"query"`
	Body       string      `json:"body"`
	Partial    bool        `json:"partial"`
	Output     *OutputInfo `json:"output"`
}
type ParamInfo struct {
	Name     string     `json:"name"`
	TS       string     `json:"ts"`
	Optional bool       `json:"optional"`
	Style    string     `json:"style"`
	Rules    []RuleInfo `json:"rules"`
}
type OutputInfo struct {
	TS   string `json:"ts"`
	List bool   `json:"list"`
}
type TypeInfo struct {
	Name   string      `json:"name"`
	Go     string      `json:"go"`
	Fields []FieldInfo `json:"fields"`
}
type FieldInfo struct {
	Name     string     `json:"name"`
	TS       string     `json:"ts"`
	Optional bool       `json:"optional"`
	Readonly bool       `json:"readonly"`
	Rules    []RuleInfo `json:"rules"`
}
type RuleInfo struct {
	Name string `json:"name"`
	Arg  string `json:"arg"`
}
```
- `Handler` is the API alone, at `/api/...`: a path no route has is a 404, a method a route does not take a 405 with `Allow`, the redirect of a path that is not clean a 3xx with `Location`, and a panic a 500, all in the error shape. Every answer carries `Cache-Control: no-store` unless the handler sets its own.
- `Serve` returns `Err()` if there is a mistake. With `WARP_DESCRIBE=1` it prints `Describe` as JSON and returns, without migrating, starting or listening. Otherwise it starts the app, as `Start` does, then listens on `Host` and `Port`, logs `listening on http://localhost:<port>`, and serves as `Run` does until SIGINT or SIGTERM, so nothing answers on the port while the modules migrate. It returns nil after a clean stop. A taken port stops what started and is `warp: port 8080 is taken; stop what uses it or set PORT in .env.` Once the first signal comes it stops taking them, so a second Ctrl-C ends the process at once. Under `weft dev` (`WEFT_DEV_PARENT=1`) it logs nothing and also stops when standard input closes.
- `Run` does the same on `ln` until `ctx` ends: `Migrate` on every `Migrator`, then `Start` on every `Starter`, in order; it serves `dist` and the API through `Serve` (the API alone when `dist` is nil); when `ctx` ends it lets the requests in flight finish, for up to 10 seconds, then calls `Stop` on every `Stopper`, last first, with 10 seconds more. When a step fails, every module already migrated or started is stopped. A client has 10 seconds to send a request's headers. A nil `ln` is `warp: Run needs a listener; call Serve, or pass one from net.Listen.` With `WARP_DESCRIBE=1` it closes `ln` and does what `Serve` does then, and `Start` prints the description and ends the process, so a describing run never migrates or serves.
- `Start` runs every `Migrate`, then every `Start`, without listening, once; `Stop` stops what it started, last first, within 10 seconds. A module whose `Stop` has not returned when they are up is left running, and named: `warp: module "x" did not stop before the app's time to stop ran out, so the app went on without it; make its Stop return when its context ends.` A test that calls `Handler` uses them; `Run` is built on them.
- `Mistake` records a registration mistake found by a package outside the core, with `warp: ` in front, for `Err` to report. `Module` is the name of the module whose `Register` is running, `""` outside one.
- `ClientIP` is the client's address without its port, and `IsHTTPS` whether it came over HTTPS; with `TRUST_PROXY` on they read `X-Forwarded-For` and `X-Forwarded-Proto`. `ClientKey` is the address for IPv4 and its /64 for IPv6, for a limit to count by.

## Module

```go
type Module interface {
	Name() string
	Register(app *App) error
}
type Migrator interface{ Migrate(ctx context.Context) error }
type Starter interface{ Start(ctx context.Context) error }
type Stopper interface{ Stop(ctx context.Context) error }
```

A module registers its routes in `Register`. The three optional interfaces add a step: `Migrate` and `Start` before the app serves, `Stop` after it stops.

```go
func Provide[T any](app *App, value T)
func Need[T any](app *App, module string) T
```

`Provide` stores one value of type `T` for the app's lifetime, from a module's `Register`; `Need` returns it, or the zero `T` and a mistake saying to list `module` first. A second value of one type, or a `Provide` after the start, is a mistake.

## Typed routes

```go
func Get[In, Out any](app *App, path string, fn func(context.Context, In) (Out, error), opts ...RouteOption)
func Post[In, Out any](app *App, path string, fn func(context.Context, In) (Out, error), opts ...RouteOption)
func Put[In, Out any](app *App, path string, fn func(context.Context, In) (Out, error), opts ...RouteOption)
func Patch[In, Out any](app *App, path string, fn func(context.Context, In) (Out, error), opts ...RouteOption)
func Delete[In, Out any](app *App, path string, fn func(context.Context, In) (Out, error), opts ...RouteOption)

type None struct{}

func Request(ctx context.Context) *http.Request
func ResponseHeader(ctx context.Context) http.Header
func Sent(ctx context.Context, key string) bool
func SentKeys(ctx context.Context) []string
```

- Each registers `fn` at `/api` plus `path`. `In` is a struct whose fields are tagged `json`, `query` or `path`, or one field `body:"json"` that takes the whole body, or `None`; it is filled, checked by `Validate`, and given to `fn`. `Out` is answered as JSON: 200, 201 for `Post`, 204 with no body for `None`.
- `Post`, `Put` and `Patch` read a JSON body, at most `MaxBodyBytes`, sent as `application/json` or a `+json` type, and nested at most 64 levels (`The body is nested more than 64 levels deep.`). A wrong value names its field: `price: Send a number.`, `when: Send a time like 2026-11-25T10:00:00Z.`; `null` for a field that cannot be null is `Send a value, not null.`, and a fixed-size array of another length `Send a list of 3 values.` `Get` and `Delete` read none. Taking JSON alone keeps another site from sending a route a body; it does not protect a route that takes none.
- An input that embeds a pointer to a struct, or a struct with a `json` name or `json:"-"`, is a registration mistake.
- `Request` returns the request a typed handler is answering, or nil outside one. On `POST`, `PUT` and `PATCH` its body has already been read.
- `ResponseHeader` returns the headers its answer is sent with, whether it succeeds or fails.
- `Sent` reports whether the JSON body had a top-level key, and `SentKeys` lists them in the body's order. On `Patch` the rules run only on the fields sent, and the body type's own `Validate` method does not run.
- A field with the rule `readonly` is answered and never read, and zero after decoding whatever was sent; on an embedded struct, or inside a type that decodes itself, it is a mistake, and so are two keys of one object that differ only in case; an input or body type with a method `Validate(ctx context.Context) error` has it called once the tag rules pass. A `map[string]string` query field takes `key[name]=value`. An answer holds no `null` list, map or bytes.

## RouteOption

```go
type RouteOption func(*routeConfig)

func With(mw func(http.Handler) http.Handler) RouteOption
func Status(code int) RouteOption
func Name(name string) RouteOption
func Permission(permission string) RouteOption
func Options(opts ...RouteOption) RouteOption
func Requires(module, constructor string) RouteOption
```

- `With` puts `mw` around one route, inside every `Wrap`; the first given is the outermost.
- `Status` sets the status a typed route answers with, from 200 to 599; any other is a mistake. A route with an output and `Status(204)` or `Status(304)` answers 500 and logs why once.
- `Name` names the route within its module, in place of the name made from its method and path; two routes of one module with one name are a mistake.
- `Permission` records the permission the route needs, and guards nothing. `Options` makes one option of several, in order. `Requires` records that the route needs the module named `module`, made by `constructor`; an app that lists none of that name has a mistake, as `auth.Can` gives one.

## Route

```go
type Route struct {
	Module, Name, Method, Path string
	In, Out                    reflect.Type
	Status                     int
	Permission                 string
}
```

One registered route, as `Routes` lists it. `Module` is the module whose `Register` added it, empty outside one. `Path` has `/api` in front; `In` and `Out` are nil for a route mounted with `Handle`.

## Validate

```go
func Validate(v any) error
```

Checks `v`, a struct or a pointer to one, against the `warp` tags of its fields, and of the structs inside it: `required`, `email`, `url`, `min=N`, `max=N` and `enum=a|b`; `unique` is the store's, and `readonly` checks nothing. It returns nil, or an `*Error` from `Invalid` naming each field that fails, keyed as `address.city`, `lines.0.sku` or `by_name.x.sku`. A pointer that is not nil was sent: every rule applies to what it points at, even a zero, but `required` passes. At most 100 fields are listed, with 16 KiB of keys in all, counted before JSON escaping, so the answer stays under about 100 KB, and none longer than 1 KiB, in field, index and sorted key order; the walk stops when the 100 or the 16 KiB are spent. When one is left out the message is `More fields need attention than are listed; the first N are listed.`, or `More fields need attention than can be listed.` with no `fields`. A value with a pointer cycle is not supported. Typed routes call it on every input.

## Error

```go
type Error struct {
	Status  int               `json:"-"`
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

func (e *Error) Error() string

func BadRequest(message string) *Error       // 400 bad_request
func Unauthorized(message string) *Error     // 401 unauthorized
func Forbidden(message string) *Error        // 403 forbidden
func NotFound(message string) *Error         // 404 not_found
func Conflict(message string) *Error         // 409 conflict
func TooLarge(message string) *Error         // 413 too_large
func UnsupportedMedia(message string) *Error // 415 unsupported_media_type
func Invalid(fields map[string]string) *Error // 422 validation
func TooMany(message string) *Error          // 429 rate_limited
func Unavailable(message string) *Error      // 503 unavailable

func WriteJSON(w http.ResponseWriter, status int, v any)
func WriteError(w http.ResponseWriter, r *http.Request, err error)
```

- `Error` returns the message. `Invalid`'s message is `1 field needs attention.` or `N fields need attention.`
- `WriteJSON` writes `v` as JSON with `status`: a nil slice at the top, or one `v` points to, as `[]`, a nil one inside as `null`, and nothing for a 204. A value JSON cannot hold, such as a NaN or a channel, is logged and answered as the 500 below.
- `WriteError` writes `err` as `{"error":{...}}`. An `*Error`, even wrapped, is written as it is; any other error, and an `*Error` whose `Status` is not 400 to 599, is logged and answered 500, code `internal`, without its text.

## Config

```go
type Config struct {
	Host           string   // HOST; empty is every interface
	Port           string   // PORT; default "8080"
	MaxBodyBytes   int64    // MAX_BODY_MB x 1 MiB; default 1 MiB
	TrustProxy     bool     // TRUST_PROXY on: believe X-Forwarded-For and X-Forwarded-Proto
	URL            string   // APP_URL, the address people open the app at, without a trailing slash; default "http://localhost:" + Port
	TrustedOrigins []string // TRUSTED_ORIGINS, comma-separated: other sites whose browsers may make state-changing API requests
}

func FromEnv() Config
func ConfigFrom(vars map[string]string) Config
func (c Config) Get(key string) string
func (c Config) On(key string) bool
func (c Config) Err() error
```

- `FromEnv` reads `./.env` if it exists, then the environment, which wins.
- `ConfigFrom` does the same from a map, for tests and for apps that configure themselves.
- `Get` returns a key's value, `""` when it is unset; `On` reports whether it is `on`, `true`, `1` or `yes`, in any case.
- `Err` reports an `.env` that exists but cannot be read, every line of it that cannot be read, and every bad `PORT`, `MAX_BODY_MB`, `APP_URL` or entry of `TRUSTED_ORIGINS`, one sentence each. `New` carries it into the app's `Err`.

## Serve

```go
func Serve(dist fs.FS, api http.Handler, opts ...ServeOption) http.Handler
```

Returns the handler for a Weft app's pages and API: `/api` and what is under it go to `api` (a 404 in the error shape, `This app has no API.`, when it is nil), the rest to `dist`, a page build or a single-page app, read into memory once, here. [Serving from Go](../guides/go-serving.md) says what each request gets, its compression, caching and Early Hints. `App.Serve` and `App.Run` build it over the app's `Handler`.

It panics at start, not on a request, in one sentence that starts `warp.Serve:` and names the file, when the build is incomplete, `dist` has no `index.html`, or `dist` is nil. With `WARP_DESCRIBE=1` it prints the description of an app with no routes and exits 0, so an app that serves without a `warp.App` describes itself to `weft gen`; once an App in the process has a route, which it cannot see, it says `warp: Serve was called from the package and the app's routes are on an App; call the app's own Serve so they can be described.` and exits 1.

## ServeOption

```go
type ServeOption func(*options)
func WithFiles(prefix string, dir fs.FS) ServeOption
```

Changes what Serve serves; `WithFiles` is the one option today. It serves the files of `dir`, a folder that does not change while the server runs, under `prefix`, which starts and ends with `/`: `WithFiles("/releases/", d)` serves `d`'s `0.0.1/SHA256SUMS` at `/releases/0.0.1/SHA256SUMS`. Each file's ETag and type are read once at start, its bytes on each request; one changed since the start gets a 500, one added since is not served. [Files from disk](../guides/go-serving.md#files-from-disk) has the rest.

## Version

```go
var Version = "0.0.0-dev"
```

In the root package, `github.com/flakerimi/weft`, which holds nothing else. The Weft release this build belongs to. Release builds set it with `-ldflags "-X github.com/flakerimi/weft.Version=0.0.3"`.
