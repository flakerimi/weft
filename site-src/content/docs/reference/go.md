---
title: Go package
description: The weft package's functions, options and variables.
order: 31
---

This page lists everything the `weft` Go package exports today. Import it as `github.com/flakerimi/weft`. It uses only the standard library. The [Serving from Go](../guides/go-serving.md) guide explains the behaviour in more detail.

## Serve

```go
func Serve(dist fs.FS, api http.Handler, opts ...Option) http.Handler
```

Returns the handler for a Weft application.

- Paths `/api` and under `/api/` go to `api`. With `api` nil they get a JSON 404: `{"error":{"code":"not_found","message":"no API"}}`.
- When `dist` holds a page build (`routes.json` and the `manifest.json` that `weft-build-pages` writes), every file is read into memory once, here. Each request gets a page, a listed file, the app's `index.html` under its prefix, or the not-found page with status 404. Each answer is the stored brotli, gzip or plain variant the client accepts best, with an `ETag`; `If-None-Match` gets a 304. Hashed files are cached for a year, the rest revalidated. A page with linked files gets a `103 Early Hints` first.
- Otherwise `dist` is a single-page app: its files are served as they are, and every other path gets its `index.html`. Files under `/assets/` are cached for a year; a missing one is a 404.

Serve panics, in one sentence that names the file, when the build is incomplete or `dist` has no `index.html`. It does that at start, not on a request.

## Option

```go
type Option func(*options)
```

Changes what Serve serves. `WithFiles` is the one option today.

## WithFiles

```go
func WithFiles(prefix string, dir fs.FS) Option
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

The Weft release this build belongs to. Release builds set it with `-ldflags "-X github.com/flakerimi/weft.Version=0.0.3"`.
