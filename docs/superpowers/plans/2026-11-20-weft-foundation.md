# Weft Foundation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** The first working slice of Weft: the repository and tooling, a `weft` CLI installable with curl from `weft.base.al`, design tokens and the `neutral` and `lisa` themes, the adaptive core (`useWeft`, `WeftProvider`, `useWidth`), the layout components, and a Go binary that embeds a Vapor playground showing them.

**Architecture:** One repository: a Go module at the root (package `weft` plus `cmd/weft`) and a pnpm workspace under `packages/` and `playground/web`. Components are Vue 3.6 `<script setup vapor>` SFCs styled with plain CSS on `--weft-*` custom properties. The Go package serves an embedded Vite build. Release binaries and the install script live in `site/` and are pushed to the Lisa server.

**Tech Stack:** Go 1.26, Vue `3.6.0-rc.16` (Vapor), Vite 8, Vitest 5 with happy-dom, TypeScript 7, pnpm 10.

**Spec:** `docs/superpowers/specs/2026-11-20-weft-design.md` (this plan covers phases 1 and 2, plus the minimal `weft.Serve` needed to run the playground).

## Global Constraints

- Every component is `<script setup vapor>`; no Options API, `getCurrentInstance`, `app.config.globalProperties`, `v-memo`, or VNode render functions.
- Apps and tests mount with `createVaporApp` from `vue`; never install `vaporInteropPlugin`.
- Pin `vue` to exactly `3.6.0-rc.16`. Other versions checked 2026-11-20: `vite ^8.3.8`, `@vitejs/plugin-vue ^6.0.11`, `vitest ^5.1.1`, `happy-dom ^21.0.4`, `typescript ^7.0.2`, `vue-tsc ^3.3.14`. Re-run `npm view <pkg> version` before adding anything else.
- No third-party UI library. Weft has no Tailwind dependency.
- Component names are plain PascalCase (`Stack`, `Card`), no prefix. CSS classes are `weft-<component>`; custom properties `--weft-*`.
- Capability detection only: never test user-agent or device names.
- Nothing is published to npm. Package names `@weft/*` are workspace-internal.
- Go uses only the standard library in package `weft`.
- Commit messages carry no `Co-Authored-By` line (owner's global rule). End them with `Claude-Session: https://claude.ai/code/session_011JKtgodRX7P34o3jKTnB8d`.
- Use `trash`, never `rm`, for deletions, and confirm any deletion with the owner first.

## Review Focus

1. **A touch laptop or a tablet with a mouse switches pointer type while the page is open:** `input` must follow the `(pointer: coarse)` change event without a reload (Task 4).
2. **A theme token missing from a theme:** a component must still render with a visible fallback, not transparent text; `themeToCss` must reject a theme with a missing key (Task 3).
3. **A client route that looks like a file (`/reports/2026.11`) or a missing asset (`/assets/gone.js`):** the route gets `index.html`, the missing asset gets 404, never `index.html` served as JavaScript (Task 6).
4. **Installer on an unsupported platform or with a bad checksum:** it must stop with one clear sentence and a non-zero exit, and never leave a half-written binary on the path (Task 2).
5. **Nested `WeftProvider`s:** an inner provider overriding only `surface` keeps the outer `input`, including the live updates (Task 4).

---

### Task 1: Repository, tooling and the `weft version` CLI

**Files:**
- Create: `go.mod` (`module github.com/flakerimi/weft`, `go 1.26`)
- Create: `cmd/weft/main.go`, `cmd/weft/version.go`, `cmd/weft/main_test.go`
- Create: `version.go` (package `weft`)
- Create: `package.json` (private root, scripts `test`, `build`, `typecheck`), `pnpm-workspace.yaml` (`packages/*`, `playground/web`), `tsconfig.base.json` (strict, `moduleResolution: bundler`, `jsx: preserve`)
- Create: `LICENSE` (MIT, `Copyright (c) 2026 Flakerim Ismani`, the name in `git config user.name`)
- Create: `.github/workflows/ci.yml`
- Modify: `.gitignore` (already ignores `.claude/settings.local.json`, `.claude/scheduled_tasks.lock`, `node_modules/`, `dist/`, `.DS_Store`; add `site/releases/`)

**Interfaces:**
- Produces: `weft.Version` (`var Version = "0.0.0-dev"`, overridden with `-ldflags "-X github.com/flakerimi/weft.Version=<v>"`); CLI `weft version` printing `weft <Version> (<GOOS>/<GOARCH>)`; `run(args []string, stdout, stderr io.Writer) int` in `cmd/weft/main.go` as the testable entry point. Unknown command prints `weft: unknown command "<name>". Run "weft help".` to stderr and returns 2.

- [ ] **Step 1: Write failing tests** in `cmd/weft/main_test.go`: `TestVersion` (run `["version"]` → exit 0, stdout matches `^weft 0\.0\.0-dev \((darwin|linux|windows)/(amd64|arm64)\)\n$`), `TestUnknownCommand` (run `["nope"]` → exit 2, stderr equals the sentence above), `TestHelp` (run `[]` and `["help"]` → exit 0, stdout lists `version`).
- [ ] **Step 2: Run** `go test ./...` → FAIL (no `run`).
- [ ] **Step 3: Implement** `run` with a `map[string]func(...) int` of commands; `main` calls `os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))`.
- [ ] **Step 4: Run** `go test ./...` → PASS.
- [ ] **Step 5: Root JS tooling.** `package.json` with `"private": true`, `"engines": {"node": ">=22", "pnpm": ">=10"}`, scripts `"test": "pnpm -r test"`, `"build": "pnpm -r build"`, `"typecheck": "pnpm -r typecheck"`. Run `pnpm install` → succeeds with an empty workspace.
- [ ] **Step 6: CI** in `.github/workflows/ci.yml`: on push and pull request, jobs `go` (setup-go `1.26.x`, `go vet ./...`, `go test ./...`) and `web` (setup-node 22, pnpm 10, `pnpm install --frozen-lockfile`, `pnpm typecheck`, `pnpm test`, `pnpm build`). Pin action versions to their latest major (check each on GitHub before writing).
- [ ] **Step 7: Commit** `git commit -m "Add repository tooling and weft version"`.

### Task 2: Release script and curl installer

**Files:**
- Create: `scripts/release.sh`
- Create: `site/install` (POSIX `sh`, no bashisms), `site/index.html` (placeholder: name, one-line pitch, the install command)
- Create: `cmd/weft/install_test.go`

**Interfaces:**
- Consumes: `weft.Version` ldflag from Task 1.
- Produces: release layout `site/releases/<version>/weft_<os>_<arch>.tar.gz` (one `weft` binary inside), `site/releases/<version>/SHA256SUMS` (`sha256sum` format), `site/releases/latest` (one line: the version). Platforms: `darwin/arm64`, `darwin/amd64`, `linux/arm64`, `linux/amd64`.
- Produces: installer contract. Env: `WEFT_BASE` (default `https://weft.base.al`), `WEFT_VERSION` (default: contents of `$WEFT_BASE/releases/latest`), `WEFT_BIN_DIR` (default `$HOME/.local/bin`). Exit 0 and print `Installed weft <version> to <dir>/weft`; on failure exit 1 with exactly one line on stderr: `weft install: <sentence>`.

- [ ] **Step 1: Write failing tests** in `cmd/weft/install_test.go`, each serving a fake release tree from `httptest.NewServer(http.FileServer(...))` built in `t.TempDir()` and running `sh ../../site/install` with `WEFT_BASE` set to the test server and `WEFT_BIN_DIR` to a temp dir (skip on Windows):
  - `TestInstallLatest`: `latest` = `0.0.1`, valid tarball with an executable script `weft` → exit 0, `<bin>/weft` exists and is executable, stdout `Installed weft 0.0.1 to <bin>/weft`.
  - `TestInstallBadChecksum`: `SHA256SUMS` wrong → exit 1, stderr `weft install: checksum mismatch for weft_<os>_<arch>.tar.gz`, `<bin>/weft` absent.
  - `TestInstallUnsupportedPlatform`: env `WEFT_TEST_UNAME_S=Plan9` (installer reads `uname -s`/`uname -m` unless these test overrides are set) → exit 1, stderr `weft install: no build for plan9/<arch>`.
  - `TestInstallKeepsOldBinaryOnFailure`: existing `<bin>/weft` with content `old`, bad checksum → file still contains `old`.
- [ ] **Step 2: Run** `go test ./cmd/weft -run Install` → FAIL (no `site/install`).
- [ ] **Step 3: Implement `site/install`.** Map `uname -s` (`Darwin`→`darwin`, `Linux`→`linux`) and `uname -m` (`arm64`/`aarch64`→`arm64`, `x86_64`→`amd64`); download with `curl -fsSL` into `mktemp -d`; verify with `shasum -a 256` or `sha256sum`, whichever exists; extract; `mv` into place only after verification (atomic replace); `chmod 755`. Print a hint if `WEFT_BIN_DIR` is not on `PATH`.
- [ ] **Step 4: Run** the install tests → PASS.
- [ ] **Step 5: Implement `scripts/release.sh <version>`**: refuses without a version matching `^[0-9]+\.[0-9]+\.[0-9]+(-[a-z0-9.]+)?$`; runs `go test ./...`; builds the four platforms with `CGO_ENABLED=0` and the ldflag; writes tarballs, `SHA256SUMS` and `latest` under `site/releases/`. It does not push or tag.
- [ ] **Step 6: Verify:** `scripts/release.sh 0.0.0-test` then `WEFT_BASE=file://$PWD/site WEFT_BIN_DIR=$(mktemp -d) sh site/install` → prints `Installed weft 0.0.0-test …`, and running that binary with `version` prints `weft 0.0.0-test (darwin/arm64)`. (`curl` accepts `file://`.) Then remove `site/releases/0.0.0-test` with `trash`.
- [ ] **Step 7: Commit** `git commit -m "Add release script and curl installer"`.

### Task 3: Tokens and themes (`packages/themes`)

**Files:**
- Create: `packages/themes/package.json` (`@weft/themes`, private, `type: module`, exports `.` → `src/index.ts`, `./neutral.css`, `./lisa.css`, `./tailwind.css`; scripts `build`, `test`, `typecheck`, `sync:lisa`)
- Create: `packages/themes/src/tokens.ts`, `src/to-css.ts`, `src/neutral.ts`, `src/lisa.ts`, `src/index.ts`, `scripts/build.ts`, `scripts/sync-lisa.sh`
- Create: `packages/themes/src/lisa/tokens.json` (copied from `/Volumes/External/Lisa/docs/contract/tokens.json` by `sync-lisa.sh`)
- Create: `packages/themes/tailwind.css` (hand-written `@theme` mapping)
- Test: `packages/themes/test/to-css.test.ts`, `test/lisa.test.ts`, `test/neutral.test.ts`

**Interfaces:**
- Produces, in `src/tokens.ts`:

```ts
export const colorKeys = ['bg','surface','surface-raised','border','text','text-muted',
  'primary','primary-fg','primary-soft','danger','warning','success','info','focus'] as const
export type ColorKey = typeof colorKeys[number]
export type ColorSet = Record<ColorKey, string>
export interface Theme {
  name: string
  light: ColorSet
  dark: ColorSet
  font: { sans: string; mono: string; weightBody: number; weightHeading: number; weightTitle: number }
  radius: { control: number; card: number; panel: number }          // px
  space: { 1: number; 2: number; 3: number; 4: number; 6: number; 8: number } // px
  motion: { control: number; layout: number; dialog: number }        // ms
  ease: { standard: [number,number,number,number]; emphasized: [number,number,number,number] }
}
```

- Produces: `themeToCss(theme: Theme): string`, emitting `:root, [data-weft-scheme="light"] { --weft-color-<key>: …; --weft-radius-control: 10px; --weft-space-4: 16px; --weft-duration-control: 150ms; --weft-ease-standard: cubic-bezier(…); --weft-font-sans: …; --weft-weight-body: … }`, then `[data-weft-scheme="dark"] { --weft-color-<key>: … }` and `@media (prefers-color-scheme: dark) { :root:not([data-weft-scheme="light"]) { … } }`, then the density block: `[data-weft-input="pointer"] { --weft-control-height: 36px; --weft-hit-min: 24px }`, `[data-weft-input="touch"] { --weft-control-height: 44px; --weft-hit-min: 44px }`, `[data-weft-surface="landing"] { --weft-text-base: 18px; --weft-page-max: 1200px; --weft-section-gap: var(--weft-space-8) }`, `[data-weft-surface="app"] { --weft-text-base: 14px; --weft-page-max: none; --weft-section-gap: var(--weft-space-6) }`. Throws `Error('theme "<name>" is missing <light|dark>.<key>')` for any missing colour.
- Produces: `neutral: Theme`, `lisa(tokens: LisaTokens, opts?: { accent?: string }): Theme`, and built files `dist/neutral.css`, `dist/lisa.css`.

Pinned values. `neutral` light: bg `#FAFAFA`, surface `#FFFFFF`, surface-raised `#FFFFFF`, border `#E4E4E7`, text `#18181B`, text-muted `#71717A`, primary `#2563EB`, primary-fg `#FFFFFF`, primary-soft `#DBEAFE`, danger `#DC2626`, warning `#D97706`, success `#16A34A`, info `#0284C7`, focus `#2563EB`. Dark: bg `#09090B`, surface `#18181B`, surface-raised `#27272A`, border `#27272A`, text `#FAFAFA`, text-muted `#A1A1AA`, primary `#3B82F6`, primary-fg `#FFFFFF`, primary-soft `#172554`, danger `#EF4444`, warning `#F59E0B`, success `#22C55E`, info `#38BDF8`, focus `#60A5FA`. Font `Inter, system-ui, sans-serif` / `ui-monospace, monospace`, weights 400/600/700; radius 8/12/16; space 4/8/12/16/24/32; motion 150/220/260; ease standard `[0.2,0,0,1]`, emphasized `[0.3,0,0,1]`.

`lisa` mapping from `tokens.json` (`value` of each entry): light bg `color.ink.paper`, surface `color.ink.surface`, surface-raised `color.ink.surface`, border `color.ink.line-200`, text `color.ink.ink-900`, text-muted `color.ink.ink-500`; dark bg `color.dark.base`, surface `color.dark.elevated`, surface-raised `color.dark.card`, border `color.dark.line`, text `color.dark.text-primary`, text-muted `color.dark.text-secondary`; both schemes primary `color.accent.accent-<accent>-500` (default accent `olive`), primary-soft `accent-<accent>-300`, primary-fg `color.ink.warm-white`, focus = primary, danger `color.semantic.error`, warning `.warning`, success `.success`, info `.info`; font `font.ui.value`/`font.mono.value`, weights `type.body-weight`/`type.heading-weight`/`type.title-weight`; radius `metric.radius-control`/`radius-card`/`radius-panel`; space `metric.space-*`; motion `motion.control`/`layout`/`dialog`; ease `curve.standard`/`curve.emphasized`.

- [ ] **Step 1: Write failing tests.**
  - `to-css.test.ts`: `themeToCss(neutral)` contains `--weft-color-bg: #FAFAFA;`, `--weft-radius-control: 8px;`, `--weft-ease-standard: cubic-bezier(0.2, 0, 0, 1);`, `[data-weft-input="touch"]` with `--weft-control-height: 44px`; a theme with `light.primary` deleted throws `theme "neutral" is missing light.primary`.
  - `lisa.test.ts` (reads the vendored `src/lisa/tokens.json`): light bg `#F1F5F9`, dark bg `#0F172A`, primary `#6B7D2E`, font sans starts with `Rubik`, radius control 10, motion control 150; `lisa(tokens, { accent: 'violet' }).light.primary` equals `color.accent.accent-violet-500.value`; unknown accent throws `unknown Lisa accent "teal-ish"`; `themeToCss(lisa(tokens))` contains neither `#FFFFFF` nor `#000000` (case-insensitive) — Lisa's `noWhiteAnywhere` rule.
  - `neutral.test.ts`: every `colorKeys` entry present in both schemes and a valid `#RRGGBB`.
- [ ] **Step 2: Run** `pnpm --filter @weft/themes test` → FAIL.
- [ ] **Step 3: Implement** `tokens.ts`, `to-css.ts`, `neutral.ts`, `lisa.ts` per the interfaces and pinned values; `sync-lisa.sh` copies the contract file and refuses if the source path is missing; `scripts/build.ts` writes `dist/neutral.css` and `dist/lisa.css` (run with `node --experimental-strip-types`, available in Node 22.6+).
- [ ] **Step 4: Write `tailwind.css`**: `@theme inline { --color-bg: var(--weft-color-bg); … one line per colour key …; --radius-control: var(--weft-radius-control); --font-sans: var(--weft-font-sans) }`.
- [ ] **Step 5: Run** tests → PASS; `pnpm --filter @weft/themes build` → both CSS files written.
- [ ] **Step 6: Commit** `git commit -m "Add design tokens and the neutral and lisa themes"`.

### Task 4: Adaptive core (`packages/ui`)

**Files:**
- Create: `packages/ui/package.json` (`@weft/ui`, private, `type: module`, deps `vue: 3.6.0-rc.16`, `@weft/themes: workspace:*`; devDeps vite, `@vitejs/plugin-vue`, vitest, happy-dom, typescript, vue-tsc; exports `.` → `src/index.ts`, `./weft.css` → `dist/weft.css`)
- Create: `packages/ui/vite.config.ts` (plugin-vue; Vitest `environment: 'happy-dom'`)
- Create: `packages/ui/src/adaptive/detect.ts`, `src/adaptive/context.ts`, `src/adaptive/useWidth.ts`, `src/adaptive/WeftProvider.vue`, `src/index.ts`
- Create: `packages/ui/test/mount.ts` (test helper)
- Test: `packages/ui/test/detect.test.ts`, `test/provider.test.ts`, `test/useWidth.test.ts`

**Interfaces:**
- Produces:

```ts
export type Input = 'pointer' | 'touch'
export type Surface = 'landing' | 'app'
export type Width = 'narrow' | 'regular' | 'wide'
export function detectInput(mq?: (q: string) => MediaQueryList): Ref<Input> // live
export interface WeftContext { input: Readonly<Ref<Input>>; surface: Readonly<Ref<Surface>> }
export function useWeft(): WeftContext   // outside any provider: live detected input, surface 'app'
export function widthOf(px: number): Width // <640 narrow, <1024 regular, else wide
export function useWidth(el: Ref<HTMLElement | null>): Readonly<Ref<Width>> // ResizeObserver on the element
```

- `WeftProvider` props: `input?: Input`, `surface?: Surface`, `as?: string` (default `'div'`). Renders `<as class="weft-provider" data-weft-input data-weft-surface>` with a default slot, and provides a context where unset props inherit from the parent context (live).
- Test helper: `mount(component, props?, slots?) → { el: HTMLElement, unmount(): void }` using `createVaporApp(component, props).mount(div)`.
- `input` is `'touch'` exactly when `(pointer: coarse)` matches.

- [ ] **Step 1: Write failing tests** with a fake `matchMedia` (an object whose `matches` can be flipped and whose `change` listeners can be fired):
  - `detect.test.ts`: coarse → `'touch'`; fine → `'pointer'`; flip from fine to coarse and fire `change` → ref becomes `'touch'` (Review Focus 1). `widthOf(639)` narrow, `widthOf(640)` regular, `widthOf(1023)` regular, `widthOf(1024)` wide.
  - `provider.test.ts`: `<WeftProvider surface="landing">` renders `data-weft-surface="landing"` and `data-weft-input` equal to the detected input; a child component calling `useWeft()` sees `surface === 'landing'`; nested `<WeftProvider surface="app">` inside `<WeftProvider input="touch">` gives the child `input 'touch'`, `surface 'app'`; with the outer provider unset, flipping the media query updates the inner child's `input` (Review Focus 5); `useWeft()` outside any provider returns surface `'app'`.
  - `useWidth.test.ts`: stub `ResizeObserver`, report width 500 → `'narrow'`, then 1100 → `'wide'`; unmount disconnects the observer.
- [ ] **Step 2: Run** `pnpm --filter @weft/ui test` → FAIL.
- [ ] **Step 3: Implement** the files above. Context is an `InjectionKey<WeftContext>`; the provider uses `computed` over its props and the parent context. If `createVaporApp` mounting of `<script setup vapor>` fails under Vitest, stop and report the exact error; do not fall back to `createApp`.
- [ ] **Step 4: Run** tests → PASS; `pnpm --filter @weft/ui typecheck` (`vue-tsc --noEmit`) → no errors.
- [ ] **Step 5: Commit** `git commit -m "Add adaptive core: useWeft, WeftProvider, useWidth"`.

### Task 5: Layout components

**Files:**
- Create: `packages/ui/src/layout/Stack.vue`, `Grid.vue`, `Page.vue`, `Section.vue`, `Card.vue`, `Divider.vue`
- Create: `packages/ui/src/styles/base.css` (box-sizing, body font and colours from tokens, focus ring), `src/styles/layout.css`, `src/styles/weft.css` (imports `base.css`, `layout.css`)
- Modify: `packages/ui/src/index.ts` (export the six components and the adaptive API)
- Modify: `packages/ui/package.json` (script `build` copies/bundles `src/styles/weft.css` to `dist/weft.css` with Vite's CSS build)
- Test: `packages/ui/test/layout.test.ts`

**Interfaces:**
- Consumes: `useWeft`, `useWidth`, `widthOf` (Task 4); `--weft-*` tokens (Task 3).
- Produces (props, all optional unless stated):
  - `Stack`: `direction: 'column'|'row'` = `'column'`, `gap: 0|1|2|3|4|6|8` = `4`, `align: 'start'|'center'|'end'|'stretch'` = `'stretch'`, `justify: 'start'|'center'|'end'|'between'` = `'start'`, `wrap: boolean` = `false`, `as: string` = `'div'`. Renders `class="weft-stack"` with `data-direction`, `data-gap`, `data-align`, `data-justify`, `data-wrap` (only when true).
  - `Grid`: `cols: number | { narrow?: number; regular?: number; wide?: number }` = `{ narrow: 1, regular: 2, wide: 3 }`, `gap` as Stack. Uses `useWidth` on its own root and sets `style="--weft-grid-cols: N"` for the current width; `data-width` shows which.
  - `Page`: `title?: string`, `description?: string`; slots `default`, `header`, `actions`. Renders `<main class="weft-page">` with an `<h1>` when `title` is set; max width from `--weft-page-max`.
  - `Section`: `title?`, `description?`; slots `default`, `actions`. Renders `<section class="weft-section">` with an `<h2>`.
  - `Card`: `padding: 0|2|3|4|6` = `4`, `as` = `'div'`; slots `default`, `header`, `footer`, `actions`. Renders `class="weft-card"`; header region only when `header` slot or `actions` slot is given.
  - `Divider`: `orientation: 'horizontal'|'vertical'` = `'horizontal'`; renders `<hr class="weft-divider" aria-orientation>` for horizontal, `<div role="separator" aria-orientation="vertical">` for vertical.

- [ ] **Step 1: Write failing tests** in `layout.test.ts`: Stack default attributes (`data-direction="column"`, `data-gap="4"`); `<Stack direction="row" :gap="2" wrap>` attributes; Grid with stubbed width 500 → `--weft-grid-cols: 1`, `data-width="narrow"`; width 1100 → `3`; `:cols="4"` → `4` at every width; Page with `title="Customers"` renders `h1` text `Customers` and the `actions` slot content; Section `h2`; Card without header/actions slots renders no `.weft-card__header`; Divider vertical has `role="separator"` and `aria-orientation="vertical"`.
- [ ] **Step 2: Run** → FAIL.
- [ ] **Step 3: Implement** components and CSS. CSS uses only `var(--weft-*)`, each with a fallback for colour and size (e.g. `color: var(--weft-color-text, CanvasText)`) so a missing token never yields invisible text (Review Focus 2). `layout.css` maps `data-gap` values to `gap: var(--weft-space-N)`; Grid uses `grid-template-columns: repeat(var(--weft-grid-cols), minmax(0, 1fr))`.
- [ ] **Step 4: Run** tests and typecheck → PASS; `pnpm --filter @weft/ui build` → `dist/weft.css` exists.
- [ ] **Step 5: Commit** `git commit -m "Add layout components: Stack, Grid, Page, Section, Card, Divider"`.

### Task 6: `weft.Serve`

**Files:**
- Create: `serve.go`, `serve_test.go` (package `weft`, module root)

**Interfaces:**
- Produces: `func Serve(dist fs.FS, api http.Handler) http.Handler`.
  - Paths under `/api/` go to `api` (unchanged path). `api == nil` → `/api/*` answers 404 JSON `{"error":{"code":"not_found","message":"no API"}}`.
  - Existing files are served from `dist`. Files under `/assets/` get `Cache-Control: public, max-age=31536000, immutable`; everything else `Cache-Control: no-cache`.
  - A path that is not a file: if it starts with `/assets/` → 404; otherwise → `index.html` with status 200 and `Content-Type: text/html; charset=utf-8`.
  - `dist` without `index.html` → `Serve` panics at construction with `weft.Serve: dist has no index.html`.

- [ ] **Step 1: Write failing tests** with `fstest.MapFS{"index.html": …, "assets/app-abc.js": …, "favicon.svg": …}` and `httptest`: `/api/ping` reaches a stub handler; `/assets/app-abc.js` → 200 with the immutable header and a JavaScript content type; `/assets/gone.js` → 404 (Review Focus 3); `/customers/42` → 200 index HTML; `/reports/2026.11` → 200 index HTML (Review Focus 3); `/favicon.svg` → 200 with `no-cache`; `/` → index; nil api → JSON 404; missing `index.html` panics with the exact message.
- [ ] **Step 2: Run** `go test ./...` → FAIL.
- [ ] **Step 3: Implement** with `http.FileServerFS` for files and `fs.Stat` to decide file vs route; strip the leading `/` and clean the path before `fs.Stat`.
- [ ] **Step 4: Run** → PASS.
- [ ] **Step 5: Commit** `git commit -m "Add weft.Serve for embedded frontends"`.

### Task 7: Playground (Go binary embedding a Vapor app)

**Files:**
- Create: `playground/web/package.json` (private, deps `vue: 3.6.0-rc.16`, `@weft/ui: workspace:*`, `@weft/themes: workspace:*`; scripts `dev`, `build`, `typecheck`), `playground/web/vite.config.ts` (plugin-vue, `server.proxy['/api'] = 'http://localhost:8080'`, `build.outDir = 'dist'`), `playground/web/index.html`, `playground/web/src/main.ts`, `src/App.vue`, `src/Controls.vue`
- Create: `playground/main.go` (`//go:embed all:web/dist`), `playground/main_test.go`
- Create: `scripts/playground.sh` (`pnpm --filter playground build && go run ./playground`)

**Interfaces:**
- Consumes: `Serve` (Task 6); `WeftProvider`, layout components (Tasks 4–5); `@weft/themes/neutral.css`, `lisa.css` (Task 3); `@weft/ui/weft.css`.
- Produces: `/api/info` returning `{"version": weft.Version}`; a page that shows every layout component with controls to switch theme (`neutral`/`lisa`, by swapping which theme stylesheet is active), scheme (`light`/`dark`/system via `data-weft-scheme` on `<html>`), `input` override (auto/pointer/touch) and `surface` (landing/app). Listens on `:8080` or `$PORT`.
- `main.ts` mounts with `createVaporApp(App).mount('#app')`.

- [ ] **Step 1: Write failing test** `playground/main_test.go`: `TestPlaygroundServes` builds the handler via an exported `handler() http.Handler`, GETs `/` → 200 and body contains `<div id="app">`; GETs `/api/info` → JSON with `version`. Skip with a clear message if `web/dist/index.html` is absent (`run scripts/playground.sh build first`).
- [ ] **Step 2: Build the web app and run** `pnpm --filter playground build && go test ./playground` → FAIL (no `handler`).
- [ ] **Step 3: Implement** the playground web app and `main.go`.
- [ ] **Step 4: Run** the build and the test → PASS. Add the playground build before `go test` in the CI `go` job (needs Node in that job).
- [ ] **Step 5: See it work.** Run `scripts/playground.sh`, open `http://localhost:8080` in the browser, switch each control, and check: both themes load, dark scheme changes colours, `touch` raises control height, `landing` widens the type scale, Grid changes column count when the window narrows. Take a screenshot of each theme for the README (`docs/assets/playground-neutral.png`, `docs/assets/playground-lisa.png`).
- [ ] **Step 6: Commit** `git commit -m "Add the playground: Go binary embedding the Vapor app"`.

### Task 8: README and the `weft.base.al` site

**Files:**
- Create: `README.md`
- Modify: `site/index.html` (real landing for the project: pitch, install command, link to the GitHub repo, status "pre-alpha")
- Create: `scripts/publish-site.sh`

**Interfaces:**
- Consumes: install command and release layout (Task 2), screenshots (Task 7).
- `scripts/publish-site.sh` runs `scripts/release.sh "$1"` then `lisa server push --name weft site`, then fetches `https://weft.base.al/releases/latest` and fails unless it equals the version.

- [ ] **Step 1: Write README.md**: wordmark heading, one-sentence pitch, status badge text "pre-alpha", the install line `curl -fsSL https://weft.base.al/install | sh`, a 15-line example (a Go `main.go` with `weft.Serve` and a Vue page using `Page`, `Grid`, `Card`), the adaptive table from the spec (input × surface), the two screenshots, a short roadmap pointing at the spec's phases, licence. Plain names only, no claims about features not built yet beyond the roadmap.
- [ ] **Step 2: Check** every link in README and `site/index.html` resolves (`curl -sI`), except `weft.base.al` paths published in Step 4.
- [ ] **Step 3: Before pushing**, run `lisa server apps` and confirm no application is named `weft` (a push under an existing name replaces it), and list `site/` to confirm it holds only `index.html`, `install` and `releases/`. Tell the owner the push publishes `https://weft.base.al`.
- [ ] **Step 4: Publish** `scripts/publish-site.sh 0.0.1`. Then `lisa server apps` shows `weft`; `curl -fsSL https://weft.base.al/install | WEFT_BIN_DIR=$(mktemp -d) sh` installs `0.0.1` and that binary prints `weft 0.0.1 (darwin/arm64)`.
- [ ] **Step 5: Commit and tag** `git commit -m "Add README and the weft.base.al site"`, `git tag v0.0.1`, `git push --follow-tags`.

---

Plans that follow, each written when the one before lands: (2) headless layer and primitives, (3) Go-to-Vue contract and auth, (4) `@weft/vite`, `weft create`/`dev`/`build`/`upgrade`, (5) app blocks and charts, (6) agent kit (catalogue, skills, lint, `check`, `mcp`), (7) landing blocks and `0.1.0`.
