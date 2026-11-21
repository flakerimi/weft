---
title: Your first Go handler
description: Serve the built pages and an API from one Go binary.
order: 12
---

On this page you write a Go program that serves the pages you built in [Your first site](first-site.md) and answers one API route. Both come from one binary.

## Write the program

The site's folder lives in the repository, so its Go code is part of the `github.com/flakerimi/weft` module and imports the `weft` package directly. Write `hello/main.go`:

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
	log.Print("hello on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", weft.Serve(dist, api)))
}
```

- `//go:embed all:dist` puts the build in the binary. `all:` also takes files whose names start with `.` or `_`, which a plain `embed` leaves out.
- `weft.Serve(dist, api)` sends paths under `/api/` to `api` and everything else to the build.
- `api` is a plain `http.Handler`, so any router works.

## Run it

Build the pages first, since the binary embeds them, then run the program:

```sh
pnpm --filter hello build
go run ./hello
```

In another terminal, ask for the API and the page:

```sh
curl http://localhost:8080/api/hello
curl -I -H 'Accept-Encoding: br' http://localhost:8080/
```

The API answers `{"message":"Hello from Go"}`. The page comes back with `Content-Encoding: br`: the server sends the brotli file the build made, and compresses nothing itself. Its `ETag` lets a browser that has the page get a `304 Not Modified` instead.

A path with no page, such as `/nope`, gets a 404. Add `pages/404.vue` to give it a page of its own.

## Build one binary

```sh
go build -o hello-site ./hello
```

`hello-site` is the whole site: pages, API and server. Copy it to a machine and run it; it needs no Node and no files beside it. [Publishing](../guides/publishing.md) says more.

## Next

- [Pages and islands](../guides/pages-and-islands.md): more pages, and the parts that run in the browser.
- [Serving from Go](../guides/go-serving.md): everything `weft.Serve` does.
