# Weft App Tooling Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A person with only the `weft` binary, Go, Node and pnpm runs `weft create myapp`, `weft dev` and `weft build`, and gets one Go binary serving a Weft site and its API, with no clone of the Weft repository and nothing from npm named `@weft/*`.

**Architecture:** The `weft` binary carries a *bundle*: Weft's three JavaScript packages, the `weft` Go package, the component catalogue and the app template. `weft create` writes the template and unpacks the bundle into the app's own git-ignored `.weft/` folder; the app is a pnpm workspace whose members are `.weft/packages/*`, the same topology as this repository, and its `go.mod` replaces `github.com/flakerimi/weft` with `./.weft/go`. `weft build` runs the existing page build and then `go build`; `weft dev` runs a new page dev server from `@weft/vite` beside the app's Go API and restarts Go when a `.go` file changes.

**Tech Stack:** Go 1.26 standard library only (`embed`, `os/exec`, `text/template`); Vue `3.6.0-rc.16`; Vite `^8.3.8`; `magic-string` (`1.4.5` on 2026-11-23). Run `npm view <pkg> version` again before adding each. (Tailwind was planned for Task 7, which the owner dropped on 2026-11-24.)

**Spec:** `docs/superpowers/specs/2026-11-20-weft-design.md`, phase 4 and *Distribution*. `site-src/content/docs/coming-next.md` lists what this plan delivers: `weft create`, `weft dev`, `weft build`, packages in the binary, auto-imported components. (Tailwind in a new app was dropped with Task 7.)

## Calls made by Claude (they change the spec; Task 8 writes them into it)

1. **The packages are unpacked into the app, at `.weft/`, not into `~/.weft/<version>/`.** A `file:` path into a home folder puts one person's path in `package.json`, so a teammate's clone or a CI run cannot install. `.weft/` is git-ignored and any `weft` command restores it.
2. **The app is a pnpm workspace over `.weft/packages/*`** with `workspace:*` dependencies, instead of `file:` paths. It is how `site-src` and `playground` already consume the packages, so everything the page build relies on (TypeScript source exports, `sideEffects`, CSS resolution) behaves as it is tested here.
3. **The Go package ships in the bundle too**, used through a `replace` in the app's `go.mod`. The repository is private until `0.1.0`, so `go get github.com/flakerimi/weft` fails for anyone else; and the CLI version then decides the Go side as well.
4. **App settings live in `package.json` under `"weft"`** (`version`, `theme`), not in `weft.config.ts`. The Go CLI reads them with `encoding/json` and no Node.
5. **No `--local` flag.** A `weft` built without a bundle reads one from the folder `$WEFT_BUNDLE` names, staged by `scripts/bundle.sh`. One code path serves release and development.
6. **`weft dev` reloads the page on a change; it does not hot-replace modules.** Pages are server-rendered on each request, so a reload is always correct.
7. **`weft upgrade` is in this plan**: with call 1 it is a version rewrite, a re-unpack and an install.
8. **Out of scope, and added to `coming-next.md`:** a client-side router and `layouts/` for app routes (given phase 4c). The template is a page build with islands and a Go API.

## Global Constraints

- **Inherited:** every rule in `AGENTS.md` and the Global Constraints of the foundation, fast-pages and components plans.
- **Go uses the standard library only**, in `cmd/weft` and in the template. No `fsnotify`: watching is polling.
- **`vue` is exactly `3.6.0-rc.16`** in the template. Every component the template has is `<script setup vapor lang="ts">`. No `vaporInteropPlugin`.
- **Every message the CLI prints on failure is one sentence** starting `weft <command>: `, naming the file or tool and saying what to do, with exit code 1 (2 for wrong usage).
- **Prerequisites are checked before anything is written:** `go`, `node` 22.18 or later, `pnpm` 10 or later on `PATH`.
- **Nothing is deleted outside what the command made.** `.weft/` is replaced only when it holds a `VERSION` file the CLI wrote; an app folder is never overwritten.
- **No npm publishing.** The template's `package.json` is `"private": true`.
- **Release platforms stay** darwin and linux, arm64 and amd64. Process handling may be Unix-only behind build tags; `go vet ./...` must still pass.
- **Tests that install from the registry** (they run `pnpm install` in a temporary app) are skipped under `go test -short` and when `pnpm` is not on `PATH` outside CI, as the browser tests skip without Chrome.
- **Docs describe the code as it is**: each task that makes something work moves it out of `coming-next.md` in the same commit.

