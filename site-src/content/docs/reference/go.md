---
title: Go package
description: Everything the warp package exports, with its signature, the signatures of data, auth and warptest, and the root package's Version.
order: 36
---

Look up here everything the `warp` Go package exports today, with its signature and what it does, and the signatures of the packages beside it. Import it as `github.com/flakerimi/weft/warp`. It uses only the standard library. [Warp, the Go side](../guides/warp.md) explains the app and [Typed routes](../guides/routes.md) its routes and errors, and [Serving from Go](../guides/go-serving.md) what `Serve` sends.

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
- `Routes` returns every route, in the order registered. `Describe` returns them with every struct type they take and answer, each field with its JSON name, TypeScript type and rules, which `weft gen` reads; a type it cannot write as TypeScript, such as two types of one name, is a mistake it returns.
- `Handler` is the API alone, at `/api/...`: a path no route has is a 404, a method a route does not take a 405 with `Allow`, and a panic a 500, all in the error shape.
- `Serve` returns `Err()` if there is a mistake. With `WARP_DESCRIBE=1` it prints `Describe` as JSON and returns, without migrating, starting or listening. Otherwise it listens on `Host` and `Port`, logs `listening on http://localhost:<port>`, and calls `Run` until SIGINT or SIGTERM. It returns nil after a clean stop. A taken port is `warp: port 8080 is taken; stop what uses it or set PORT in .env.` Once the first signal comes it stops taking them, so a second Ctrl-C ends the process at once. Under `weft dev` (`WEFT_DEV_PARENT=1`) it logs nothing and also stops when standard input closes.
- `Run` does the same on `ln` until `ctx` ends: `Migrate` on every `Migrator`, then `Start` on every `Starter`, in order; it serves `dist` and the API through `Serve` (the API alone when `dist` is nil); when `ctx` ends it lets the requests in flight finish, for up to 10 seconds, then calls `Stop` on every `Stopper`, last first, with 10 seconds more. When a step fails, every module already migrated or started is stopped. A client has 10 seconds to send a request's headers. A nil `ln` is `warp: Run needs a listener; call Serve, or pass one from net.Listen.`
- `Start` runs every `Migrate`, then every `Start`, without listening, once; `Stop` stops what it started, last first, within 10 seconds. A test that calls `Handler` uses them; `Run` is built on them.
- `Mistake` records a registration mistake found by a package outside the core, with `warp: ` in front, for `Err` to report.
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
- `Post`, `Put` and `Patch` read a JSON body, at most `MaxBodyBytes`, sent as `application/json` or a `+json` type, and nested at most 64 levels (`The body is nested more than 64 levels deep.`). A value its field's own decoder refuses, such as `"tomorrow"` for a `time.Time`, is `The body has a value its field cannot take.` `Get` and `Delete` read none. Taking JSON alone keeps another site from sending a route a body; it does not protect a route that takes none.
- An input that embeds a pointer to a struct, or a struct with a `json` name or `json:"-"`, is a registration mistake.
- `Request` returns the request a typed handler is answering, or nil outside one. On `POST`, `PUT` and `PATCH` its body has already been read.
- `ResponseHeader` returns the headers its answer is sent with, whether it succeeds or fails.
- `Sent` reports whether the JSON body had a top-level key, and `SentKeys` lists them in the body's order. On `Patch` the rules run only on the fields sent, and the body type's own `Validate` method does not run.
- A field with the rule `readonly` is answered and never read; an input or body type with a method `Validate(ctx context.Context) error` has it called once the tag rules pass. A `map[string]string` query field takes `key[name]=value`. An answer holds no `null` list, map or bytes.

## RouteOption

```go
type RouteOption func(*routeConfig)

func With(mw func(http.Handler) http.Handler) RouteOption
func Status(code int) RouteOption
func Name(name string) RouteOption
func Permission(permission string) RouteOption
func Options(opts ...RouteOption) RouteOption
```

- `With` puts `mw` around one route, inside every `Wrap`; the first given is the outermost.
- `Status` sets the status a typed route answers with, from 200 to 599; any other is a mistake.
- `Name` names the route within its module, in place of the name made from its method and path; two routes of one module with one name are a mistake.
- `Permission` records the permission the route needs, and guards nothing. `Options` makes one option of several, in order.

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
- `WriteJSON` writes `v` as JSON with `status`: a nil slice at the top as `[]`, a nil one inside as `null`, and nothing for a 204. A value JSON cannot hold, such as a NaN or a channel, is logged and answered as the 500 below.
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

