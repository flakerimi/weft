---
name: weft-go
description: Use when serving a Weft build from Go or shipping it - writing the main.go that embeds dist with weft.Serve, adding an /api handler, serving a folder from disk with weft.WithFiles, building one static binary or a scratch container image, checking compression, ETags and 304s with curl, and installing or updating the weft command.
---

# Serving and publishing a Weft site from Go

`weft.Serve` turns a Weft build into an `http.Handler`. It reads every page and file into memory when it starts, already compressed by the build, and answers each request from a map. Your API sits beside it, so the whole site is one binary. The package is `github.com/flakerimi/weft` and uses only Go's standard library. Build the pages first; the `weft-site` skill covers that.

## The program

```go
// Command hello serves the pages in dist and a small API, from one binary.
package main

import (
	"embed"
	"encoding/json"
	"io/fs"
	"log"
	"net/http"

	"github.com/flakerimi/weft"
)

//go:embed all:dist
var site embed.FS

func main() {
	dist, err := fs.Sub(site, "dist")
	if err != nil {
		log.Fatal(err)
	}
	api := http.NewServeMux()
	api.HandleFunc("GET /api/hello", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"message": "Hello from Go"})
	})
	log.Fatal(http.ListenAndServe(":8080", weft.Serve(dist, api)))
}
```

- Write `//go:embed all:dist`, never a plain `dist`. Without `all:`, files whose names start with `.` or `_` are left out.
- `/api` and everything under `/api/` go to `api`. With `nil`, they get a JSON 404: `{"error":{"code":"not_found","message":"no API"}}`.
- A site inside the repository is part of the `github.com/flakerimi/weft` module. Run it with `go run ./hello`.

## What Serve answers

The answer depends on what `dist` holds:

- **A page build** (`routes.json` and `manifest.json`). A request gets, in order:
  1. the page at that path, with or without a trailing slash;
  2. a file the manifest lists;
  3. the app's `index.html` under `--app-prefix`;
  4. the `pages/404.vue` page, with status 404.
- **A single-page app** (a plain Vite build with `index.html`). A request gets the file at its path, or else `index.html`.

The build's own files are never served: `manifest.json`, `routes.json`, `.weft-build` and `/pages/…`.

For a page build, Serve also:

- sends the stored brotli or gzip variant the request accepts best, with `Vary: Accept-Encoding`. It compresses nothing per request;
- gives every page and file the build's strong `ETag`, and a matching `If-None-Match` gets `304`;
- caches files named by a hash for a year (`immutable`) and revalidates everything else (`no-cache`);
- sends `103 Early Hints` first for a page with fonts, a linked stylesheet or island scripts;
- answers only `GET` and `HEAD`. Anything else gets `405` with `Allow: GET, HEAD`.

Serve panics at start, in one sentence naming the file, if the build is incomplete. Fix the build, not the server.

Check it:

```sh
curl http://localhost:8080/api/hello
curl -sI -H 'Accept-Encoding: br' http://localhost:8080/
```

The page should come back with `Content-Encoding: br` and an `ETag`.

## Files from disk

Some files should not go into the binary: downloads, media, anything that changes between releases. Serve a folder for them:

```go
files, err := os.OpenRoot("/data/files")
if err != nil {
	log.Fatal(err)
}
handler := weft.Serve(dist, nil, weft.WithFiles("/files/", files.FS()))
```

- The prefix starts and ends with `/`. With `/`, files are served at their own paths.
- Open the folder with `os.OpenRoot`, so a symlink added later cannot lead out of it.
- Each file is read once at start for its ETag and type, and its bytes are read from disk on each request.
- A file changed since the start gets `500`, and a new file is not served. Restart after you change the folder.
- Serve panics if the folder holds a symlink or a dotfile, or if a path is also a page or file of the build, or is under `/api/`.

## One binary

```sh
pnpm --filter hello build
CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o hello-site ./hello
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags "-s -w" -o hello-site ./hello
```

Always build the pages before `go build`, because the binary embeds `dist/` as it is at that moment. The binary needs no Node and no files beside it. It runs on `FROM scratch`:

```dockerfile
FROM golang:1.26 AS build
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o /out/hello-site ./hello

FROM scratch
COPY --from=build /out/hello-site /hello-site
USER 65534:65534
EXPOSE 8080
ENTRYPOINT ["/hello-site"]
```

Build the image from the repository root with `hello/dist` already built. Put `**/node_modules` in `.dockerignore`.

## The weft command

```sh
curl -fsSL https://weft.base.al/install | sh
weft version
weft update --check
weft update --version 0.0.2
```

- The installer reads `WEFT_VERSION`, `WEFT_BIN_DIR` (default `$HOME/.local/bin`) and `WEFT_BASE`.
- `weft update` checks the archive against `SHA256SUMS` and swaps the binary in place.
- A development build (`0.0.0-dev`) refuses to update.

## Do not

- Add a dependency to the Go side. The `weft` package and its users here use only the standard library.
- Compress, render or hash anything per request. Everything comes from the build's manifest.
- Embed releases or large media. Serve them with `WithFiles`.
- Run `scripts/publish-site.sh`, `scripts/rollback-site.sh` or `lisa` for weft.base.al without `--dry-run`. Publishing that site is the owner's step.

## Not yet

These are planned but not built (see https://weft.base.al/docs/coming-next):

- `weft dev`, `weft build` and `weft deploy`;
- `weft gen` and the Go-to-Vue contract: `weft.Validate`, `weft.Resource[T]` and the wire format;
- `weft.Auth` and `weft.Require`;
- `weft.SSE` with `useLive`.

Today an API is plain `net/http`.
