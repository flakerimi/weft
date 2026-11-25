---
title: Warp, the Go side
description: An app's modules, its typed routes and their validation, the one error shape, and configuration from .env.
order: 22
---

On this page you learn how the Go half of an app works: the app and its modules, routes that are typed Go functions, the rules that check their input, the one shape every error has, and the configuration in `.env`. Warp is the package `github.com/flakerimi/weft/warp`, and uses only Go's standard library. What it does with the built pages is in [Serving from Go](go-serving.md).

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

The modules are listed in one place, `app/modules.go`. Nothing registers itself from an `init` function, so this list is the whole app:

```go
// Package app lists the modules of the app.
package app

import (
	"github.com/flakerimi/weft/warp"

	"myapp/app/hello"
)

// Modules is the one list of the app's modules, in the order they start.
func Modules() []warp.Module {
	return []warp.Module{
		hello.Module(),
	}
}
```

To add a module, make a folder `app/<name>/` with a `Module()` function and add it to this list.

## A module

A module has a name and registers its routes on the app it is given:

```go
type Module interface {
	Name() string
	Register(app *App) error
}
```

The template's `app/hello/routes.go` is one, with the route `GET /api/hello`.

Two modules may not have the same name. An error from `Register` is reported as `warp: module "hello" could not register: <the error>`.

A module may also have any of three more methods, each a step of the app's life:

| Interface | Method | When it runs |
| --- | --- | --- |
| `warp.Migrator` | `Migrate(ctx context.Context) error` | First, before anything starts, in the order the modules are listed |
| `warp.Starter` | `Start(ctx context.Context) error` | Next, before the app serves, in the same order |
| `warp.Stopper` | `Stop(ctx context.Context) error` | Last, after the app has stopped serving, in reverse order |

When the app is told to stop, it takes no new requests and lets the ones in flight finish, for up to 10 seconds. Then it stops the modules, last first, with 10 seconds more for them. When a step fails, the app stops the modules that already started, and `Serve` returns the error, such as `warp: module "jobs" could not start: <the error>`.

A module with work of its own while the app runs:

```go
type module struct{ stop context.CancelFunc }

func (*module) Name() string                 { return "jobs" }
func (*module) Register(app *warp.App) error { return nil }

// Start runs before the app serves.
func (m *module) Start(ctx context.Context) error {
	work, stop := context.WithCancel(context.Background())
	m.stop = stop
	go run(work)
	return nil
}

// Stop runs after the app has stopped serving.
func (m *module) Stop(ctx context.Context) error {
	m.stop()
	return nil
}
```

`run` is the module's own work, which returns when `work` is cancelled.

## Typed routes

A route is a Go function that takes a context and an input struct and returns an output and an error. Register it with `warp.Get`, `warp.Post`, `warp.Put`, `warp.Patch` or `warp.Delete`, and write its path without `/api`, which the app adds:

```go
// Product is what the routes answer with.
type Product struct {
	ID    int     `json:"id"`
	Name  string  `json:"name"`
	Price float64 `json:"price"`
}

// ShowIn is the input of GET /api/products/{id}.
type ShowIn struct {
	ID int `path:"id"`
}

// ListIn is the input of GET /api/products.
type ListIn struct {
	Page    int      `query:"page"`
	PerPage *int     `query:"per_page" warp:"min=1,max=100"`
	Tags    []string `query:"tag"`
}

// CreateIn is the body of POST /api/products.
type CreateIn struct {
	Name  string  `json:"name" warp:"required,max=120"`
	Price float64 `json:"price" warp:"min=0"`
	Plan  string  `json:"plan" warp:"enum=basic|pro"`
}

func (module) Register(app *warp.App) error {
	warp.Get(app, "/products", list)
	warp.Get(app, "/products/{id}", show)
	warp.Post(app, "/products", create)
	warp.Delete(app, "/products/{id}", remove)
	return nil
}

func list(ctx context.Context, in ListIn) ([]Product, error) {
	return nil, nil // a nil slice is answered as []
}

func show(ctx context.Context, in ShowIn) (Product, error) {
	if in.ID != 1 {
		return Product{}, warp.NotFound(fmt.Sprintf("No product has id %d.", in.ID))
	}
	return Product{ID: 1, Name: "Lamp", Price: 12}, nil
}

func create(ctx context.Context, in CreateIn) (Product, error) {
	return Product{ID: 2, Name: in.Name, Price: in.Price}, nil
}

func remove(ctx context.Context, in ShowIn) (warp.None, error) {
	return warp.None{}, nil
}
```

`GET /api/products/1` answers `{"id":1,"name":"Lamp","price":12}`. The paths are the standard library's `http.ServeMux` patterns, wildcards included.

The output is written as JSON, with status 200. `warp.Post` answers 201 instead, and the option `warp.Status(200)` changes it, as in `warp.Post(app, "/search", search, warp.Status(200))`. An output of `warp.None` answers 204 with no body. An input of `warp.None` reads nothing from the request.

