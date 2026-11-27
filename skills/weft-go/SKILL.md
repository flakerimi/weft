---
name: weft-go
description: Use when writing or shipping the Go side of a Weft app - warp.App in main.go and its modules in app/modules.go, typed routes with warp.Get and warp.Post, tags and validation, the error shape, .env configuration, records with warp/data, login with warp/auth, the client weft gen writes, testing with warptest, serving the page build with warp.Serve and a folder with warp.WithFiles, and building the binary with weft build for another system or a scratch image.
---

# The Go side of a Weft app, and shipping it

Warp (`github.com/flakerimi/weft/warp`, standard library only) is an app's Go half: a `warp.App` of modules with typed Go functions as routes, and `warp.Serve`, serving the page build from memory beside the API in one binary. Pages are the `weft-site` skill. Install `weft` per https://weft.base.al/docs/getting-started/install.

## The app

`weft create` writes `main.go`:

```go
func main() {
	server := warp.New(warp.FromEnv())
	server.Use(app.Modules()...)
	// nil once asked to stop: clean exit.
	if err := server.Serve(dist); err != nil {
		log.Fatal(err)
	}
}
```

- `dist` embeds `web/dist`, and is nil under `-tags dev`, as `weft dev` builds it: `go test -tags dev ./...` until the first `weft build`.
- `app/modules.go` lists the modules in the order they start; a new one is `app/<name>/` with a `Module()`, added there.
- A module is `Name() string` and `Register(app *warp.App) error`, and may add `Migrate`, `Start` and `Stop(ctx) error`: in order before serving, stop in reverse after.
- `go.mod` requires Warp (`github.com/flakerimi/weft`) at the release's tag, from GitHub; `weft upgrade` moves it. Never edit `.weft/`.

## Typed routes

```go
type CreateIn struct {
	Name  string  `json:"name" warp:"required,max=120"`
	Price float64 `json:"price" warp:"min=0"`
	Plan  string  `json:"plan" warp:"enum=basic|pro"`
}

func (module) Register(app *warp.App) error {
	warp.Get(app, "/products/{id}", show) // func(ctx, ShowIn) (Product, error)
	warp.Post(app, "/products", create)   // 201
	return nil
}
```

- 200, `Post` 201, `warp.Status(200)` to change it; `warp.None` reads nothing, or answers 204.
- Each exported input field takes exactly one tag: `json` (the body: POST, PUT, PATCH only), `body:"json"` (the whole body, on one struct field), `query` (text, bool, numbers, a pointer to one, `[]string`, `map[string]string` for `filter[k]`) or `path` (a `{name}` wildcard).
- A body is `application/json` (else 415), with no field the input lacks (400).
- Rules: `required`, `email`, `url`, `min=N`, `max=N`, `enum=a|b`, `unique` (a `data` store checks it), `readonly` (never read). All rules but `required` pass an empty value; a non-nil pointer was sent. On `Patch` rules run only on sent fields (`warp.Sent`). A failure is 422. A `Validate(ctx) error` method checks across fields.
- `GET /products` and `GET /products/{id}` are both named `products`: give each a `warp.Name`. A mistake is one sentence from `app.Err()`, and `Serve` will not start.
- `warp.Request(ctx)` is the request, `warp.ResponseHeader(ctx)` the answer's headers. `app.Handle("GET /export.csv", h)` mounts a plain handler; `app.Wrap(mw)` wraps every API request, `warp.With(mw)` one route.

## Errors and configuration

Errors are `{"error":{"code":"…","message":"…","fields":{…}}}`. Return `warp.BadRequest`, `Unauthorized`, `Forbidden`, `NotFound`, `Conflict`, `TooLarge`, `UnsupportedMedia`, `Invalid(fields)` (422), `TooMany` (429) or `Unavailable` (503); any other error is logged and answered 500 `internal`. `warp.WriteError` writes one from a plain handler.

