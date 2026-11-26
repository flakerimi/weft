---
title: Warp, the Go side
description: An app and its modules, what a module needs from another, configuration from .env, a proxy in front, cross-site requests, and testing.
order: 22
---

On this page you learn how the Go half of an app is put together: the app and its modules, how one module gets what another provides, the configuration in `.env`, running behind a proxy, the cross-site check, and testing. Warp is the package `github.com/flakerimi/weft/warp`, and uses only Go's standard library. Its routes, their input and their errors are in [Typed routes](routes.md); what it does with the built pages is in [Serving from Go](go-serving.md).

## An app and its modules

The `main.go` that `weft create` writes makes the app, lists its modules and serves it:

```go
// Command myapp serves the app: the Go API and the built pages, in one binary.
package main

import (
	"io/fs"
	"log"

	"github.com/flakerimi/weft/warp"

	"myapp/app"
)

// dist is the built pages. It is nil under the dev tag, which weft dev uses so
// the program compiles before any build; weft dev serves the pages itself.
var dist fs.FS

func main() {
	server := warp.New(warp.FromEnv())
	server.Use(app.Modules()...)
	// Serve returns nil once it was asked to stop: that is a clean exit.
	if err := server.Serve(dist); err != nil {
		log.Fatal(err)
	}
}
```

