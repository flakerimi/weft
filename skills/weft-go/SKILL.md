---
name: weft-go
description: Use when writing the Go side of a Weft app or shipping it - the warp.App in main.go and its modules in app/modules.go, typed routes with warp.Get and warp.Post, json, query and path tags and warp validation rules, the one error shape, .env configuration, testing a module with app.Handler, serving the page build with warp.Serve and a folder from disk with warp.WithFiles, building the binary with weft build for another system or a scratch container image, checking compression and ETags with curl, and installing or updating the weft command.
---

# The Go side of a Weft app, and shipping it

An app's Go half is Warp, `github.com/flakerimi/weft/warp`, on Go's standard library alone: a `warp.App` with a list of modules, whose routes are typed Go functions with a validated input, answering errors in one JSON shape; `warp.Serve` serves the page build from memory beside the API, so the app is one binary. Pages are the `weft-site` skill.

## The app

`weft create` writes `main.go`:

```go
func main() {
	server := warp.New(warp.FromEnv())
	server.Use(app.Modules()...)
	// Serve returns nil once it was asked to stop: that is a clean exit.
	if err := server.Serve(dist); err != nil {
		log.Fatal(err)
	}
}
```

- `dist` is set by `dist.go`, which embeds `web/dist`; under `-tags dev`, as `weft dev` builds it, it stays nil. Run Go commands with `-tags dev` until the first `weft build`: `go test -tags dev ./...`.
- `app/modules.go` is the one list of modules, in the order they start; nothing registers itself from `init`. A new module is `app/<name>/` with a `Module()`, added to the list.
- A module is `Name() string` and `Register(app *warp.App) error`, and may add `Migrate`, `Start` and `Stop` (`func(ctx context.Context) error`): migrate and start in order before serving, stop in reverse after. On SIGINT or SIGTERM requests in flight get up to 10 s.
- Never edit `.weft/`, where `go.mod` finds Warp.

## Typed routes

```go
type CreateIn struct {
	Name  string  `json:"name" warp:"required,max=120"`
	Price float64 `json:"price" warp:"min=0"`
	Plan  string  `json:"plan" warp:"enum=basic|pro"`
}

func (module) Register(app *warp.App) error {
	warp.Get(app, "/products/{id}", show) // func(ctx context.Context, in ShowIn) (Product, error)
	warp.Post(app, "/products", create)   // answers 201
	return nil
}
```

- Paths have no `/api`; the app adds it. `Get`, `Post`, `Put`, `Patch`, `Delete`. 200, `Post` 201, `warp.Status(200)` to change it; `warp.None` as input reads nothing, as output answers 204.
- Each exported input field has exactly one tag: `json` (the body: POST, PUT, PATCH only), `query` (text, bool, numbers, a pointer to one, `[]string`) or `path` (a `{name}` wildcard).
- A body must be `application/json` (else 415), one JSON object, with no field the input lacks (400). A path value that does not convert is 404.
- Rules: `required`, `email`, `url`, `min=N`, `max=N`, `enum=a|b`, `unique` (not checked yet). Every rule but `required` passes an empty value; `required` on a pointer passes when it is not nil. A failure is 422 with a message per field.
- A mistake in a route (a field with no tag, a `path` tag with no wildcard, an unknown rule) is one sentence from `app.Err()`, and `Serve` refuses to start.
- `warp.Request(ctx)` is the request, `warp.ResponseHeader(ctx)` the answer's headers. `app.Handle("GET /export.csv", h)` mounts a plain handler; `app.Wrap(mw)` wraps every API request, `warp.With(mw)` one route.

## Errors and configuration

Errors are `{"error":{"code":"…","message":"…","fields":{…}}}`. Return `warp.BadRequest` (400), `Unauthorized` (401), `Forbidden` (403), `NotFound` (404), `Conflict` (409), `TooLarge` (413), `UnsupportedMedia` (415), `Invalid(fields)` (422) or `TooMany` (429), with a one-sentence message; any other error or a panic is logged and answered 500 `internal`, without its text. `warp.WriteError` writes one from a plain handler.

