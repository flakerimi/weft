---
title: CLI
description: Every command, flag and error of weft, weft-build-pages and weft-dev, and the installer's settings.
order: 35
---

What is planned and not built is under [Coming next](../coming-next.md).

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

It downloads the archive for your platform and `SHA256SUMS`, checks the archive against it, writes the new binary beside the running one and renames it into place, so the old one stays whole until the new one is complete. It follows a symlink to the real file.

It says `weft X is up to date.` when there is nothing to do, and `Updated weft X → Y.` when it is done. Any failure is one sentence on standard error, with exit status 1; an unknown flag exits with status 2. A development build refuses to update and says to run the installer.

| Variable | What it does |
| --- | --- |
| `WEFT_BASE` | Where the releases are. Default `https://weft.base.al` |

### weft create

Makes an app in a new folder named after it. It writes the app's files from the template `weft` carries, unpacks Weft's packages into the app's `.weft/`, records the Weft version and theme in `package.json`'s `"weft"` object, and runs `pnpm install` and `go mod download`, which fetches Warp's module from GitHub at the release's tag. Then it prints `Created <name>. Next: cd <name> && weft dev`. [Create an app](../getting-started/create-an-app.md) walks through what it writes.

```text
weft create <name> [--module <path>] [--theme neutral|lisa|weft] [--no-install]
```

| Flag | What it does |
| --- | --- |
| `--module <path>` | The Go module path in `go.mod`. Default: the name |
| `--theme` | `neutral` (the default), `lisa` or `weft`. See [Themes](../guides/themes.md) |
| `--no-install` | Skip `pnpm install` and `go mod download`, which `weft dev` or `weft build` runs later |

The name is a lowercase letter, then lowercase letters, digits and hyphens, at most 64. It is also the binary's name, so `web`, `api`, `app` and `dist` are refused: the binary would land in a folder of the app. Wrong usage, such as a bad name, theme or flag, is one sentence with exit status 2.

Before it writes anything it checks that `go`, `node` 22.18 or later and `pnpm` 10 or later are on `PATH`, and that the folder is missing or empty; otherwise it says `weft create: <name> already exists and is not empty; choose another name or move it away.`, or names the missing tool, with exit status 1. If `pnpm install` fails, the app stays, and it says `weft create: pnpm install did not finish; run pnpm install in <name> to finish.`

### weft dev