## Review Focus

1. **A path with a space or a non-ASCII letter.** `weft create` in `/tmp/My Apps/café`, then `dev` and `build`, work as in a plain path. (Tasks 2, 5, 6.)
2. **A fresh clone.** An app with no `.weft/` and no `node_modules/` builds with one `weft build`; an app recorded at another Weft version is refused in a sentence that names both versions and `weft upgrade`. (Task 5.)
3. **Stopping and failing in `weft dev`.** Ctrl-C leaves no child process; a Go compile error is shown and the next save recovers; a taken port is one sentence, not a stack trace. (Task 6.)
4. **Auto-import leaves the author's names alone.** A name the file imports or declares, an app component with a Weft name, a lowercase tag and a tag inside `<pre v-pre>` are not touched. (Task 3.)
5. **`weft create` half-way.** An existing non-empty folder is refused with nothing written; a failed `pnpm install` leaves the app in place and says to run `pnpm install` in it. (Task 2.)

---

## File structure

| Path | Responsibility |
| --- | --- |
| `scripts/bundle.sh` | Stage the bundle into a folder: packages (runtime files only), Go package, catalogue, template |
| `cmd/weft/bundle/` | Where a release stages the bundle for `go:embed`; holds only `README.md` in git |
| `cmd/weft/bundle.go` | Open the bundle (embedded, or `$WEFT_BUNDLE`); unpack it into an app's `.weft/` |
| `cmd/weft/app.go` | Find the app root, read and write `package.json`'s `"weft"`, check prerequisites, `ensure` |
| `cmd/weft/create.go`, `build.go`, `dev.go`, `upgrade.go` | One command each |
| `cmd/weft/proc_unix.go`, `proc_other.go` | Start a child in its own process group and stop the group |
| `templates/app/` | The app template; files ending `.tmpl` are rendered and lose the suffix |
| `packages/vite/src/auto-import.ts` | The `weftAutoImport()` Vite plugin |
| `packages/vite/src/dev.ts`, `bin/weft-dev.mjs` | The page dev server |

The bundle's layout, which Tasks 1, 2 and 8 share:

```
<bundle>/
  weft/                     → copied to <app>/.weft/
    packages/ui/            package.json, src/, dist/weft.css
    packages/themes/        package.json, src/, dist/, tailwind.css
    packages/vite/          package.json, src/, bin/
    go/                     go.mod, serve.go, files.go, manifest.go, version.go
    catalog/components.json
  template/                 a copy of templates/app/
```

---

### Task 1: The bundle, and unpacking it

**Files:**
- Create: `scripts/bundle.sh`, `cmd/weft/bundle.go`, `cmd/weft/bundle/README.md`, `cmd/weft/bundle_test.go`
- Modify: `.gitignore` (add `cmd/weft/bundle/*` and `!cmd/weft/bundle/README.md`)

**Interfaces:**
- Produces:
  - `scripts/bundle.sh <out>`: builds `@weft/themes` and `@weft/ui`, then writes the layout above into `<out>`. `<out>` must be missing, empty, or a bundle this script wrote (it has `.weft-bundle`); anything else is refused. Each package's `package.json` keeps `name`, `version`, `private`, `type`, `sideEffects`, `exports`, `bin`, `dependencies` and nothing else, so no `devDependencies` or `scripts`. No `test/`, `examples/`, `node_modules/`, `*.docs.md` or `*_test.go`. `go/version.go` is written with `Version` set to `$WEFT_VERSION` when given.
  - `func openBundle() (fs.FS, error)`: the folder `$WEFT_BUNDLE` names if set, else the embedded `bundle` when it has `weft/packages`; otherwise the error `this weft was built without its packages; install a release from https://weft.base.al/install, or set WEFT_BUNDLE to a folder made by scripts/bundle.sh`.
  - `func unpack(bundle fs.FS, appDir, version string) error`: replaces `<appDir>/.weft` with the bundle's `weft/` and writes `.weft/VERSION`. It writes to `.weft.new` and renames. It refuses a `.weft` that has no `VERSION` file.
  - `func unpacked(appDir string) string`: the content of `.weft/VERSION`, or `""`.