### The three tags

Every exported field of an input has exactly one of three tags, which says where it comes from:

| Tag | Filled from | Types |
| --- | --- | --- |
| `json:"name"` | The JSON body. Only `POST`, `PUT` and `PATCH` read one. | Anything `encoding/json` reads |
| `query:"name"` | The query string | `string`, `bool`, any int, uint or float, a pointer to one of them, or `[]string` |
| `path:"name"` | The wildcard `{name}` of the path | `string`, `bool`, any int, uint or float |

The body is read first, then the query, then the path, so a body can never set a query or path field. A struct inside a body field is read from the body too: its fields need `json` tags, and take no `query` or `path` tag.

In the query, a pointer stays nil when the key is absent, and a `[]string` gets every value of a repeated key (`?tag=a&tag=b`). A key that is present but empty, such as `?page=`, counts as absent for a number or a bool.

### JSON bodies only

A body that is not empty must be sent with `Content-Type: application/json`, or a type that ends in `+json`; anything else is answered 415. A page on another site can send a form or `text/plain` with the visitor's cookies and no preflight, and taking JSON alone keeps it from writing to a typed route. An empty body reads as `{}`.

### When the input is wrong

The handler is not called, and the answer is in the [error shape](#errors):

| The request | Status, code | Message, fields |
| --- | --- | --- |
| A body larger than `MAX_BODY_MB` | 413 `too_large` | `The body is larger than 1 MB.` |
| A body that is not sent as JSON | 415 `unsupported_media_type` | `Send the body as application/json.` |
| A body that is not JSON, or has more after the value | 400 `bad_request` | `The body is not valid JSON.` |
| A body that is JSON but not an object | 400 `bad_request` | `The body must be a JSON object.` |
| A body field the input does not have | 400 `bad_request` | `The body has the field "nme", which this route does not take.` |
| A body field of the wrong type | 400 `bad_request` | `1 field has the wrong type.`, with `price: Send a number.` (or `Send text.`, `Send true or false.`, `Send a list.`, `Send an object.`) |
| A query value that does not convert | 400 `bad_request` | `1 field has the wrong type.`, with `page: Send a whole number.` (or `Send a number.`, `Send true or false.`) |
| A path value that does not convert | 404 `not_found` | `Nothing is at /api/products/abc.` |
| A rule that fails | 422 `validation` | `1 field needs attention.`, with a message per field (see [Validation](#validation)) |

A field the body does not have is refused because a misspelt field that is dropped without a word is how an update loses data.

### The request and the answer's headers

A handler gets the request it is answering, and headers to send with its answer, from its context:

```go
// whoami reads the request and sets a header on the answer.
func whoami(ctx context.Context, _ warp.None) (Product, error) {
	r := warp.Request(ctx)
	warp.ResponseHeader(ctx).Set("Cache-Control", "no-store")
	return Product{Name: r.Header.Get("User-Agent")}, nil
}
```

On `POST`, `PUT` and `PATCH` the request's body has already been read into the input, so there is nothing left in it to read. The headers are sent whether the handler succeeds or fails, so a `Set-Cookie` goes out with an error too.

### Mistakes in registering

A mistake in a route is found when it is registered, before any request: `warp.Get` and the rest return nothing, and `app.Err()` reports every mistake so far, one sentence each. `Serve` returns them all without listening, so the app does not start. For example:

```text
warp: GET /products/{id}: BadIn.Name has no json, query or path tag; add one.
warp: GET /products/{id}: BadIn.Email is tagged json, and GET reads no body; tag it query or path.
warp: GET /products/{id}: BadIn.Email has the rule "emial", which warp does not know; use required, email, url, min=, max=, enum= or unique.
```

Besides those, a registration is refused when a field has two of the three tags, a `json` tag has no name or a name that is not letters, digits, `_`, `-` and `.`, two fields have the same key, a `path` tag names no wildcard of the path or a wildcard has no field, a query or path field has a type it cannot be filled into, a struct inside a body field has a field without a `json` tag or with a `query` or `path` tag, the input embeds a pointer to a struct, a rule does not fit its field, `warp.Status` is given a code outside 100 to 599, a path starts with `/api/`, or the same route is registered twice.

## Validation

Rules go in a field's `warp` tag, separated by commas. They run on every typed input after it is filled and before the handler:

| Rule | For | Fails when | Message |
| --- | --- | --- | --- |
| `required` | anything | the value is zero, an empty slice or map, a nil pointer, or text that is empty once trimmed | `This is required.` |
| `email` | text | it is not an email address, or it has a display name | `Enter an email address.` |
| `url` | text | it is not an absolute `http` or `https` address with a host | `Enter a web address that starts with http:// or https://.` |
| `min=N` | text, a number, a list | it has fewer than N characters, is below N, or has fewer than N items | `Use at least N characters.`, `Enter N or more.`, `Choose at least N.` |
| `max=N` | the same | more than N | `Use at most N characters.`, `Enter N or less.`, `Choose at most N.` |
| `enum=a\|b` | text | it is not one of them | `Choose one of: a, b.` |
| `unique` | text, a number | never: the rule is accepted, and nothing checks it yet (see [Coming next](../coming-next.md)) | |

- Every rule but `required` passes an empty value, so an optional field may be left out: an empty `email` passes, and a 0 passes `min=1`. Write `required,min=1` to refuse 0.
- `required` on a pointer passes whenever the pointer is not nil, even when it points at a zero: a pointer is how an input says a value was sent. The other rules apply to what it points at.
- Characters are counted as letters, not bytes: `çikë` is 4.
- Only the first rule a field fails is reported, in the order of the tag.
- A field's key is its `json`, `query` or `path` name. A field of a struct inside is `address.city`, and of a list of structs `lines.0.sku`.

For the body `{"name":"","price":-1,"plan":"gold"}`, `POST /api/products` answers 422 with:

```json
{"error":{"code":"validation","message":"3 fields need attention.","fields":{"name":"This is required.","plan":"Choose one of: basic, pro.","price":"Enter 0 or more."}}}
```

`warp.Validate(v)` checks any struct against its tags the same way, and returns nil or the 422 error.

## Errors

Every error the API answers has one shape, with `Content-Type: application/json`:

```json
{"error":{"code":"not_found","message":"No product has id 7."}}
```

`fields` is there only when it has something, as in the 422 above. A handler chooses its answer by returning one of these:

| Function | Status | Code |
| --- | --- | --- |
| `warp.BadRequest(message)` | 400 | `bad_request` |
| `warp.Unauthorized(message)` | 401 | `unauthorized` |
| `warp.Forbidden(message)` | 403 | `forbidden` |
| `warp.NotFound(message)` | 404 | `not_found` |
| `warp.Conflict(message)` | 409 | `conflict` |
| `warp.TooLarge(message)` | 413 | `too_large` |
| `warp.UnsupportedMedia(message)` | 415 | `unsupported_media_type` |
| `warp.Invalid(fields)` | 422 | `validation`, with the message `1 field needs attention.` or `N fields need attention.` |
| `warp.TooMany(message)` | 429 | `rate_limited` |

Each returns a `*warp.Error`, which is found even when it is wrapped, as by `fmt.Errorf("load: %w", err)`. Any other error, and a panic, is logged with the method and the path, and answered 500, code `internal`, with `Something went wrong on the server; its log has the details.` and none of the error's text. The next request is served as usual.

The app answers some errors itself:

- a path under `/api` that no route has: 404 `not_found`, `Nothing is at /api/nope.`;
- a method a route does not take: 405 `method_not_allowed`, `/api/products/1 does not take PUT; use DELETE or GET.`, with an `Allow` header.

`warp.WriteError(w, r, err)` writes an error in this shape from a plain handler, and `warp.WriteJSON(w, status, v)` writes a value as JSON, a nil slice as `[]`.

## Configuration

`warp.FromEnv()` reads `.env` in the folder the app runs in, if there is one, then the environment, which wins over the file. Warp reads three keys:

| Key | What it is | Default |
| --- | --- | --- |
| `PORT` | The port to listen on, 1 to 65535 | `8080` |
| `HOST` | The address to listen on | empty: every interface |
| `MAX_BODY_MB` | The largest body a typed route reads, in whole megabytes, 1 or more | `1` |

A module reads its own keys with `app.Config().Get("KEY")`, which is `""` when it is unset, and `app.Config().On("KEY")`, which is true for `on`, `true`, `1` and `yes`, in any case.

`.env` has one `KEY=value` per line. Blank lines and lines that start with `#` are skipped, and a leading `export ` is dropped. A value in single or double quotes is what is between them; an unquoted value is trimmed and ends at ` #`. There is no `${}` expansion. A line it cannot read, or a `PORT` or `MAX_BODY_MB` that is not a number it takes, stops the app with one sentence, such as `PORT is "abc"; set it to a number from 1 to 65535 in .env or the environment.`

`weft create` writes `.env`, which git ignores, and `.env.sample`, which is committed. `weft dev` sets `PORT` and `HOST=127.0.0.1` itself, and restarts the Go server when `.env` changes. `warp.ConfigFrom(map[string]string{...})` makes a configuration from a map, for tests.

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

`app.Routes()` lists every route registered, with its method, its path with `/api`, its input and output types, and its status.

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

`warp.Config{}` works as it is: port 8080 and a 1 MB body limit. Check `app.Err()` first, so a registration mistake fails the test with its sentence. Run the tests with `go test -tags dev ./...`; the tag is needed until the first `weft build` makes `web/dist`.