Runs an app while you work. It builds the Go code with the `dev` tag into `.weft/dev/app` and runs it on a free port, starts [weft-dev](#weft-dev) beside it with the app's theme, which renders each page when asked for and sends `/api` to the Go server, and prints `weft dev: http://localhost:<n>` once both answer. Open that address with `localhost`, not `127.0.0.1`: the page server may listen on `::1` only. Run it anywhere inside the app; it checks the app as `weft build` does. It reads the theme once: restart it after changing `"theme"` in `package.json`.

| Flag | What it does |
| --- | --- |
| `--port n` | The page server's port. Default `5173`; `0` takes a free one |

Every 300 ms it looks for a changed `.go` file outside `.weft/`, `.git/`, `node_modules/` and `web/`, and for a changed `.env` in the app's folder. When one changed it builds again and, if that succeeds, replaces the running Go server. A failed build prints the compiler's output; the last build serves `/api` until you save the fix. A rebuilt server that stops by itself, such as one that panics as it starts, prints `weft dev: the Go server stopped after the restart; fix it and save again.`, and the next save builds and starts it again. An unreadable folder is skipped and named once. A change under `web/` reloads the browser and needs no build; a change to `web/vite.config` needs a restart. The Go server's own output passes through. It runs with `PORT` set to a free port, `HOST=127.0.0.1`, `WEFT_DEV_PARENT=1` and `APP_URL=http://localhost:<the page server's port>`, which win over `.env`, so it listens on `127.0.0.1` only and prints no address. A non-empty `APP_URL` in `weft dev`'s own environment is passed on instead.

The Go server must listen on `PORT`, which warp's `Serve` reads. If it does not answer there within 30 s of starting, `weft dev` stops both servers and says `weft dev: the Go server did not answer within 30 s; check that main.go serves the app (warp's Serve reads PORT), and run weft dev again.`, with exit status 1. The page server has the same 30 s, after which it says `weft dev: the page server did not answer within 30 s; see its output above.` The app has 5 s to describe itself after each build, else `weft dev: the app did not describe itself within 5 s, so web/.weft-types/api.ts is as it was; make sure main.go calls Serve before doing other work.` and goes on.

Ctrl-C, SIGTERM, closing the terminal (SIGHUP) or `Ctrl-\` (SIGQUIT) stops both servers, with exit status 0. Killed, as by SIGKILL, it leaves neither running: each exits when its standard input, a pipe `weft dev` holds, ends. If the page server, or the Go server it started with, stops by itself, `weft dev` stops the other and says `weft dev: the page server stopped; see its output above.` (or `the Go server`), with exit status 1. A port in use is the sentence `weft dev: port <n> is in use; stop what is using it or pass --port.`, checked before anything else, so nothing is installed or started. Wrong usage exits with status 2.

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
- When `.weft/` is missing or holds another version, as in a fresh clone, it unpacks the packages into it and runs `pnpm install`. A `weft` built from the repository (`0.0.0-dev`) also unpacks again when the bundle's files changed, which `.weft/VERSION` records as a hash on its second line; a release is the version it says. It runs `pnpm install` too when `node_modules/` is missing, and `go mod download` when `go.sum` lacks Warp's module. A `.weft/` without the `VERSION` file `weft` writes is not its own and is left alone, with a sentence saying to move it out of the app folder. Beside `.weft/` only `.weft.lock` is made, while it unpacks; a second `weft` waits for it, and takes over at once one a killed `weft` left.
- After an install, or when `web/.weft-types/components.d.ts` is missing, it writes that file with [weft-types](#weft-types), so `pnpm typecheck` checks the components' props before any build.

Outside an app it says `weft build: no Weft app here; run it in a folder made by weft create.` Any failure is one sentence on standard error, with exit status 1; wrong usage exits with status 2.

### weft unpack

Makes `.weft/` present and current as `weft build` does first, with no `pnpm install`. In a fresh clone of an app a development build made, whose `go.mod` replaces Warp's module with `./.weft/go`, it lets `go test ./...` and an editor resolve it. It prints `Unpacked .weft (Weft <version>)` or `.weft is current (Weft <version>)`, and refuses an app on another Weft version as `weft build` does.

### weft gen

Writes `web/.weft-types/api.ts` from the app's Go code. It builds the app with the `dev` tag, as `weft dev` does, and runs it with `WARP_DESCRIBE=1`, a free `PORT` and `HOST=127.0.0.1`, so warp's `Serve` prints the routes and types as JSON. From that it writes an interface per Go type, `api.<module>.<name>()` per typed route (`api.app` for routes outside a module), the `ApiError` every failed call rejects with, and `schema`, the rules of each body type for a form to check before it sends; `unique` and the rules of nested types are checked by the server only, and arrive as its field errors. It prints `Wrote web/.weft-types/api.ts`, or `web/.weft-types/api.ts is up to date` and leaves the file alone. `weft dev` runs it after each Go build that succeeds, and `weft build` before the page build. Never edit the file. An app no package of which calls `warp.New`, as `go list` finds them with the `dev` tag, is not run: it gets the client of an app with no routes, which `weft gen` says, or keeps one that has routes, saying so (`weft gen` exits 1).

A compile error ends `weft gen: go build failed; fix what its output above says and run weft gen again.`; an app that exits with an error, or prints something else or nothing, its own output and `weft gen: the app did not describe itself; fix what its output above says and run weft gen again.`; one still running after 30 s is stopped. Each exits 1 and leaves the file as it was. `weft build` stops on the first two only; on the others it says so, as `weft dev` does, and goes on. A name the file cannot use (two routes that become one in camelCase, a type named as TypeScript reserves, an optional field, or any field of a partial body's type, named `constructor` or `toString`) is a sentence that says what to rename.

### weft routes

Builds and describes the app as `weft gen` does, and prints its routes in the order registered; an app with no `warp.App` gets a sentence saying so. A route that takes every method is `ANY`; no module or no permission is `-`.

```text
METHOD  PATH        MODULE  NAME   PERMISSION
GET     /api/hello  hello   hello  -
```

### weft upgrade

Moves an app to the Weft version of the `weft` you run: it rewrites `weft.version` in `package.json`, changing no other byte, moves `go.mod`'s require of Warp's module to that release, dropping a replace into `./.weft/go` (another is kept, and named), unpacks that version's packages into `.weft/`, and runs `pnpm install` and `go mod download`. It prints `Moved <name> from Weft <a> to <b>`, or `<name> is already on Weft <b>` when there is nothing to do. It never moves an app backwards: an app on a newer Weft is refused with `weft upgrade: this app is on Weft <a>, newer than this weft <b>; run weft update to get <a> or later.`, and exit status 1. A development build's `0.0.0-dev` is older than any release.

| Flag | What it does |
| --- | --- |
| `--no-install` | Skip `pnpm install` and `go mod download` |

#### Moving an app to Warp

An app made by Weft 0.0.6 or before calls `weft.Serve` in `main.go`, which is now `warp.Serve`, so once upgraded it does not compile (`undefined: weft.Serve`). `weft upgrade` says so after its output, and still exits 0: `weft upgrade: main.go still calls weft.Serve, which moved; see https://weft.base.al/docs/reference/cli#moving-an-app-to-warp for the two lines to change.` The smallest change is two lines of `main.go`: the import of `github.com/flakerimi/weft` becomes the first line below, and the `weft.Serve` call in `handler` the second.

```go
	"github.com/flakerimi/weft/warp"

		return warp.Serve(dist, api.Handler())
```

The `listen` function stays as it is. Such an app has no `warp.App`, so `weft build` and `weft gen` give it a client with no routes at once. An app made by 0.0.7 has one, so after `weft upgrade` they write its routes. To use modules and typed routes, build a `warp.App` and serve it with its own `Serve`, as a new app's `main.go` does (see [Warp, the Go side](../guides/warp.md)); then they write its routes. An App passed to `warp.Serve` stops them, saying to call its own `Serve`.

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

With names it prints each component as Markdown: its purpose, `Use:` and `Avoid:`, `Adapts:` on each axis, `Props:` as `name: type = default`, with `required` where it is, then what it does, `Emits:` with each payload, `Slots:` with each slot's props, `Notes:`, and its first example. A section it has nothing for is left out. Names match without regard to case or hyphens, so `date-picker` finds `DatePicker`. Kbd here is shortened:

````text
# Kbd

A key on the keyboard, for showing shortcuts.

Use: In help text, menus and tooltips, one Kbd per key of a shortcut.
Avoid: For code or a value to copy, use a code element.

Adapts:
- …

Props:
- value: string — The key: a character (`K`) or a name with a symbol (`command`, `shift`…). …

Slots: default

Notes:
…
````

| Flag | What it does |
| --- | --- |
| `--examples` | Every example of each component named, not only the first |
| `--json` | The entries as the catalogue has them, as a JSON array: every one, the ones named, or the ones found |
| `--find` | Search instead of naming: the components whose name has each word, or whose purpose, use or notes have it as a whole word, without regard to case. Words may be separate or in one argument. |

`--find` prints the list's lines of the components that match every word, in the catalogue's order:

```text
$ weft catalog --find week
  Calendar        A month of dates to choose one, several or a range from, in the locale's names, digits and first day of the week.
  DatePicker      A date field where a date is typed as the locale writes it or chosen from a calendar that opens beside it.
  WeftProvider    Sets how the part of the page inside it is used, its input and its surface, for every Weft component within.
```

When nothing matches it says `weft catalog: nothing matches "<words>"; run weft catalog to list them.`, with exit status 1. `--find` with no words, or only blank ones, is wrong usage.

An unknown name prints nothing for the others and exits with status 1, naming up to three components: the names it starts or is part of (`weft catalog: no component named "date"; did you mean DatePicker, DateRangePicker?`); when there are none, the components whose purpose, use or notes have it as a word, then names within a few edits (`weft catalog: no component named "modal"; did you mean Dialog, OverlayPanel, Sheet?`); or `…; run weft catalog to list them.` when none is close. It never prints another component in place of the one named. Wrong usage exits with status 2.

## The installer

`curl -fsSL https://weft.base.al/install | sh` installs the latest release. It runs on macOS and Linux, on arm64 and amd64, and needs `curl`, and `sha256sum` or `shasum`.

| Variable | What it does | Default |
| --- | --- | --- |
| `WEFT_VERSION` | The release to install | the latest |
| `WEFT_BIN_DIR` | Where to put `weft` | `$HOME/.local/bin` |
| `WEFT_BASE` | Where the releases are | `https://weft.base.al` |

## weft-build-pages

The page build. It comes with `@weft/vite`; run it from a site's `package.json` script.

```text
weft-build-pages --root <site> --out <dist> [--app-prefix /app [--app-dir <app build>]] [--theme neutral|lisa|weft] [--preload-font <file.woff2>]... [--separate-runtime <section path>]...
```

| Flag | What it does |
| --- | --- |
| `--root` | The site's folder, which holds `pages/`. Required. |
| `--out` | Where the build goes. Required. It must be missing, empty, or a folder an earlier build wrote and nobody changed since; anything else is refused. |
| `--theme` | `neutral` (the default), `lisa` or `weft`. See [Themes](../guides/themes.md). |
| `--app-prefix` | A path prefix, such as `/app`, for a client-rendered app built beside the pages. |
| `--app-dir` | Where that app's Vite build is. Default `<root>/app-dist`. |
| `--preload-font` | A font file, such as `lucide.woff2`, to preload on the pages whose CSS names it, besides the Latin faces of the families they use. Give it once per font. |
| `--separate-runtime` | A section of the site, such as `/components`, whose pages' islands get a runtime of their own, with only the Vue they use, so heavy islands do not grow the shared one. A page is in the longest section it is in or under: `/components/button` is in `/components`. Give it once per section, as a path with no trailing `/`. |

It prints `weft: built N pages into <dist>`. It needs Node 22.18 or later, which runs TypeScript without a build step. Any error is one sentence; the folder at `--out` is left as it was.

## weft-dev

The page dev server. It comes with `@weft/vite`.

```text
weft-dev --root <site> [--port n] [--host h] [--api <url>] [--theme neutral|lisa|weft] [--exit-with-parent]
```

| Flag | What it does |
| --- | --- |
| `--root` | The site's folder, which holds `pages/`. Required. Its `package.json` may be in a folder above. |
| `--port` | The port to listen on. Default `5173`; `0` takes a free one. |
| `--host` | The host to listen on. Default `localhost`. Any other exposes Vite's file serving, the app's folder included, to the network. |
| `--api` | Where `/api` and every path under it go, such as `http://127.0.0.1:8081`. Without it they are pages like any other. |
| `--theme` | `neutral` (the default), `lisa` or `weft`, as for the build. |
| `--exit-with-parent` | Stop, with exit status 0, when standard input ends. `weft dev` passes it with a pipe it holds open. Without it standard input is not read. |

It prints `weft: dev server at <url>` once it listens. Each page is rendered when asked for, as the build renders it, so a saved edit shows on the next request; a dynamic page's `getPaths()` is called then too. A page that exports `runtime = true` is served, as its shell, for every path its pattern matches that no other page gives. An unknown path gets `pages/404.vue` with status 404, or a plain 404 without one. A page that fails to render gets a 500 page naming its file and the error, and reloads when you save the fix; the server keeps running. A change to any file under `--root` reloads the browser. A file a page writes with `useFile` is served once that page has rendered.

It stops on SIGTERM or Ctrl-C. A port in use is the sentence `weft dev: port <n> is in use; stop what is using it or pass --port.`, with exit status 1; a flag it does not know exits with status 2.

## weft-types

Writes a site's `.weft-types/components.d.ts`, which tells TypeScript what each auto-imported component is, as a build does, without building. It comes with `@weft/vite`; `weft create`, `weft build`, `weft dev` and `weft upgrade` run it in an app.

```text
weft-types --root <site>
```

It prints nothing once the file is written. When `@weft/ui` cannot be found from the site, it says so with exit status 1.