- [ ] **Step 1: Write the failing tests** in `cmd/weft/bundle_test.go`, over a small `fstest.MapFS` bundle:
  - `TestUnpackWritesTree`: after `unpack`, `.weft/packages/ui/package.json` and `.weft/go/serve.go` exist and `unpacked(dir) == "0.0.6"`.
  - `TestUnpackReplacesOlder`: a second `unpack` with a bundle lacking one file leaves that file gone, and no `.weft.new` remains.
  - `TestUnpackRefusesForeignFolder`: a `.weft/` with a file and no `VERSION` gives an error containing `.weft` and `move it away`, and the file is still there.
  - `TestOpenBundleWithoutPackages`: with `WEFT_BUNDLE` unset and the embedded bundle holding only `README.md`, the error is the sentence above.
  - `TestBundleScript` (skipped under `-short`): runs `scripts/bundle.sh` into `t.TempDir()`; asserts `weft/packages/ui/dist/weft.css` exists, `weft/packages/vite/package.json` has no `devDependencies` key, no path contains `node_modules` or ends in `_test.go`, and `template/package.json.tmpl` exists once Task 2 has landed (assert only when `templates/app` exists).
- [ ] **Step 2: Run** `go test ./cmd/weft -run 'Unpack|Bundle'`. Expected: FAIL, `undefined: unpack`.
- [ ] **Step 3: Implement** `bundle.go` (`//go:embed all:bundle`) and `scripts/bundle.sh`. The script filters each `package.json` with `node -e`.
- [ ] **Step 4: Run** the same command. Expected: PASS. Then break the `VERSION` check on purpose and see `TestUnpackRefusesForeignFolder` fail; restore it.
- [ ] **Step 5: Commit** `CLI: the bundle and unpacking it into an app`.

### Task 2: The template and `weft create`

**Files:**
- Create: `templates/app/**` (below), `cmd/weft/app.go`, `cmd/weft/create.go`, `cmd/weft/create_test.go`
- Modify: `cmd/weft/main.go` (register `create`), `scripts/bundle.sh` (copy the template)

The template, with `{{.Name}}`, `{{.Module}}`, `{{.Version}}`, `{{.Theme}}` and `{{.Tailwind}}`:

| File | Content |
| --- | --- |
| `package.json.tmpl` | `name`, `private`, `type: module`, `"weft": {"version", "theme"}`, dependencies `@weft/ui`, `@weft/themes` as `workspace:*` and `vue` `3.6.0-rc.16`; devDependencies `@weft/vite` `workspace:*`, `@vitejs/plugin-vue`, `vite`, `typescript`, `vue-tsc` at the versions this repository uses |
| `pnpm-workspace.yaml` | `packages: ['.weft/packages/*']` |
| `go.mod.tmpl` | `module {{.Module}}`, `go 1.26`, `require github.com/flakerimi/weft v0.0.0`, `replace github.com/flakerimi/weft => ./.weft/go` |
| `main.go.tmpl` | Listens on `$PORT` or 8080; `weft.Serve(dist, api.Handler())` when `dist != nil`, else `api.Handler()` alone |
| `dist.go.tmpl` | `//go:build !dev`; `//go:embed all:web/dist`; sets `dist` |
| `dist_dev.go.tmpl` | `//go:build dev`; `dist` stays nil, so `weft dev` compiles before any build |
| `api/api.go.tmpl`, `api/api_test.go.tmpl` | `func Handler() http.Handler`; `GET /api/hello` answers `{"message":"Hello from Go"}` |
| `web/pages/index.vue`, `web/pages/404.vue` | Landing surface; uses `Page`, `Section`, `Card` and `Island` with no import lines |
| `web/components/Hello.vue` | An island that fetches `/api/hello` and shows the message in a `Badge` |
| `web/tsconfig.json`, `web/env.d.ts` | Includes `.weft-types/components.d.ts` (Task 3) |
| `gitignore.tmpl` | `.weft/`, `node_modules/`, `web/dist/`, `web/.weft-types/`, the binary's name |
| `AGENTS.md.tmpl`, `CLAUDE.md` , `README.md.tmpl` | The commands, the Vapor and token rules, `.weft/catalog/components.json`, https://weft.base.al/llms.txt |