`warp.FromEnv()` reads `.env`, then the environment, which wins: `PORT` (8080), `HOST`, `MAX_BODY_MB` (1), `APP_URL`, `TRUST_PROXY` (`on` behind a proxy that ends TLS, else no HTTPS and one address for all), `TRUSTED_ORIGINS`. A non-GET `/api` request from another site is 403. Read your own keys with `app.Config().Get` and `On`.

## Data, login and types

- `data.Module()` first in `app/modules.go`, `DB_DRIVER=sqlite` and `DB_URL=file:data/app.db` in `.env`, `_ "modernc.org/sqlite"` in `main.go`, then `go mod tidy` to add the driver to `go.mod`. A model embeds `data.Record`; `data.Resource(app, "/products", data.NewStore[Product](data.For(app), "products"), data.PerAction(auth.Per("products")))` gives five routes. `data:"sort,filter,search"` opens fields to `?sort=-price&q=lamp&filter[status]=live`.
- `auth.Module()` after data: `AUTH_SECRET` (32+ bytes), `AUTH_OWNER_EMAIL`; the log prints `<APP_URL>/setup#token=…`, a 404 until the app writes `web/pages/setup.vue` and a `/login`: copy both from [the login guide](https://weft.base.al/docs/guides/login#the-pages). `auth.Can("x:edit")` guards a route; `auth.UserOf(ctx)`.
- `weft dev` and `weft build` run `weft gen`, which writes `web/.weft-types/api.ts`: `api.<module>.<name>()`, `ApiError`, and `schema` for `Form`.

## Testing a module

Test `app.Handler()` with `httptest` after checking `app.Err()`; `app.Start(ctx)` a module that migrates or starts. `warptest.New(t, vars, modules...)` starts the app with a client that keeps cookies; `authtest.Owner(t, c)` signs it in.

## What Serve answers

`/api` goes to the API. Any other path gets its page, else a listed file, else the app's `index.html` under `--app-prefix`, else a run-time page whose pattern matches, else `pages/404.vue` with 404. Serve sends the stored brotli or gzip and the build's `ETag`, caches hashed files for a year, sends `103 Early Hints` to a navigation over HTTP/2 or through a proxy, and answers only `GET` and `HEAD`. It panics at start if the build is incomplete.

## Files from disk

For downloads and media:

```go
files, err := os.OpenRoot("/data/files")
if err != nil {
	log.Fatal(err)
}
if err := server.Serve(dist, warp.WithFiles("/files/", files.FS())); err != nil {
	log.Fatal(err)
}
```

Each file is read once at start for its ETag; one changed since gets 500, a new one is not served. A symlink or dotfile panics. `weft dev` does not serve the files.

## One binary

```sh
weft build  # web/dist, then ./<name>
GOOS=linux GOARCH=arm64 weft build -o myapp-linux-arm64
```

The binary is built with `-trimpath -ldflags "-s -w"` and `CGO_ENABLED=0` unless set. It needs nothing beside it and runs on `FROM scratch`:

```dockerfile
FROM scratch
COPY myapp /myapp
USER 65534:65534
EXPOSE 8080
ENTRYPOINT ["/myapp"]
```

Build it for the image's system, using a `.dockerignore` of `*` and `!myapp`.

## Do not

- Add a Go dependency you do not need. Go: use the standard library and `github.com/flakerimi/weft/warp`; add a dependency only for what the standard library cannot do, such as a database driver.
- Register a module from `init`, write `/api` in a route's path, or answer an error with JSON of your own.
- Compress, render or hash anything per request, or embed large media.
- Run `scripts/publish-site.sh`, `scripts/rollback-site.sh` or `lisa` for weft.base.al without `--dry-run`: that is the owner's step.

## Not yet

Planned, not built (see https://weft.base.al/docs/coming-next): PostgreSQL, MySQL and relations; roles, registration, reset and `mail`; `weft scaffold`; `storage`, `events` and `live`; `jobs`, `i18n`, `middleware`, `docs`; `weft deploy`.
