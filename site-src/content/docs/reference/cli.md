---
title: CLI
description: Every command, flag and error of weft, and the installer's settings.
order: 35
---

What is planned and not built is under [Coming next](../coming-next.md). The page build's own commands, which `weft` runs, are in [Page tools](page-tools.md).

## weft

```text
weft — one UI system for Go and Vue applications

Usage: weft <command>

Commands:
  build      build the pages, then the app's binary (-o <file>)
  catalog    list the components, print one (weft catalog <Name>) or search (--find)
  create     make an app (weft create <name>)
  dev        run the app, restarting Go when a .go file or .env changes (--port n)
  gen        write web/.weft-types/api.ts, the typed API client and form rules, from the app's Go
  routes     list the app's routes: method, path, module, name and permission
  scaffold   write a resource: its model, routes, test and pages (weft scaffold <name> <field[:type]>...)
  unpack     make sure .weft/ is unpacked and current, without installing
  update     update weft to the latest release (--check, --version X)
  upgrade    move the app to this weft's Weft version (--no-install)
  version    print the Weft version
```

`weft`, `weft help`, `weft -h` and `weft --help` print this. An unknown command prints `weft: unknown command "x". Run "weft help".` and exits with status 2. `-h` or `--help` after a command prints that command's usage, such as `Usage is weft dev [--port n].`, with exit status 0.

### weft version

Prints the version and the platform it was built for:

```text
weft 0.0.3 (darwin/arm64)
```

A build that is not a release says `0.0.0-dev`.

### weft update

Replaces the running `weft` with another release from `weft.base.al`.

| Flag | What it does |
| --- | --- |
| `--check` | Only says whether the latest release differs from yours, and changes nothing. After a rollback of the site the latest may be older than yours. |
| `--version X` | Installs release `X` instead of the latest, older ones included |

It downloads your platform's archive and `SHA256SUMS`, checks one against the other, and renames the new binary into place (through a symlink to the real file), the old one whole until then. It says `weft X is up to date.` or `Updated weft X → Y.`; a failure is one sentence (exit 1), an unknown flag exits 2, and a development build says to run the installer. `WEFT_BASE` is where the releases are (default `https://weft.base.al`).

### weft create