## data, auth and warptest

The packages beside `warp`, each explained in its guide: [Data](../guides/data.md), [Login](../guides/login.md) and [testing](../guides/warp.md#testing-a-module).

```go
package data // github.com/flakerimi/weft/warp/data

func Module() warp.Module
func For(app *warp.App) *DB
func (db *DB) Tx(ctx context.Context, fn func(ctx context.Context) error) error
func (db *DB) Exec(ctx context.Context, query string, args ...any) (sql.Result, error)
func (db *DB) Query(ctx context.Context, query string, args ...any) (*sql.Rows, error)
func (db *DB) QueryRow(ctx context.Context, query string, args ...any) *Row
func (db *DB) SQL() *sql.DB
func (db *DB) SetClock(now func() time.Time)

type Record struct {
	ID        int64     `json:"id" warp:"readonly"`
	CreatedAt time.Time `json:"created_at" warp:"readonly"`
	UpdatedAt time.Time `json:"updated_at" warp:"readonly"`
}
type ID struct{ ID int64 `path:"id"` }
type Query struct {
	Page    int               `query:"page" warp:"min=1"`
	PerPage int               `query:"per_page" warp:"min=1,max=100"`
	Sort    string            `query:"sort"`
	Q       string            `query:"q" warp:"max=200"`
	Filter  map[string]string `query:"filter"`
}
type List[T any] struct {
	Items   []T `json:"items"`
	Total   int `json:"total"`
	Page    int `json:"page"`
	PerPage int `json:"per_page"`
}

func NewStore[T any](db *DB, table string) *Store[T]
func (s *Store[T]) List(ctx context.Context, q Query) (List[T], error)
func (s *Store[T]) Get(ctx context.Context, id int64) (T, error)
func (s *Store[T]) First(ctx context.Context, field string, value any) (v T, ok bool, err error)
func (s *Store[T]) Count(ctx context.Context) (int, error)
func (s *Store[T]) Create(ctx context.Context, v *T) error
func (s *Store[T]) Update(ctx context.Context, id int64, v *T, fields []string) (T, error)
func (s *Store[T]) Delete(ctx context.Context, id int64) error

func Resource[T any](app *warp.App, path string, store *Store[T], opts ...ResourceOption)
func Each(opts ...warp.RouteOption) ResourceOption
func PerAction(fn func(action string) []warp.RouteOption) ResourceOption
```

```go
package auth // github.com/flakerimi/weft/warp/auth

func Module() warp.Module
func Can(permission string) warp.RouteOption
func Per(resource string) func(action string) []warp.RouteOption
func UserOf(ctx context.Context) (User, bool)
func AddUser(ctx context.Context, app *warp.App, email, password, role string) (User, error)
func RemoveUser(ctx context.Context, app *warp.App, id int64) error
func SetupToken(app *warp.App) string

type User struct {
	data.Record
	Email string `json:"email" warp:"required,email,unique"`
	Role  string `json:"role"`
}

package authtest // github.com/flakerimi/weft/warp/auth/authtest

const Password = "the owner's password, for tests"
func Owner(t testing.TB, c *warptest.Client) auth.User
```

```go
package warptest // github.com/flakerimi/weft/warp/warptest

func New(t testing.TB, vars map[string]string, modules ...warp.Module) *Client
func (c *Client) Get(path string) *Response
func (c *Client) Post(path string, body any) *Response
func (c *Client) Patch(path string, body any) *Response
func (c *Client) Delete(path string) *Response
func (c *Client) Do(r *http.Request) *Response
func (c *Client) Fork() *Client
func (r *Response) Want(status int) *Response
func (r *Response) JSON(v any)
func (r *Response) Error() *warp.Error
```

`SetupToken` is the started app's setup token, for tools that cannot read the log; `""` once a user exists.

## Version

```go
var Version = "0.0.0-dev"
```

In the root package, `github.com/flakerimi/weft`, which holds nothing else. The Weft release this build belongs to. Release builds set it with `-ldflags "-X github.com/flakerimi/weft.Version=0.0.3"`.
