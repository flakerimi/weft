---
title: Go package
description: Everything the warp package exports, with its signature, and the root package's Version.
order: 31
---

Look up here everything the `warp` Go package exports today, with its signature and what it does. Import it as `github.com/flakerimi/weft/warp`. It uses only the standard library. [Warp, the Go side](../guides/warp.md) explains the app, its routes and its errors, and [Serving from Go](../guides/go-serving.md) what `Serve` sends.

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
```

- `New` makes an app configured by `cfg`, with no modules. The zero `Config` works: port 8080, a 1 MB body limit.
- `Config` returns the configuration, with its defaults filled in.
- `Use` calls each module's `Register` at once, in order. Two modules with one name, a `Register` that fails, and a `Use`, `Handle` or `Wrap` after the app started are mistakes; one made after the start is also logged, since `Run` has checked `Err` by then.
- `Handle` mounts `h` at `pattern`: a method, a space and a path without `/api` (`"GET /export.csv"`), or a path alone for every method.
- `Wrap` puts `mw` around every API request; the first given is the outermost.
- `Err` returns every mistake so far, the configuration's first, joined; nil when there is none. Each is one sentence; a registration mistake starts `warp: `, a configuration one does not.
- `Routes` returns every route, in the order registered.
- `Handler` is the API alone, at `/api/...`: a path no route has is a 404, a method a route does not take a 405 with `Allow`, and a panic a 500, all in the error shape.
- `Serve` returns `Err()` if there is a mistake. Otherwise it listens on `Host` and `Port`, logs `listening on http://localhost:<port>`, and calls `Run` until SIGINT or SIGTERM. It returns nil after a clean stop. A taken port is `warp: port 8080 is taken; stop what uses it or set PORT in .env.` Once the first signal comes it stops taking them, so a second Ctrl-C ends the process at once. Under `weft dev` (`WEFT_DEV_PARENT=1`) it logs nothing and also stops when standard input closes.
- `Run` does the same on `ln` until `ctx` ends: `Migrate` on every `Migrator`, then `Start` on every `Starter`, in order; it serves `dist` and the API through `Serve` (the API alone when `dist` is nil); when `ctx` ends it lets the requests in flight finish, for up to 10 seconds, then calls `Stop` on every `Stopper`, last first, with 10 seconds more. When a step fails, the modules already started are stopped. A client has 10 seconds to send a request's headers. A nil `ln` is `warp: Run needs a listener; call Serve, or pass one from net.Listen.`

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
```

- Each registers `fn` at `/api` plus `path`. `In` is a struct whose fields are tagged `json`, `query` or `path`, or `None`; it is filled, checked by `Validate`, and given to `fn`. `Out` is answered as JSON: 200, 201 for `Post`, 204 with no body for `None`.
- `Post`, `Put` and `Patch` read a JSON body, at most `MaxBodyBytes`, sent as `application/json` or a `+json` type, and nested at most 64 levels (`The body is nested more than 64 levels deep.`). A value its field's own decoder refuses, such as `"tomorrow"` for a `time.Time`, is `The body has a value its field cannot take.` `Get` and `Delete` read none. Taking JSON alone keeps another site from sending a route a body; it does not protect a route that takes none.
- An input that embeds a pointer to a struct, or a struct with a `json` name or `json:"-"`, is a registration mistake.
- `Request` returns the request a typed handler is answering, or nil outside one. On `POST`, `PUT` and `PATCH` its body has already been read.
- `ResponseHeader` returns the headers its answer is sent with, whether it succeeds or fails.

## RouteOption

```go
type RouteOption func(*routeConfig)

