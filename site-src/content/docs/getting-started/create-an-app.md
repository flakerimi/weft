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

It writes the app into a new folder, `myapp`, puts Weft's packages in `myapp/.weft/`, and runs `pnpm install`, which fetches Vue, Vite and the other public packages from the npm registry, and `go mod download`, which fetches Warp from GitHub. Then it prints:

```text
Created myapp. Next: cd myapp && weft dev
```

If Go cannot fetch Warp (offline, or behind a proxy that refuses it), `weft create` says so and leaves the app; once Go can reach the module proxy, `cd myapp && weft dev` (or `weft build`) fetches it and finishes.

The app has:

| Path | What it is |
| --- | --- |
| `web/pages/` | The pages. `index.vue` is `/`, and `404.vue` is the page for an unknown path. |
| `web/components/` | The app's own components. `Hello.vue` is an island that asks the API for a message. |
| `app/modules.go` | The one list of the app's Go modules. |
| `app/hello/` | The first module: `routes.go` has the route `GET /api/hello`, and `hello_test.go` its test. |
| `main.go`, `dist.go`, `dist_dev.go` | The Go server, a `warp.App` from `github.com/flakerimi/weft/warp`. `dist.go` embeds the built pages. |
| `.env`, `.env.sample` | The server's settings, such as `PORT`. Git ignores `.env`; `.env.sample` is committed. |
| `package.json` | Its `"weft"` object records the Weft version and the theme. |
| `go.mod`, `go.sum` | `go.mod` requires Warp, the module `github.com/flakerimi/weft`, at the release's tag, `v<version>`; `go.sum` holds its checksums. Commit both. `weft upgrade` moves the version. |
| `.weft/` | Weft's packages, at the app's version. Git ignores it, and every `weft` command puts it back when it is missing. |
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

In `app/hello/routes.go`, change `"Hello from Go"` to `"Hello from my API"` and save. `weft dev` builds the Go code again and restarts the server, and `/api/hello` answers `{"message":"Hello from my API"}`. If the build fails, `weft dev` prints the compiler's output, the last build keeps serving `/api`, and the next save tries again.

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
2026/11/25 17:40:59 listening on http://localhost:8080
```

It serves the pages and `/api` from one file, and needs no Node. It reads `PORT` from `.env`, and the environment wins over it: `PORT=3000 ./myapp` listens on port 3000. Ctrl-C stops it.

## In a clone of the app

Commit the app without `.weft/` and `node_modules/`, as its `.gitignore` says. Whoever clones it runs `weft dev` or `weft build`, which unpack `.weft/` and run `pnpm install` first. Their `weft` must be at the app's Weft version; `weft upgrade` moves the app to a newer one.

## Next

- [Your first page](first-site.md): add a page of your own.
- [Your first Go route](first-go-handler.md): add a route that checks what it is sent.
- [The CLI](../reference/cli.md): every command, flag and error.
