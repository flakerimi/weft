---
title: Create an app
description: Make an app with weft create, add a resource with weft scaffold, run it with weft dev while you edit, and build it into one binary with weft build.
order: 11
---

On this page you make an app, add a resource to it, run it while you change a page and the API, and build it into one binary that serves both. You need the `weft` command, Go 1.26, Node 22.18 or later and pnpm 10, as [Install](install.md) describes. You need no clone of the Weft repository: `weft` carries Weft's packages.

## Make the app

```sh copy
weft create myapp
```

It writes the app into a new folder, `myapp`, puts Weft's packages in `myapp/.weft/`, and runs `pnpm install`, which fetches Vue, Vite and the other public packages from the npm registry, `go mod download`, which fetches Warp from GitHub and the SQLite driver, and `weft gen`, which writes the typed client. Then it prints:

```text
Set AUTH_OWNER_EMAIL in myapp/.env to the owner's email address; it is owner@example.com until you do.
Created myapp. Next: cd myapp && weft dev
```

The app keeps its records in SQLite, and its first user is its owner, whose email is `AUTH_OWNER_EMAIL` in `.env`: set it to yours. The owner invites the others, whose mail is printed in the app's log until `MAIL=smtp` ([Accounts](../guides/accounts.md)). `.env` also holds `AUTH_SECRET`, 48 random bytes made for this app, which sign the session cookie.

If Go cannot fetch them (offline, or behind a proxy that refuses it), `weft create` says so and leaves the app; once Go can reach the module proxy, `cd myapp && weft gen` (or `weft dev`) fetches them and finishes.

The app has:

| Path | What it is |
| --- | --- |
| `web/pages/` | The pages. `index.vue` is `/`, `login.vue` signs in, `setup.vue` sets the owner's password, `register.vue`, `forgot.vue` and `password.vue` ask for and use a mailed link, `account.vue` and `users.vue` manage accounts, and `404.vue` is the page for an unknown path. |
| `web/components/` | The app's own components: `Hello.vue`, an island that asks the API for a message, and the forms and panels of those pages. |
| `web/answers.ts`, `web/session-check.ts` | What a form says when a call fails, and a check that the generated `Me` fits `@weft/ui`'s session. |
| `app/modules.go` | The one list of the app's Go modules: `middleware`, `data`, `mail`, `auth` and `hello`. |
| `app/hello/` | The first module: `routes.go` has the route `GET /api/hello`, which answers anyone, and `hello_test.go` its test. |
| `main.go`, `dist.go`, `dist_dev.go` | The Go server, a `warp.App` from `github.com/flakerimi/weft/warp`, with the SQLite driver. `dist.go` embeds the built pages. |
| `.env`, `.env.sample` | The settings: the port, the database, `AUTH_SECRET` and `AUTH_OWNER_EMAIL`, and in `.env.sample` the others, commented with their defaults. Git ignores `.env` and `data/`, where the database is; `.env.sample` is committed and says what each one is. |
| `package.json` | Its `"weft"` object records the Weft version and the theme. |
| `go.mod`, `go.sum` | `go.mod` requires Warp, the module `github.com/flakerimi/weft`, at the release's tag, `v<version>`; `go.sum` holds its checksums. Commit both. `weft upgrade` moves the version. |
| `.weft/` | Weft's packages, at the app's version. Git ignores it, and every `weft` command puts it back when it is missing. |
| `AGENTS.md`, `CLAUDE.md` | What an agent working on the app needs to know. |

`--theme lisa` or `--theme weft` picks another theme, and `--module` sets the Go module path, which is the app's name otherwise. See [weft create](../reference/cli.md#weft-create) for every flag and refusal.

## Scaffold a resource

The app's records are resources: a model, its five routes and its test, in a module of its own. `weft scaffold` writes one from its name and typed fields:

```sh copy
cd myapp && weft scaffold product name:string:required:unique price:float:min=0 status:enum=draft,live:required --api
```

```text
Scaffolded product as products:
  app/products/model.go
  app/products/routes.go
  app/products/products_test.go
  app/modules.go (products.Module() added)
Wrote web/.weft-types/api.ts
```

`app/products/` is the model, `Product`, and `data.Resource`, which answers `GET`, `POST`, `PATCH` and `DELETE` under `/api/products`, each behind the owner's login; `app/modules.go` lists it last; and `weft gen` has written `api.products.list()`, `show`, `create`, `update` and `remove` into the client, with the form rules in `schema.ProductInput`. `go test ./...` runs the story test it wrote, over a database of its own. Without `--api` it also writes the pages `/products`, a `DataTable` that searches, sorts, filters and pages on the server and makes, changes and deletes products in an `EntityDialog`, and `/products/<id>`. Every type and modifier is in [weft scaffold](../reference/cli.md#weft-scaffold).

## Run it

```sh
weft dev
```

It builds the Go server, starts it beside the page server, and prints the address to open once both answer:

```text
2026/11/27 09:00:00 Set up the owner at http://localhost:5173/setup#token=…
weft dev: http://localhost:5173
```

Open the setup link to choose the owner's password; it works once, while no user exists. The front page shows three cards and a badge that says `Hello from Go`, which the `Hello` island fetched from `/api/hello`. The page server sends `/api` on to the Go server.

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
weft: built 4 pages into web/dist
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

Commit the app without `.weft/` and `node_modules/`, as its `.gitignore` says. Whoever clones it runs `weft dev` or `weft build`, which unpack `.weft/` and run `pnpm install` first. CI that runs only Go runs `weft unpack` first: `web/dist`, which git ignores, then holds the empty marker `dist.go` embeds, and `go test ./...` builds. Their `weft` must be at the app's Weft version; `weft upgrade` moves the app to a newer one.

## Next

- [Your first page](first-site.md): add a page of your own.
- [Your first Go route](first-go-handler.md): add a route that checks what it is sent.
- [The CLI](../reference/cli.md): every command, flag and error.
