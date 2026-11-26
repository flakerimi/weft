---
title: Serving from Go
description: What warp.Serve sends for each request, and how to serve files from disk beside the build.
order: 27
---

On this page you learn what `warp.Serve` answers for each request, how it caches and compresses, and how to serve a folder from disk next to your pages. An app's `server.Serve(dist)` serves through it, with the app's API beside the pages; [Warp, the Go side](warp.md) and [Typed routes](routes.md) cover the API. For the signatures, see the [Go package reference](../reference/go.md).

## Two kinds of build

`warp.Serve(dist, api)` looks at `dist` once, when it is called, which for an app is when `server.Serve(dist)` starts:

- **A page build**, which `weft-build-pages` writes: `dist` has `routes.json` and `manifest.json`. Serve reads every file into memory and answers each request from there.
- **A single-page app**, which a plain Vite build writes: `dist` has an `index.html` and no route map. Serve sends its files as they are, and `index.html` for every other path, so your Vue router can handle it.

Either way, `/api` and the paths under it go to `api`, which for an app is its `app.Handler()`. With `api` nil they get a 404 in the error shape, `This app has no API.`

## A page build

For each request that is not for the API, Serve sends:

1. the page at that path, with or without a trailing slash;
2. else the file the manifest lists at that exact path;
3. else, under the app's prefix (see below), the app's `index.html`;
4. else the not-found page, `pages/404.vue`, with status 404, or a plain 404.

It never serves the build's own files: `manifest.json`, `routes.json`, `.weft-build`, and the page files under `/pages/`.

### Compression

The build stored each text file three times: as it is, brotli and gzip. Serve picks the one the request's `Accept-Encoding` likes best, brotli when they tie, and sends it with `Vary: Accept-Encoding`. It compresses nothing per request.

### Caching

- Every page and file of the build, the not-found page included, carries the strong `ETag` the build gave it, with `-br` or `-gz` added for a compressed copy; so does each file `WithFiles` serves. A request whose `If-None-Match` holds it gets `304 Not Modified` with no body. Your API's answers are your own, and a single-page app's files carry no ETag.
- Files named by a hash of their bytes (the island scripts, linked CSS, fonts, images in Markdown) get `Cache-Control: public, max-age=31536000, immutable`.
- Everything else, pages included, gets `Cache-Control: no-cache`: the browser keeps it, and asks each time whether it changed.

### Early Hints

When a page has a linked stylesheet, fonts or island scripts, its response carries their `Link` headers. The island scripts hinted are the loader, and the runtime too only on a page with an island that starts on `load`.

Serve can also send those `Link` headers ahead of the page, in a `103 Early Hints`, so the browser starts fetching them while the page is on its way. Chrome acts on a 103 over HTTP/2 or later and ignores one over HTTP/1.1, so sent straight to a browser over HTTP/1.1 it would be a wasted write on every page. Serve sends it when the request is a browser's navigation (`Sec-Fetch-Mode: navigate`) and either came over HTTP/2 or later, or came through a proxy (it has a `Forwarded`, `X-Forwarded-For` or `Via` header): a proxy that speaks HTTP/2 to the browser and passes 1xx responses on forwards the 103 to it. A proxy that adds none of those headers and speaks HTTP/1.1 to the binary gets no 103, so set one at the proxy (with nginx, `proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;`); a request over HTTP/1.0 never gets one. This only decides whether to send the extra response, so `TRUST_PROXY` does not change it, and the page's own response is the same either way.

### Methods

Pages and files answer `GET` and `HEAD`. Other methods get `405` with an `Allow: GET, HEAD` header.

## A client-rendered app beside the pages

A page build can carry a single-page app under a path prefix. Build the app with Vite first, then give its prefix and folder to the page build:

```sh
weft-build-pages --root . --out dist --app-prefix /app --app-dir app-dist
```

Paths under `/app` get the app's `index.html`, except files under `/app/assets/`, which get a 404 when missing. Set `build.manifest: true` in the app's Vite config, so the files it hashed are cached for a year.

## Files from disk

Some files do not belong in the binary: downloads, large media, anything that changes between releases. `warp.WithFiles` serves a folder from disk under a prefix. Give it to `Serve` in the app's `main.go`:

```go
func main() {
	files, err := os.OpenRoot("/data/files")
	if err != nil {
		log.Fatal(err)
	}
	server := warp.New(warp.FromEnv())
	server.Use(app.Modules()...)
	if err := server.Serve(dist, warp.WithFiles("/files/", files.FS())); err != nil {
		log.Fatal(err)
	}
}
```

- Serve reads each file once, at start, for its ETag and type. It reads the bytes from disk for each request, so a large folder is not held in memory.
- A file changed on disk since the start gets a `500`, never a stale ETag. Restart the server after you change the folder.
- It serves files at their exact paths only, never a folder listing.
- It refuses to start, naming the file, if the folder holds a symlink or a name that starts with `.`, or if a path is also a page or file of the build, or under `/api/`.
- Under `weft dev`, `dist` is nil and the dev server serves the pages, so the app serves its API alone and the folder is not served.

Opening the folder with `os.OpenRoot` keeps a symlink added later from leading out of it. This site serves its installer and releases this way.