`warp.FromEnv()` reads `.env`, then the environment, which wins: `PORT` (8080), `HOST` (every interface), `MAX_BODY_MB` (1). Read your own keys with `app.Config().Get` and `On`. `.env` is git-ignored, `.env.sample` committed; `weft dev` restarts Go when `.env` changes.

## Testing a module

Build `warp.New(warp.Config{})`, `Use(Module())`, check `app.Err()`, then call `app.Handler()`, the API alone, with `httptest`: `app.Handler().ServeHTTP(rec, httptest.NewRequest("GET", "/api/hello", nil))`. Set `Content-Type: application/json` on a request with a body.

## What Serve answers

`/api` and everything under it go to the API. Every other path of a page build gets the page at that path, else a file the manifest lists, else the app's `index.html` under `--app-prefix`, else `pages/404.vue` with 404; a single-page app gets its file, or `index.html`. Serve sends the stored brotli or gzip the request accepts best, compressing nothing per request; gives each file the build's `ETag` (`If-None-Match` gets 304); caches hashed files for a year; sends `103 Early Hints`; and answers only `GET` and `HEAD`. It panics at start, naming the file, if the build is incomplete: fix the build.

Check it after `weft build` and `./myapp`: `curl -sI -H 'Accept-Encoding: br' http://localhost:8080/` shows `Content-Encoding: br` and an `ETag`.

## Files from disk

For downloads and media that do not belong in the binary, open the folder in `main.go` and give it to `Serve`:

```go
files, err := os.OpenRoot("/data/files")
if err != nil {
	log.Fatal(err)
}
if err := server.Serve(dist, warp.WithFiles("/files/", files.FS())); err != nil {
	log.Fatal(err)
}
```

The prefix starts and ends with `/`. Each file is read once at start for its ETag; one changed since gets 500, a new one is not served. Serve panics if the folder holds a symlink, a dotfile, or a path the build or `/api/` has.

## One binary

```sh
weft build                                     # the pages into web/dist, then ./<name>
GOOS=linux GOARCH=arm64 weft build -o myapp    # for a Linux server on arm64
```

`weft build` builds the pages first, since the binary embeds `web/dist`, then the binary with `-trimpath -ldflags "-s -w"`, and `CGO_ENABLED=0` unless the environment sets it. It needs no Node and no files beside it, and runs on `FROM scratch`:

```dockerfile
FROM scratch
COPY myapp /myapp
USER 65534:65534
EXPOSE 8080
ENTRYPOINT ["/myapp"]
```

Build the image in the app's folder, from a binary built for its system, with a `.dockerignore` of `*` and `!myapp`.

## The weft command

`curl -fsSL https://weft.base.al/install | sh` installs it (`WEFT_VERSION`, `WEFT_BIN_DIR`, `WEFT_BASE`). `weft update` moves to the latest release; `--check` only looks, `--version 0.0.2` picks one.

## Do not

- Add a Go dependency the app does not need. Go: use the standard library and `github.com/flakerimi/weft/warp`; add a dependency only for what the standard library cannot do, such as a database driver.
- Register a module from `init`, write `/api` in a route's path, or answer an error with JSON of your own.
- Compress, render or hash anything per request, or embed large media.
- Run `scripts/publish-site.sh`, `scripts/rollback-site.sh` or `lisa` for weft.base.al without `--dry-run`. Publishing that site is the owner's step.

## Not yet

Planned, not built (see https://weft.base.al/docs/coming-next): `weft gen` and types from Go; `warp/data`, list queries and `unique` checked; `auth` and `mail`; `weft scaffold`; `storage`, `events` and `live`; `jobs`, `i18n`, `middleware`, `docs`; `weft deploy`. Today write TypeScript types by hand, and store data with `database/sql`.
