---
title: Create an app
description: Make an app with weft create, run it with weft dev while you edit, and build it into one binary with weft build.
order: 11
---

On this page you make an app, run it while you change a page and the API, and build it into one binary that serves both. You need the `weft` command, Go 1.26, Node 22.18 or later and pnpm 10, as [Install](install.md) describes. You need no clone of the Weft repository: `weft` carries Weft's packages.

## Make the app

```sh copy
weft create myapp
```

It writes the app into a new folder, `myapp`, puts Weft's packages in `myapp/.weft/`, and runs `pnpm install`, which fetches Vue, Vite and the other public packages from the npm registry. After pnpm's output it prints:

```text
Created myapp. Next: cd myapp && weft dev
```

The app has:

| Path | What it is |
| --- | --- |
| `web/pages/` | The pages. `index.vue` is `/`, and `404.vue` is the page for an unknown path. |
| `web/components/` | The app's own components. `Hello.vue` is an island that asks the API for a message. |
| `api/` | The Go API, served under `/api`, with a test. |
| `main.go`, `dist.go`, `dist_dev.go` | The Go server. `dist.go` embeds the built pages. |
| `package.json` | Its `"weft"` object records the Weft version and the theme. |
| `.weft/` | Weft's packages and Go package, at the app's version. Git ignores it, and every `weft` command puts it back when it is missing. |
| `AGENTS.md`, `CLAUDE.md` | What an agent working on the app needs to know. |

`--theme lisa` or `--theme weft` picks another theme, and `--module` sets the Go module path, which is the app's name otherwise. See [weft create](../reference/cli.md#weft-create) for every flag and refusal.

## Run it

```sh
cd myapp
weft dev
```

It builds the Go server, starts it beside the page server, and prints the address to open once both answer:

```text
weft dev: http://localhost:5173
```

The page shows three cards and a badge that says `Hello from Go`, which the `Hello` island fetched from `/api/hello`. The page server sends `/api` on to the Go server.

`web/pages/index.vue` imports only `useHead` and `Hello`: Weft's components, such as `Page`, `Section` and `Card`, need no import in a `.vue` file.

## Change a page

In `web/pages/index.vue`, change `title="Welcome"` to `title="My app"` and save. The browser reloads, and the page's heading is `My app`. Each page is rendered when it is asked for, so a save shows on the next request.

## Change the API

In `api/api.go`, change `"Hello from Go"` to `"Hello from my API"` and save. `weft dev` builds the Go code again and restarts the server, and `/api/hello` answers `{"message":"Hello from my API"}`. If the build fails, `weft dev` prints the compiler's output, the last build keeps serving `/api`, and the next save tries again.

Ctrl-C stops both servers.

## Build it

```sh
weft build
```

It builds the pages into `web/dist`, then the Go binary, which embeds them:

```text
weft: built 2 pages into web/dist
Built myapp
```

## Run the binary

```sh
./myapp
```

```text
2026/11/24 22:05:14 myapp on http://localhost:8080
```

It serves the pages and `/api` from one file, and needs no Node. Set `PORT` to listen on another port.

## In a clone of the app

Commit the app without `.weft/` and `node_modules/`, as its `.gitignore` says. Whoever clones it runs `weft dev` or `weft build`, which unpack `.weft/` and run `pnpm install` first. Their `weft` must be at the app's Weft version; `weft upgrade` moves the app to a newer one.

## Next

- [Pages and islands](../guides/pages-and-islands.md): what a page can do, and when a part of it should be an island.
- [Serving from Go](../guides/go-serving.md): `weft.Serve` and the API beside it.
- [The CLI](../reference/cli.md): every command, flag and error.