**Interfaces:**
- Consumes: `openBundle`, `unpack` (Task 1).
- Produces:
  - `weft create <name> [--module <path>] [--theme neutral|lisa|weft] [--tailwind] [--no-install]`. `<name>` matches `^[a-z][a-z0-9-]{0,63}$`; `--module` defaults to the name; `--theme` to `neutral`.
  - `type appConfig struct{ Version, Theme string }`; `func readApp(dir string) (appConfig, error)`; `func writeAppVersion(dir, version string) error` (changes only `weft.version`, keeps the file's other bytes and key order); `func findApp(start string) (string, error)` (the nearest folder at or above `start` whose `package.json` has `"weft"`); `func prerequisites() error`.
  - On success, stdout ends `Created <name>. Next: cd <name> && weft dev\n`.

- [ ] **Step 1: Write the failing tests** in `create_test.go`, running `run("create", …)` in a temp folder with `WEFT_BUNDLE` pointing at a bundle staged once per test binary (`TestMain`, skipped under `-short`), and `--no-install` except where said:
  - `TestCreateWritesApp`: `myapp/go.mod` starts `module myapp`, `myapp/package.json` has `"weft":{"version":"0.0.0-dev","theme":"neutral"}`, no file name ends `.tmpl`, `myapp/.gitignore` exists, `myapp/.weft/VERSION` is `0.0.0-dev`.
  - `TestCreateRefusesExistingFolder`: `myapp/` holding `keep.txt` → exit 1, stderr `weft create: myapp already exists and is not empty; choose another name or move it away.\n`, and the folder holds only `keep.txt`.
  - `TestCreateRefusesBadName`: `My App`, `1app`, `../x` → exit 2 and nothing written.
  - `TestCreateInPathWithSpace` (Review Focus 1): parent `My Apps/café` works the same.
  - `TestCreateInstallFails` (Review Focus 5): with a fake `pnpm` on `PATH` that exits 1, exit 1, stderr ends `run pnpm install in myapp to finish.\n`, and `myapp/package.json` exists.
  - `TestCreatedAppGoTests` (skipped under `-short`): `go vet -tags dev ./...` and `go test -tags dev ./...` pass inside the created app.
  - `TestWriteAppVersionKeepsFile`: only the version's characters differ after the call.
- [ ] **Step 2: Run** `go test ./cmd/weft -run 'Create|WriteApp'`. Expected: FAIL, unknown command `create`.
- [ ] **Step 3: Implement.** Order inside `create`: check the name, check prerequisites, check the folder, open the bundle, write the template into the folder, `unpack`, then `pnpm install` with its output passed through.
- [ ] **Step 4: Run** the tests. Expected: PASS. Make `create` write before the folder check and see `TestCreateRefusesExistingFolder` fail; restore.
- [ ] **Step 5: Commit** `CLI: weft create writes an app from the template`.

### Task 3: Auto-imported components

**Files:**
- Create: `packages/vite/src/auto-import.ts`, `packages/vite/test/auto-import.test.ts`, `packages/vite/test/fixtures/auto-import/`
- Modify: `packages/vite/src/config.ts` (add it unless a plugin named `weft:auto-import` is there), `packages/vite/src/index.ts`, `packages/vite/package.json` (`magic-string`)

**Interfaces:**
- Produces:
  - `weftAutoImport(opts?: { dts?: string | false }): Plugin`, name `weft:auto-import`, `enforce: 'pre'`.
  - `componentNames(indexSource: string): string[]`: the names `@weft/ui`'s index exports as `default as <Name>` from a `.vue` file.
  - `autoImport(code: string, names: Set<string>): { code: string; map: SourceMap } | null`: for an SFC source, adds one `import { … } from '@weft/ui'` line at the top of `<script setup>` for each PascalCase template tag that is in `names` and that the script neither imports nor declares; adds `<script setup vapor lang="ts">` when the file has no script; `null` when nothing is added.
  - With `dts` (default `<root>/.weft-types/components.d.ts`), writes a `declare module 'vue' { interface GlobalComponents { … } }` file, only when its content changed.
- The parse is `@vue/compiler-sfc`'s `parse` and the template AST's element nodes, not a regular expression over the source.

- [ ] **Step 1: Write the failing tests**:
  - `adds the import a tag needs`: `<template><Button /></template>` with an empty `<script setup vapor lang="ts">` gains `import { Button } from '@weft/ui'`.
  - `keeps the author's names` (Review Focus 4), each returning `null` or leaving the name out: the script imports `Button` from `./Button.vue`; the script has `const Card = …`; the tag is `<button>`; the tag is inside `<pre v-pre>`; the name is not a Weft component.
  - `adds a script when there is none`.
  - `one import line for several tags, each name once, sorted`.
  - `componentNames reads the real index`: over `packages/ui/src/index.ts` it has `Button` and `Island`, lacks `useWeft`, and has 76 names (the count on 2026-11-23; the test compares against a count it takes from the file with a second method, not the literal).
  - `a page without imports builds`: `buildSite` over the fixture (a page using `Card` with no import) writes HTML containing the card's markup and its CSS.
  - `the declaration file names every component`.
- [ ] **Step 2: Run** `pnpm --filter @weft/vite test auto-import`. Expected: FAIL, module not found.
- [ ] **Step 3: Implement.**
- [ ] **Step 4: Run** `pnpm --filter @weft/vite test` and `pnpm --filter weft-site build`. Expected: PASS, and the site's output unchanged (`git status` shows no change under tracked files). Remove the "script declares it" check and see its test fail; restore.
- [ ] **Step 5:** Move *Auto-imported components* out of `coming-next.md` into `guides/pages-and-islands.md`; update `skills/weft-components/SKILL.md`. Commit `Vite: Weft components are imported for you`.

### Task 4: The page dev server

**Files:**
- Create: `packages/vite/src/dev.ts`, `packages/vite/bin/weft-dev.mjs`, `packages/vite/test/dev.test.ts`, `packages/vite/test/dev-hydrate.test.ts`
- Modify: `packages/vite/src/build.ts` (export what the document needs: `readStyles`, `pageHTML`, `headTags`, or move them to `packages/vite/src/document.ts` if that reads better), `packages/vite/src/index.ts`, `packages/vite/package.json` (`bin`)

**Interfaces:**
- Consumes: `discoverRoutes`, `expandRoutes`, `renderPage`, `siteConfig`, `startIslands` (existing).
- Produces:
  - `devSite(opts: { root: string; port?: number; host?: string; api?: string; theme?: Theme }): Promise<{ url: string; close(): Promise<void> }>`. Default port 5173, host `localhost`. `api` is where `/api` is proxied, e.g. `http://127.0.0.1:41234`.
  - `weft-dev --root <site> [--port n] [--api <url>] [--theme t]`, which prints `weft: dev server at <url>\n` once it listens.
- Behaviour:
  - A request whose path is a page's is rendered then, through one Vite server, so an edit shows on the next request. Dynamic pages call `getPaths()` per request.
  - The document has the theme and base CSS and the page's CSS in `<style>` tags, the head tags and pre-paint snippets the build writes, `/@vite/client`, and, when the page has islands, one module script that calls `startIslands` with those islands' files.
  - An unknown path renders `pages/404.vue` with status 404, or a plain 404 without one.
  - A change to any file under `root` sends a full reload.
  - A render error answers 500 with the message and the file in the page, and the server keeps running.
  - A port in use rejects with `weft dev: port <n> is in use; stop what is using it or pass --port.`

- [ ] **Step 1: Write the failing tests** over `test/fixtures` sites:
  - `serves a page`: `GET /` is 200 and has the page's text and a `<style>` holding a `--weft-` token.
  - `sees an edit`: after rewriting the page file in a temp copy, the next `GET /` has the new text.
  - `answers 404 with the site's page`, `answers 404 plainly without one`.
  - `proxies /api`: with a local `http.createServer` as `api`, `GET /api/x` returns its body.
  - `a render error is a 500 that names the file, and the next request works`.
  - `says when the port is taken`: the sentence above.
  - `dev-hydrate` (skipped without Chrome, as `examples-hydrate.test.ts` is): in a browser, an island with a counter button responds to a click and the console has no error.
- [ ] **Step 2: Run** `pnpm --filter @weft/vite test dev`. Expected: FAIL, module not found.
- [ ] **Step 3: Implement.** Dev's Vite server is not `createRenderServer`'s: it listens, keeps its WebSocket, and lets Vite optimise `vue` for the browser. Keep `ssr.noExternal: ['@weft/ui']`.
- [ ] **Step 4: Run** `pnpm --filter @weft/vite test` and `pnpm typecheck`. Expected: PASS. If the browser test shows hydration mismatches from unbundled modules, fix them here; do not fall back to rebuilding the site on each change without asking the owner.
- [ ] **Step 5: Commit** `Vite: a dev server that renders pages on request`.

### Task 5: `weft build` and `weft upgrade`

**Files:**
- Create: `cmd/weft/build.go`, `cmd/weft/upgrade.go`, `cmd/weft/build_test.go`
- Modify: `cmd/weft/app.go` (`ensure`), `cmd/weft/main.go`

**Interfaces:**
- Consumes: `findApp`, `readApp`, `writeAppVersion`, `prerequisites`, `openBundle`, `unpack`, `unpacked`.
- Produces:
  - `func ensure(appDir string, stdout, stderr io.Writer) (appConfig, error)`: the gate every app command passes.
    - Recorded version ≠ `weft.Version` → error `this app is on Weft <a> and this weft is <b>; run weft upgrade to move the app, or weft update --version <a> to match it`.
    - `unpacked(appDir)` ≠ recorded version → `unpack`, then `pnpm install`.
    - `node_modules/` missing → `pnpm install`.
  - `weft build [-o <file>]`: `ensure`; `pnpm exec weft-build-pages --root web --out web/dist --theme <theme>`; `go build -o <file> .` (default: the app's `package.json` name). Stdout ends `Built <file>\n`.
  - `weft upgrade [--no-install]`: `writeAppVersion` to `weft.Version`, `unpack`, `pnpm install`; prints `Moved <name> from Weft <a> to <b>\n`, or `<name> is already on Weft <b>\n`.
  - Outside an app: `weft build: no Weft app here; run it in a folder made by weft create.`

- [ ] **Step 1: Write the failing tests** (those that install are skipped under `-short`):
  - `TestBuildMakesBinary`: create, build, start the binary on a free `PORT`; `GET /` is 200 with the template's heading, `GET /api/hello` is `{"message":"Hello from Go"}`, `GET /nope` is 404.
  - `TestBuildFreshClone` (Review Focus 2): move `.weft/`, `node_modules/` and `web/dist/` out of a created app, then `weft build` succeeds.
  - `TestBuildRefusesOtherVersion`: with `weft.version` set to `9.9.9`, exit 1 and the sentence above naming `9.9.9` and `0.0.0-dev`; `.weft/VERSION` unchanged.
  - `TestUpgradeMovesApp`: from that state `weft upgrade --no-install` rewrites the version, and `package.json` otherwise matches byte for byte.
  - `TestBuildOutsideApp`: the sentence above, exit 1.
  - `TestBuildInPathWithSpace` (Review Focus 1).
  - `TestBuildFromSubfolder`: run from `myapp/web/pages`, it builds the app.
- [ ] **Step 2: Run** `go test ./cmd/weft -run 'Build|Upgrade'`. Expected: FAIL, unknown command.
- [ ] **Step 3: Implement.** Children get their output passed through and run with `Dir` set to the app; no shell, so paths need no quoting.
- [ ] **Step 4: Run** the tests. Expected: PASS. If the page build cannot resolve `@weft/ui` from `web/` (its `package.json` is one folder up), fix the resolution in `@weft/vite` with a test there, and keep the layout.
- [ ] **Step 5:** Move `weft build` out of `coming-next.md`. Commit `CLI: weft build and weft upgrade`.

### Task 6: `weft dev`

**Files:**
- Create: `cmd/weft/dev.go`, `cmd/weft/proc_unix.go`, `cmd/weft/proc_other.go`, `cmd/weft/dev_test.go`

**Interfaces:**
- Consumes: `ensure`; `weft-dev` (Task 4).
- Produces:
  - `weft dev [--port n]`: `ensure`; pick a free loopback port for Go; build the app with `go build -tags dev -o .weft/dev/app .` and run it with `PORT` set; run `pnpm exec weft-dev --root web --port <n> --api http://127.0.0.1:<go port> --theme <theme>`; print `weft dev: http://localhost:<n>\n` once both answer.
  - `func goFiles(appDir string) (map[string]time.Time, error)`: every `.go` file's modification time, skipping `.weft`, `.git`, `node_modules` and `web`. Polled every 300 ms; a difference rebuilds and, when the build succeeds, replaces the running process. A failed build prints the compiler's output and leaves the old process running.
  - `func startGroup(cmd *exec.Cmd) error`, `func stopGroup(cmd *exec.Cmd)`: its own process group; `SIGTERM`, then `SIGKILL` after 3 s.
  - SIGINT or SIGTERM stops both children and exits 0. If either child exits by itself, the other is stopped and `weft dev` exits 1 with `weft dev: <the Go server|the page server> stopped; see its output above.`

- [ ] **Step 1: Write the failing tests** (skipped under `-short`), each starting `dev` in a goroutine with a cancellable context and waiting for the URL line:
  - `TestDevServesPagesAndAPI`: `GET <url>/` has the template heading; `GET <url>/api/hello` has the message.
  - `TestDevRestartsGo`: rewrite `api/api.go` so the message is `Hello again`; within 10 s the API answers it.
  - `TestDevSurvivesCompileError` (Review Focus 3): write a syntax error, see `api.go` in stderr, the old message still served; fix it, the new message is served.
  - `TestDevStopsChildren` (Review Focus 3): after cancelling, both ports refuse connections within 5 s.
  - `TestDevPortTaken` (Review Focus 3): with the port held by the test, exit 1 and the one sentence from Task 4, and the Go child is gone.
  - `TestGoFilesSkipsFolders`.
- [ ] **Step 2: Run** `go test ./cmd/weft -run 'Dev|GoFiles'`. Expected: FAIL, unknown command.
- [ ] **Step 3: Implement.** `run`'s signature stays; `devCmd` wraps a function that takes a `context.Context`, which the tests call.
- [ ] **Step 4: Run** the tests three times (`-count=3`), since they are timing tests. Expected: PASS each time.
- [ ] **Step 5:** Move `weft dev` out of `coming-next.md`. Commit `CLI: weft dev runs the page server and the Go API together`.

### Task 7: Tailwind in a new app

> **Dropped by the owner on 2026-11-24**, before it started: Weft has no Tailwind inside, so `weft create` does not set an app up for it; `@weft/themes/tailwind.css` stays for an app that adds Tailwind by hand. Task 8 removed the `--tailwind` stub Task 2 left. The text below is kept as it was planned.

**Files:**
- Modify: `templates/app/package.json.tmpl`, `templates/app/web/pages/index.vue` → `index.vue.tmpl`
- Create: `templates/app/web/vite.config.ts.tmpl`, `templates/app/web/styles/app.css.tmpl` (both written only with `--tailwind`), tests in `cmd/weft/create_test.go` and `build_test.go`

**Interfaces:**
- Consumes: `create`'s template rendering. A template file whose rendered content is empty is not written.
- Produces: with `--tailwind`, devDependencies `tailwindcss` and `@tailwindcss/vite`; `web/vite.config.ts` with the `tailwindcss()` plugin; `web/styles/app.css` with `@import "tailwindcss";` then `@import "@weft/themes/tailwind.css";`; the index page imports the stylesheet and has one element with `class="text-primary"`.

- [ ] **Step 1: Write the failing tests**:
  - `TestCreateTailwindFiles`: the two files exist and `package.json` names both packages; without the flag neither file exists and `package.json` names neither.
  - `TestBuildTailwind` (skipped under `-short`): the built `web/dist/pages/index/index.html` has a rule for `.text-primary` whose value reads `--weft-color-primary`.
- [ ] **Step 2: Run** them. Expected: FAIL.
- [ ] **Step 3: Implement.** If the page build's CSS collection does not carry what Tailwind's plugin emits, fix that in `packages/vite/src/render.ts` with a test in `packages/vite/test/`.
- [ ] **Step 4: Run** `go test ./cmd/weft` and `pnpm --filter @weft/vite test`. Expected: PASS.
- [ ] **Step 5:** Move *Tailwind in a new app* out of `coming-next.md` into `guides/themes.md`. Commit `CLI: weft create --tailwind`.

### Task 8: Releases carry the bundle; docs; the spec

**Files:**
- Modify: `scripts/release.sh`, `.github/workflows/ci.yml`, `AGENTS.md`, `README.md`, `docs/superpowers/specs/2026-11-20-weft-design.md`, `site-src/content/docs/coming-next.md`, `site-src/content/docs/getting-started/install.md`, `site-src/content/docs/reference/cli.md`, `skills/weft-site/SKILL.md`, `skills/weft-go/SKILL.md`, `site-src/agents/llms.ts` if it lists pages by hand
- Create: `site-src/content/docs/getting-started/create-an-app.md`, `cmd/weft/release_test.go`

**Interfaces:**
- Consumes: `scripts/bundle.sh` (Task 1).
- Produces: `scripts/release.sh <version> [RELEASES]` stages the bundle into `cmd/weft/bundle` with `WEFT_VERSION=<version>` before the four `go build`s, and afterwards restores the folder to `README.md` alone by moving the staged bundle to a temporary folder (no `rm`).

- [ ] **Step 1: Write the failing test** `TestReleasedBinaryCreatesAndBuilds` (skipped under `-short`): run `scripts/release.sh 0.0.0-test <tmp>`; unpack the host platform's tarball; with `WEFT_BUNDLE` unset, `weft create demo` then `weft build` in `demo` succeed; `demo/.weft/VERSION` is `0.0.0-test`; the binary is under 15 MB; afterwards `cmd/weft/bundle` holds only `README.md` and `git status --porcelain` is empty.
- [ ] **Step 2: Run** it. Expected: FAIL with the "built without its packages" sentence.
- [ ] **Step 3: Implement** the script change.
- [ ] **Step 4: Run** it. Expected: PASS.
- [ ] **Step 5: Docs.**
  - `create-an-app.md`: install, `weft create`, `weft dev`, edit a page, edit the API, `weft build`, run the binary. Every command and output line is copied from a real run.
  - `reference/cli.md`: `create`, `dev`, `build`, `upgrade` with their flags and their error sentences.
  - `coming-next.md`: remove what now works; add *App routes* (a client-side router and `layouts/`) as phase 4c; keep `weft gen` and the rest.
  - The spec: write the eight calls into *Calls made by Claude*, and correct *Architecture*, *Distribution* and the CLI table to match.
  - `AGENTS.md`: add `templates/`, `scripts/bundle.sh` and the new commands to the layout table.
  - The skills: an app is made with `weft create`; the in-repository way stays for work on Weft itself.
- [ ] **Step 6: CI.** The `go` job runs `go test ./...` without `-short`, so the installing tests run there; confirm the job has `pnpm` and `node` before the Go step (it does today).
- [ ] **Step 7: Run the full suite** from `AGENTS.md`: `pnpm install`, `pnpm test`, `pnpm typecheck`, `scripts/playground.sh build`, `scripts/site.sh build`, `pnpm --filter weft-bench build`, `go vet ./...`, `go test ./...`. Expected: all pass, with no warning left unfixed.
- [ ] **Step 8: Commit** `Releases: the binary carries the packages and the template; docs for weft create, dev and build`.

Cutting and publishing `0.0.6` is the owner's step and is not part of this plan.