func With(mw func(http.Handler) http.Handler) RouteOption
func Status(code int) RouteOption
```

- `With` puts `mw` around one route, inside every `Wrap`; the first given is the outermost.
- `Status` sets the status a typed route answers with, from 200 to 599; any other is a mistake.

## Route

```go
type Route struct {
	Method, Path string
	In, Out      reflect.Type
	Status       int
}
```

One registered route, as `Routes` lists it. `Path` has `/api` in front; `In` and `Out` are nil for a route mounted with `Handle`.

## Validate

```go
func Validate(v any) error
```

Checks `v`, a struct or a pointer to one, against the `warp` tags of its fields, and of the structs inside it: `required`, `email`, `url`, `min=N`, `max=N`, `enum=a|b` and `unique`. It returns nil, or an `*Error` from `Invalid` naming each field that fails, keyed as `address.city`, `lines.0.sku` or `by_name.x.sku`. A pointer that is not nil was sent: every rule applies to what it points at, even a zero, but `required` passes. At most 100 fields are listed, with 16 KiB of keys in all, counted before JSON escaping, so the answer stays under about 100 KB, and none longer than 1 KiB, in field, index and sorted key order; the walk stops when the 100 or the 16 KiB are spent. When one is left out the message is `More fields need attention than are listed; the first N are listed.`, or `More fields need attention than can be listed.` with no `fields`. A value with a pointer cycle is not supported. Typed routes call it on every input.

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

func WriteJSON(w http.ResponseWriter, status int, v any)
func WriteError(w http.ResponseWriter, r *http.Request, err error)
```

- `Error` returns the message. `Invalid`'s message is `1 field needs attention.` or `N fields need attention.`
- `WriteJSON` writes `v` as JSON with `status`: a nil slice as `[]`, and nothing for a 204. A value JSON cannot hold, such as a NaN or a channel, is logged and answered as the 500 below.
- `WriteError` writes `err` as `{"error":{...}}`. An `*Error`, even wrapped, is written as it is; any other error, and an `*Error` whose `Status` is not 400 to 599, is logged and answered 500, code `internal`, without its text.

## Config

```go
type Config struct {
	Host         string // HOST; empty is every interface
	Port         string // PORT; default "8080"
	MaxBodyBytes int64  // MAX_BODY_MB x 1 MiB; default 1 MiB
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
- `Err` reports an `.env` that exists but cannot be read, every line of it that cannot be read, and every bad `PORT` or `MAX_BODY_MB`, one sentence each. `New` carries it into the app's `Err`.

## Serve

```go
func Serve(dist fs.FS, api http.Handler, opts ...ServeOption) http.Handler
```

Returns the handler for a Weft app's pages and API. `App.Serve` and `App.Run` build it over the app's `Handler`.

- `/api` and the paths under `/api/` go to `api`. With `api` nil they get a 404 in the error shape, `This app has no API.`
- When `dist` holds a page build (`routes.json` and the `manifest.json` that `weft-build-pages` writes), every file is read into memory once, here. Each request gets a page, a listed file, the app's `index.html` under its prefix, or the not-found page with status 404. Each answer is the stored brotli, gzip or plain variant the client accepts best, with an `ETag`; `If-None-Match` gets a 304. Hashed files are cached for a year, the rest revalidated. A page with linked files gets a `103 Early Hints` first.
- Otherwise `dist` is a single-page app: its files are served as they are, and every other path gets its `index.html`. Files under `/assets/` are cached for a year; a missing one is a 404.

Serve panics, in one sentence that starts `warp.Serve:` and names the file, when the build is incomplete or `dist` has no `index.html`. It does that at start, not on a request.

## ServeOption

```go
type ServeOption func(*options)
```

Changes what Serve serves. `WithFiles` is the one option today.

## WithFiles

```go
func WithFiles(prefix string, dir fs.FS) ServeOption
```

Serves the files of `dir`, a folder that does not change while the server runs, under `prefix`. `WithFiles("/releases/", d)` serves `d`'s `0.0.1/SHA256SUMS` at `/releases/0.0.1/SHA256SUMS`. `prefix` starts and ends with `/`; `/` serves the files at their own paths.

- Each file is read once at start for its ETag, the first 16 hex digits of its SHA-256, and its type. Its bytes are read from disk on each request.
- A file changed since the start gets a 500; one added since is not served.
- Serve panics, naming the file, if `dir` holds a symlink or a name starting with `.`, or if a path is also served by the build, by another folder, or is under `/api/`.

Give it `os.OpenRoot(path)`'s `FS()`, so nothing put in the folder later leads out of it.

## Version

```go
var Version = "0.0.0-dev"
```

In the root package, `github.com/flakerimi/weft`, which holds nothing else. The Weft release this build belongs to. Release builds set it with `-ldflags "-X github.com/flakerimi/weft.Version=0.0.3"`.