- `warp.New(warp.FromEnv())` makes the app, configured from `.env` and the environment (see [Configuration](#configuration)).
- `Use` registers each module at once, in the order given.
- `Serve` listens on `HOST` and `PORT`, serves the pages in `dist` and the API under `/api`, and runs until the process gets SIGINT or SIGTERM. With `dist` nil it serves the API alone.

The modules are listed in one place, `app/modules.go`. Nothing registers itself from an `init` function, so this list is the whole app. The example shop's:

```go
// Modules is the one list of the app's modules, in the order they start: the
// database first, since the login and the products keep their records in it.
func Modules() []warp.Module {
	return []warp.Module{
		data.Module(),
		auth.Module(),
		products.Module(),
	}
}
```

To add a module, make a folder `app/<name>/` with a `Module()` function and add it to this list. [Data](data.md) and [Login](login.md) are the two modules Warp has.

## A module

A module has a name and registers its routes on the app it is given:

```go
type Module interface {
	Name() string
	Register(app *App) error
}
```

The template's `app/hello/routes.go` is one, with the route `GET /api/hello`. Two modules may not have the same name. An error from `Register` is reported as `warp: module "hello" could not register: <the error>`. `Register` does no file, network or database work: that is for the steps below, so `weft gen` can register an app without opening anything.

A module may also have any of three more methods, each a step of the app's life:

| Interface | Method | When it runs |
| --- | --- | --- |
| `warp.Migrator` | `Migrate(ctx context.Context) error` | First, before anything starts, in the order the modules are listed |
| `warp.Starter` | `Start(ctx context.Context) error` | Next, before the app serves, in the same order |
| `warp.Stopper` | `Stop(ctx context.Context) error` | Last, after the app has stopped serving, in reverse order |

When the app is told to stop, it takes no new requests and lets the ones in flight finish, for up to 10 seconds. Then it stops the modules, last first, with 10 seconds more for them. When a step fails, every module that already migrated or started is stopped, last first, and `Serve` returns the error, such as `warp: module "jobs" could not start: <the error>`.

A module with work of its own while the app runs:

```go
type module struct{ stop context.CancelFunc }

func (*module) Name() string                 { return "jobs" }
func (*module) Register(app *warp.App) error { return nil }

func (m *module) Start(ctx context.Context) error {
	work, stop := context.WithCancel(context.Background())
	m.stop = stop
	go run(work)
	return nil
}

func (m *module) Stop(ctx context.Context) error {
	m.stop()
	return nil
}
```

`run` is the module's work, which returns when `work` is cancelled.

## What a module needs

A value that lives as long as the app, such as the database, is provided by one module and taken by the others, by its Go type:

```go
warp.Provide(app, db)                 // in the data module's Register
db := warp.Need[*data.DB](app, "data") // in a later module's Register
```

A module wraps `Need` in a function of its own, so app code reads `data.For(app)`. When nothing of that type was provided, `Need` returns the zero value and the app reports `warp: module "products" needs the "data" module; list data.Module() before it in app/modules.go.` Two values of one type, or a `Provide` after the app started, are mistakes too. What belongs to one request, such as the signed-in user, is on the handler's context instead, behind a function of the module that put it there: `auth.UserOf(ctx)`.

## Configuration

`warp.FromEnv()` reads `.env` in the folder the app runs in, if there is one, then the environment, which wins over the file. Warp reads these keys:

| Key | What it is | Default |
| --- | --- | --- |
| `PORT` | The port to listen on, 1 to 65535 | `8080` |
| `HOST` | The address to listen on | empty: every interface |
| `MAX_BODY_MB` | The largest body a typed route reads, in whole megabytes, 1 or more | `1` |
| `APP_URL` | The address people open the app at: `http` or `https` and a host, nothing after it. The login's setup link starts with it | `http://localhost:` and the port |
| `TRUST_PROXY` | `on` to believe the proxy in front of the app (see below) | off |
| `TRUSTED_ORIGINS` | Other sites, comma-separated, whose pages may make state-changing API requests (see [Cross-site requests](#cross-site-requests)) | none |

A module reads its own keys with `app.Config().Get("KEY")`, which is `""` when it is unset, and `app.Config().On("KEY")`, which is true for `on`, `true`, `1` and `yes`, in any case. [Data](data.md) and [Login](login.md) list theirs.

`.env` has one `KEY=value` per line. Blank lines and lines that start with `#` are skipped, and a leading `export` and space or tab is dropped. A value in single or double quotes is what is between them; an unquoted value is trimmed and ends at ` #`. There is no `${}` expansion. A line it cannot read, an `.env` it cannot open, or a value Warp cannot take stops the app with one sentence, such as `PORT is "abc"; set it to a number from 1 to 65535 in .env or the environment.` or `APP_URL is "shop.example.com"; set it to an address like https://shop.example.com.`

`weft create` writes `.env`, which git ignores, and `.env.sample`, which is committed. `weft dev` sets `PORT`, `HOST=127.0.0.1` and `APP_URL` itself (see the [CLI](../reference/cli.md#weft-dev)), and restarts the Go server when `.env` changes. `warp.ConfigFrom(map[string]string{...})` makes a configuration from a map, for tests.

## Behind a proxy

A proxy that ends TLS in front of the app, such as a load balancer or Caddy, sends every request on as plain HTTP from its own address. Without `TRUST_PROXY=on` the app does not know a request was HTTPS, so the login's cookie is not `Secure` and not named `__Host-session`; and it sees the proxy's address as every client's, so the login's limits count every client's wrong passwords together. Those are the two things it changes: the [cross-site check](#cross-site-requests) reads neither header and works the same either way. With `TRUST_PROXY=on` it believes `X-Forwarded-Proto` and the last address of `X-Forwarded-For`. It is off by default, since a client can send both: turn it on only behind a proxy that sets them.

Three methods give the answer either way:

- `app.ClientIP(r)` is the client's address, without its port: the last of `X-Forwarded-For` with `TRUST_PROXY` on, the connection's otherwise.
- `app.IsHTTPS(r)` is true for a TLS connection, or with `TRUST_PROXY` on when the last entry of `X-Forwarded-Proto` is `https`.
- `app.ClientKey(r)` is what a limit counts a client by: the address for IPv4, and the /64 it is in for IPv6 (`2001:db8:1:2::/64`), since one client is commonly given a whole /64.

## Cross-site requests

Every request under `/api` with a method other than `GET`, `HEAD` or `OPTIONS` goes through the standard library's `http.CrossOriginProtection` before any route or middleware. One a browser sent from another site, same-site ones on another origin included, is answered 403 `forbidden`, `This request came from another site.`, and runs nothing. A request with neither `Sec-Fetch-Site` nor `Origin`, such as one from `curl` or a server, passes. An app whose pages are served from another origin lists it in `TRUSTED_ORIGINS`, as `https://admin.example.com`; an entry that is not an origin stops the app with `TRUSTED_ORIGINS has "admin.example.com"; write each origin like https://admin.example.com.` Every origin listed there is trusted for every API request, signing in included.

## Plain handlers and middleware

`app.Handle` mounts any `http.Handler`, for what a typed route does not fit. Its pattern is a method, a space and a path without `/api`:

```go
app.Handle("GET /export.csv", http.HandlerFunc(exportCSV))
```

There are two places for middleware, a `func(http.Handler) http.Handler`:

- `app.Wrap(mw)` puts it around every API request;
- `warp.With(mw)`, given to one route, puts it around that route alone, inside every `Wrap`.

The first given is the outermost. A module may call both from its `Register`:

```go
func needsKey(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Key") == "" {
			warp.WriteError(w, r, warp.Unauthorized("Send your key in X-Key."))
			return
		}
		next.ServeHTTP(w, r)
	})
}
```

```go
app.Wrap(logRequests)
warp.Get(app, "/whoami", whoami, warp.With(needsKey))
```

`app.Routes()` lists every route registered, with its module, name, method, path with `/api`, input and output types, status and permission.

## Testing a module

`app.Handler()` is the API alone, at `/api/...`, so a test can call it with `net/http/httptest` and no server. The template's `app/hello/hello_test.go`:

```go
func TestHello(t *testing.T) {
	app := warp.New(warp.Config{})
	app.Use(Module())
	if err := app.Err(); err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	app.Handler().ServeHTTP(rec, httptest.NewRequest("GET", "/api/hello", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	if got := strings.TrimSpace(rec.Body.String()); got != `{"message":"Hello from Go"}` {
		t.Fatalf("body %q", got)
	}
}
```

`warp.Config{}` works as it is, with the defaults above. Check `app.Err()` first, so a registration mistake fails the test with its sentence. A module with `Migrate` or `Start` needs them run: `app.Start(ctx)` does that without listening, and `app.Stop(ctx)` stops what it started. Run the tests with `go test -tags dev ./...`; the tag is needed until the first `weft build` makes `web/dist`.

The package `github.com/flakerimi/weft/warp/warptest` does the rest. `warptest.New(t, vars, modules...)` builds the app from a map of settings, starts it, stops it when the test ends, and fails the test on a mistake; its client keeps cookies, sends JSON, and decodes the answer or the error shape. The shop's test, cut short:

```go
c := warptest.New(t, map[string]string{
	"DB_DRIVER":        "sqlite",
	"DB_URL":           "file:" + filepath.Join(t.TempDir(), "data", "shop.db"),
	"AUTH_SECRET":      "a secret for tests that is long enough to sign",
	"AUTH_OWNER_EMAIL": "owner@example.com",
}, app.Modules()...)
authtest.Owner(t, c) // sets up the owner and signs c in

var got products.Product
c.Post("/api/products", map[string]any{"name": "Lamp", "price": 30, "status": "live"}).Want(http.StatusCreated).JSON(&got)
if e := c.Fork().Get("/api/products").Want(http.StatusUnauthorized).Error(); e.Code != "unauthorized" {
	t.Errorf("without a session: %+v", e)
}
```

`Get`, `Post`, `Patch` and `Delete` send a path with `/api` in front, as the app sees it, from the app's own origin; `Do` sends a request as it is, so a test can send a cross-site one. `Want(status)` fails the test with the body when the status differs; `JSON(&v)` decodes a 2xx answer, and `Error()` returns the error shape as a `*warp.Error`. `Fork()` is a client with no cookies.

### Everything warptest exports

```go
package warptest // github.com/flakerimi/weft/warp/warptest

type Client struct {
	App *warp.App
}
type Response struct {
	Status int
	Header http.Header
	Body   []byte
}

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