Makes an app in a new folder named after it: `data` (SQLite), `auth` (the owner's login, with `/login` and `/setup`) and `hello`. It writes the app's files from the template `weft` carries, with a fresh `AUTH_SECRET` (48 random bytes, base64url) in `.env`, which only you may read, unpacks Weft's packages into `.weft/`, records the Weft version and theme in `package.json`'s `"weft"` object, and runs `pnpm install`, `go mod download` (Warp's module, from GitHub at the release's tag, and the SQLite driver) and `weft gen`. Then it prints `Set AUTH_OWNER_EMAIL in <name>/.env to the owner's email address; it is owner@example.com until you do.` and `Created <name>. Next: cd <name> && weft dev`. [Create an app](../getting-started/create-an-app.md) walks through it.

```text
weft create <name> [--module <path>] [--theme neutral|lisa|weft] [--no-install]
```

| Flag | What it does |
| --- | --- |
| `--module <path>` | The Go module path in `go.mod`. Default: the name |
| `--theme` | `neutral` (the default), `lisa` or `weft`. See [Themes](../guides/themes.md) |
| `--no-install` | Skip `pnpm install`, `go mod download` and `weft gen`, which `weft dev` or `weft build` runs later |

The name is a lowercase letter, then lowercase letters, digits and hyphens, at most 64. It is also the binary's name, so `web`, `api`, `app` and `dist` are refused: the binary would land in a folder of the app. Wrong usage, such as a bad name, theme or flag, is one sentence with exit status 2.

Before it writes anything it checks that `go`, `node` 22.18 or later and `pnpm` 10 or later are on `PATH`, and that the folder is missing or empty; otherwise it says `weft create: <name> already exists and is not empty; choose another name or move it away.`, or names the missing tool, with exit status 1. If `pnpm install` fails, the app stays, and it says `weft create: pnpm install did not finish; run pnpm install in <name> to finish.` If Go cannot download the modules, it says so, naming the network and `GOPROXY`.

### weft scaffold

Writes one resource into an app `weft create` made, then runs `weft gen`.

```text
weft scaffold <name> <field[:type[:modifier...]]>... [--plural <word>] [--api]
weft scaffold product name:string:required:max=120:unique price:float:min=0 sale:float:nullable status:enum=draft,live:required
```

`<name>` is singular snake_case (`line_item`). Its plural, made by a fixed rule (`person` → `people`, `category` → `categories`, `box` → `boxes`, else `s`) or given with `--plural`, names the package `app/<plural>`, the table, `/api/<plural>` and the permissions. Refused: `data`, `auth`, `warp`, `app`, `web`, `test`, `main`, Go's words and standard packages, names the story test uses, and names whose type Weft or the client has (`user`, `module`, `record`, `list`, `login`…).

A field's type is written, never guessed from its name; with none it is `string`.

| Type | Go | Rule | `data` tag |
| --- | --- | --- | --- |
| `string` | `string` | | `sort,search` |
| `text` | `string` | | `search` |
| `email`, `url` | `string` | `email`, `url` | `sort,search` |
| `int`, `float` | `int64`, `float64` | | `sort` |
| `bool` | `bool` | | `sort,filter` |
| `date`, `datetime` | `time.Time`, RFC 3339 | | `sort`; with `--api` only, until 0.2.0 |
| `enum=a,b` | `string` | `enum=a\|b` | `sort,filter` |

The modifiers, in any order after the type, are `nullable`, a pointer that may be `null` (`sale:float:nullable` is `*float64`); `required` (not on `bool` or with `nullable`); `unique` (not on `bool`, `text`, `date`, `datetime`); and `min=N` and `max=N`, in characters for text (not on `bool`, `date`, `datetime`, `enum`); a field has only the rules written. Enum choices are separated by commas: `|` or `(` is refused with `weft scaffold: write enum choices separated by commas after enum=, as in status:enum=draft,live.`, and a `?` with the `:nullable` form.

| It writes | What it is |
| --- | --- |
| `app/<plural>/model.go` | The model, embedding `data.Record` |
| `app/<plural>/routes.go` | `data.Resource`, with `data.PerAction(auth.Per("<plural>"))` when `app/modules.go` lists `auth.Module()`, else open, which it says |
| `app/<plural>/<plural>_test.go` | The story over HTTP on SQLite: make two, list sorted, searched and filtered, change, delete, and, with `auth`, 401 without a session |
| `app/modules.go` | The import and `<plural>.Module()` last in the list; the rest, comments too, kept |
| `web/components/<Plural>Table.vue`, `<Singular>View.vue` | Islands: a `DataTable` over `api.<plural>` and `schema.<Singular>Input`, and one record |
| `web/pages/<plural>/index.vue`, `[id].vue` | The table's page and the record's run-time page |

`--api` writes only the Go; it is needed for `date` and `datetime`, whose controls in the pages land in 0.2.0 ([Coming next](../coming-next.md#scaffolding-and-app-blocks)). It lists what it wrote under `Scaffolded <name> as <plural>:`.

Nothing is written, with one sentence and exit status 1, when `app/<plural>` is there, or without `--api` the page folder, `web/pages/<plural>.vue` or an island, or a `date` or `datetime` field, or when `app/modules.go` has no `func Modules() []warp.Module` returning a `[]warp.Module{...}` with `data.Module()`. `id`, `created_at` and `updated_at` (from `data.Record`), `actions` (`DataTable`'s), a field given twice, an unknown type or modifier, or a modifier its type does not take exit 2. A later release's type exits 1 naming it, as `weft scaffold: field "maker" has type "belongs_to", and relations land in Weft 0.3.0 (W3); use a scalar type for now.`: `belongs_to`, `has_many`, `many_to_many` (0.3.0), `image`, `file` (0.4.0), `json`, `translated` (0.6.0).

### weft dev

Runs an app while you work. It builds the Go code with the `dev` tag into `.weft/dev/app` and runs it on a free port, starts [weft-dev](page-tools.md#weft-dev) beside it with the app's theme, which renders each page when asked for and sends `/api` to the Go server, and prints `weft dev: http://localhost:<n>` once both answer. Open it with `localhost`, not `127.0.0.1`: the page server may listen on `::1` only. It checks the app as `weft build` does, and reads the theme once.

| Flag | What it does |
| --- | --- |
| `--port n` | The page server's port. Default `5173`; `0` takes a free one |

Every 300 ms it looks for a changed `.go` file outside `.weft/`, `.git/`, `node_modules/` and `web/`, and a changed `.env`; then it builds again and, if that succeeds, replaces the Go server. A failed build prints the compiler's output, and the last build serves `/api` until the fix is saved; a rebuilt server that stops by itself says `weft dev: the Go server stopped after the restart; fix it and save again.` An unreadable folder is skipped and named once. A change under `web/` reloads the browser; one to `web/vite.config` needs a restart. The Go server's output passes through. It runs with a free `PORT`, `HOST=127.0.0.1`, `WEFT_DEV_PARENT=1` and `APP_URL=http://localhost:<the page server's port>` (or a non-empty `APP_URL` from `weft dev`'s environment), which win over `.env`.

The Go server must answer on `PORT` within 30 s, as must the page server, or `weft dev` stops both, saying which and what to check, with exit status 1. The app has 5 s to describe itself after each build, else `weft dev` says so and goes on with `web/.weft-types/api.ts` as it was.

Ctrl-C, SIGTERM, SIGHUP or SIGQUIT stops both servers, with exit status 0; killed, `weft dev` leaves neither running, as each exits when its standard input, a pipe `weft dev` holds, ends. If one server stops by itself, `weft dev` stops the other and says `weft dev: the page server stopped; see its output above.` (or `the Go server`), with exit status 1. A port in use is `weft dev: port <n> is in use; stop what is using it or pass --port.`, checked first. Wrong usage exits with status 2.

### weft build

Builds an app: writes the typed client as [weft gen](#weft-gen) does, type-checks the pages against it with `vue-tsc -p web/tsconfig.json`, stopping at an error, runs `weft-build-pages --root web --out web/dist` with the app's theme, then builds the Go binary, which embeds `web/dist`, with

```text
go build -trimpath -ldflags "-s -w" -o <file> .
```

It adds `CGO_ENABLED=0`, for a static binary, unless the environment sets it. `GOOS` and `GOARCH` pass through from the environment, so `GOOS=linux GOARCH=arm64 weft build` builds for Linux on arm64. Run it anywhere inside the app; it finds the nearest `package.json` with a `"weft"` object. A `package.json` on the way that is not valid JSON stops the search: the command names it and says to fix it.

| Flag | What it does |
| --- | --- |
| `-o <file>` | The binary to write, from where you run it. Default: the app's `package.json` name, in the app's folder |

It prints the build's output, then `Built <file>`. First it checks the app:

- An app recorded at another Weft version is refused: `weft build: this app is on Weft <a> and this weft is <b>; run weft upgrade to move the app, or weft update --version <a> to match it.` For an app on a newer Weft than this `weft`, which `weft upgrade` would refuse, the sentence ends `run weft update --version <a> to match it.` alone. An app whose `"weft"` object has no `version` is refused with a sentence that says what to add.
- When `.weft/` is missing or holds another version, as in a fresh clone, it unpacks the packages into it and runs `pnpm install`; a `weft` built from the repository (`0.0.0-dev`) also when the bundle's files changed (`.weft/VERSION` holds their hash). It runs `pnpm install` too when `node_modules/` is missing or older than `.weft/`, and `go mod download` when `go.sum` lacks Warp's module. A `.weft/` without the `VERSION` file is not `weft`'s and is left alone, with a sentence. While it unpacks it holds `.weft.lock`, which a second `weft` waits for.
- After an install, or when `web/.weft-types/components.d.ts` is missing, it writes that file with [weft-types](page-tools.md#weft-types), so `pnpm typecheck` checks the components' props before any build.

Outside an app it says `weft build: no Weft app here; run it in a folder made by weft create.` Any failure is one sentence on standard error, with exit status 1; wrong usage exits with status 2.

### weft unpack

Makes `.weft/` present and current as `weft build` does, with no `pnpm install`, so in a fresh clone whose `go.mod` replaces Warp's module with `./.weft/go`, `go test` and an editor resolve it. It prints `Unpacked .weft (Weft <version>)` or `.weft is current (Weft <version>)`.

### weft gen

Writes `web/.weft-types/api.ts` from the app's Go code. It builds the app with the `dev` tag, as `weft dev` does, and runs it with `WARP_DESCRIBE=1`, a free `PORT` and `HOST=127.0.0.1`, so warp's `Serve` prints the routes and types as JSON. From that it writes an interface per Go type, `api.<module>.<name>()` per typed route (`api.app` for routes outside a module), the `ApiError` every failed call rejects with, and `schema`, the rules of each body type for a form to check before it sends; `unique` and the rules of nested types are checked by the server only, and arrive as its field errors. It prints `Wrote web/.weft-types/api.ts`, or `web/.weft-types/api.ts is up to date` and leaves the file alone. `weft dev` runs it after each Go build that succeeds, and `weft build` before the page build. Never edit the file. An app no package of which calls `warp.New`, as `go list` finds them with the `dev` tag, is not run: it gets the client of an app with no routes, which `weft gen` says, or keeps one that has routes, saying so (`weft gen` exits 1).

A compile error, an app that exits with an error or prints something else or nothing, or one still running after 30 s (stopped) is one sentence after its output, exit 1, with the file as it was. `weft build` stops on the first two only; on the others it says so, as `weft dev` does, and goes on. A name the file cannot use (two routes that become one in camelCase, a type named as TypeScript reserves, an optional field, or any field of a partial body's type, named `constructor` or `toString`) is a sentence that says what to rename.

### weft routes

Builds and describes the app as `weft gen` does, and prints its routes in the order registered; an app with no `warp.App` gets a sentence saying so. A route that takes every method is `ANY`; no module or no permission is `-`.

```text
METHOD  PATH                MODULE    NAME    PERMISSION
POST    /api/auth/login     auth      login   -
POST    /api/auth/logout    auth      logout  -
GET     /api/auth/me        auth      me      -
POST    /api/auth/setup     auth      setup   -
GET     /api/hello          hello     hello   -
GET     /api/products       products  list    products:read
GET     /api/products/{id}  products  show    products:read
POST    /api/products       products  create  products:create
PATCH   /api/products/{id}  products  update  products:edit
DELETE  /api/products/{id}  products  remove  products:delete
```

### weft upgrade

Moves an app to the Weft version of the `weft` you run: it rewrites `weft.version` in `package.json`, changing no other byte, moves `go.mod`'s require of Warp's module to that release, dropping a replace into `./.weft/go` (another is kept, and named), unpacks that version's packages into `.weft/`, and runs `pnpm install` and `go mod download`. It prints `Moved <name> from Weft <a> to <b>`, or `<name> is already on Weft <b>` when there is nothing to do. It never moves an app backwards: an app on a newer Weft is refused (exit 1), saying to run `weft update`; `0.0.0-dev` is older than any release. `--no-install` skips `pnpm install` and `go mod download`.

#### Moving an app to Warp

An app made by Weft 0.0.6 or before calls `weft.Serve` in `main.go`, which is now `warp.Serve`, so once upgraded it does not compile (`undefined: weft.Serve`). `weft upgrade` says so after its output, and still exits 0: `weft upgrade: main.go still calls weft.Serve, which moved; see https://weft.base.al/docs/reference/cli#moving-an-app-to-warp for the two lines to change.` The smallest change is two lines of `main.go`: the import of `github.com/flakerimi/weft` becomes the first line below, and the `weft.Serve` call in `handler` the second.

```go
	"github.com/flakerimi/weft/warp"

		return warp.Serve(dist, api.Handler())
```

The `listen` function stays as it is. Such an app has no `warp.App`, so `weft build` and `weft gen` give it a client with no routes at once; an app made by 0.0.7 has one, so after `weft upgrade` they write its routes. To use modules and typed routes, build a `warp.App` and serve it with its own `Serve`, as a new app's `main.go` does ([Warp, the Go side](../guides/warp.md)). An App passed to `warp.Serve` stops them, saying to call its own `Serve`.

### weft catalog

Prints the component catalogue in pieces, so an agent reads only the components it uses. Inside an app it reads `.weft/catalog/components.json`; outside one, or before `.weft/` is unpacked, the catalogue `weft` carries.

```text
weft catalog [<Name>...] [--examples] [--json]
weft catalog --find <word> [<word>...] [--json]
```

With no name it lists every component under its group, its name and its purpose:

```text
76 components. Run weft catalog <Name> for one.

disclosure
  Accordion       Stacked sections whose content opens under its header.
  Collapsible     Content shown and hidden by one trigger.
```

With names it prints each component as Markdown (`# Kbd`, then its purpose): `Use:` and `Avoid:`, `Adapts:` on each axis, `Props:` as `name: type = default`, with `required` where it is, then what it does, `Emits:` with each payload, `Slots:` with each slot's props, `Notes:`, and its first example; a section it has nothing for is left out. Names match without regard to case or hyphens, so `date-picker` finds `DatePicker`.

| Flag | What it does |
| --- | --- |
| `--examples` | Every example of each component named, not only the first |
| `--json` | The entries as the catalogue has them, as a JSON array: every one, the ones named, or the ones found |
| `--find` | Search instead of naming: the components whose name has each word, or whose purpose, use or notes have it as a whole word, without regard to case. Words may be separate or in one argument. |

`--find` prints the list's lines of the components that match every word, in the catalogue's order. When nothing matches it says `weft catalog: nothing matches "<words>"; run weft catalog to list them.`, with exit status 1. `--find` with no words, or only blank ones, is wrong usage.

An unknown name prints nothing for the others and exits with status 1, naming up to three components it starts or is part of, then ones whose text has it as a word, then names within a few edits (`weft catalog: no component named "modal"; did you mean Dialog, OverlayPanel, Sheet?`), or saying `run weft catalog to list them.` It never prints another component in its place. Wrong usage exits with status 2.

## The installer

`curl -fsSL https://weft.base.al/install | sh` installs the latest release. It runs on macOS and Linux, on arm64 and amd64, and needs `curl`, and `sha256sum` or `shasum`.

| Variable | What it does | Default |
| --- | --- | --- |
| `WEFT_VERSION` | The release to install | the latest |
| `WEFT_BIN_DIR` | Where to put `weft` | `$HOME/.local/bin` |
| `WEFT_BASE` | Where the releases are | `https://weft.base.al` |
