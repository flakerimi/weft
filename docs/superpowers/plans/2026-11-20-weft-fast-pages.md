# Weft Fast Pages Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Pages rendered to HTML at build time with zero JavaScript by default, interactive islands that hydrate on their own, and a Go server that sends pre-compressed bytes from memory with ETags, Early Hints, prefetch and view transitions — then a benchmark proving it beats Astro.

**Architecture:** A new workspace package `@weft/vite` owns build-time work in Node: file-based page discovery, a Vite client build of island chunks, server rendering of each page with Vue's server renderer, the island runtime, critical-CSS inlining, and pre-compression into a `manifest.json`. Package `weft` in Go reads that manifest from the embedded `dist` and serves pages, assets and the SPA fallback with no per-request work.

**Tech Stack:** Vue `3.6.0-rc.16` (`createVaporSSRApp`, `vue/server-renderer`), Vite 8, Node 22.6+ (`node:zlib` brotli and gzip), Go 1.26 standard library, Astro `7.4.13` and Lighthouse `13.6.0` in `bench/` only.

**Spec:** `docs/superpowers/specs/2026-11-20-weft-design.md`, section "Speed: faster than Astro…" and phase 2b.

## Global Constraints

- Everything from the foundation plan's Global Constraints still holds (Vapor-only components, no `vaporInteropPlugin`, Vue pinned to `3.6.0-rc.16`, no npm publishing, no `Co-Authored-By`, `trash` not `rm`, confirm deletions).
- Server rendering loads `vue` and `vue/server-renderer` with Node's own `import` (resolved from the app with `createRequire`), and only app and Weft source through Vite's SSR loader. Loading Vue through `ssrLoadModule` fails (`module is not defined`); found in the 2026-11-20 spike.
- A page with no island ships **no `<script>` element that loads JavaScript**. JSON script types (`speculationrules`, island props) are allowed.
- No per-request compression, rendering or hashing in Go: everything comes from the build manifest.
- Go stays standard library only. Compression happens in Node at build (`zlib.brotliCompressSync` at quality 11, `gzipSync` level 9).
- Versions checked 2026-11-20; re-check with `npm view` before adding.

## Review Focus

1. **A page whose island fails to load its chunk** (offline, 404): the server-rendered HTML must stay visible and usable as static content; no blank page, no thrown error that blocks other islands (Task 2).
2. **`Accept-Encoding` variations** (`br;q=0`, `gzip, deflate`, missing, `*`): the right variant or identity is sent, always with `Vary: Accept-Encoding`, never brotli to a client that refused it (Task 5).
3. **`If-None-Match` with a list or a weak ETag** (`W/"…"`, `"a", "b"`): matching any listed tag yields 304 with no body (Task 5).
4. **Routes that collide**: `pages/about.vue` and `pages/about/index.vue` both present, or `[slug].vue` producing a path that a static page already owns — the build must fail with one sentence naming both files (Task 3).
5. **Hydration mismatch** (an island renders differently in the browser, e.g. reads `Date.now()`): it must recover to client-rendered content without breaking the page (Task 2).

---

### Task 1: Server rendering and `<Island>` (packages/vite + packages/ui)

**Files:**
- Create: `packages/vite/package.json` (`@weft/vite`, private, `type: module`, deps `vite ^8.3.8`, `@vitejs/plugin-vue ^6.0.11`, `vue 3.6.0-rc.16`; devDeps vitest, happy-dom, typescript `~6.0.3`; exports `.` → `src/index.ts`, `./island-runtime` → `src/island-runtime.ts`)
- Create: `packages/vite/src/render.ts`, `packages/vite/test/fixtures/site/` (a tiny site: `pages/index.vue` static, `pages/counter.vue` with one island, `components/Counter.vue`)
- Create: `packages/ui/src/islands/Island.vue`; Modify: `packages/ui/src/index.ts`
- Test: `packages/vite/test/render.test.ts`, `packages/ui/test/island.test.ts`

**Interfaces:**
- Produces: `<Island when="load" | "idle" | "visible" | "media" :media?="string" :component="Comp" :props="object" />`. On the server it renders `<weft-island data-id="<chunk id>" data-when="…" data-media?="…"><script type="application/json">{props}</script><!--component SSR--></weft-island>`. In the browser it renders nothing itself (the runtime hydrates). Island ids come from a build-time registry, `registerIsland(id: string, loader: () => Promise<Component>)`, written by Task 2's plugin.
- Produces: `renderPage(opts: { root: string; page: string; props?: Record<string, unknown> }): Promise<{ html: string; islands: string[] }>` in `render.ts`, using a Vite dev server in middleware mode for app and Weft source, and native `import` for `vue` and `vue/server-renderer`.

- [ ] **Step 1: Failing tests.** `render.test.ts`: rendering `pages/index.vue` gives HTML containing the page heading and `islands: []`; rendering `pages/counter.vue` gives exactly one `<weft-island data-id="Counter" data-when="visible">` whose JSON script is `{"start":2}` and whose content contains `n is 2`. `island.test.ts`: `Island` under `createVaporApp` (client, no SSR) renders no element.
- [ ] **Step 2: Run** `pnpm --filter @weft/vite test` and `pnpm --filter @weft/ui test` → FAIL.
- [ ] **Step 3: Implement** `Island.vue` (server branch via `import.meta.env.SSR`) and `render.ts`.
- [ ] **Step 4: Run** → PASS. Commit `Add build-time page rendering and Island`.

### Task 2: Island runtime and hydration

**Files:**
- Create: `packages/vite/src/island-runtime.ts`, `packages/vite/src/plugin.ts` (Vite plugin `weft()` that emits one chunk per island component and the registry module `virtual:weft/islands`)
- Test: `packages/vite/test/island-runtime.test.ts`

**Interfaces:**
- Produces: `startIslands(registry: Record<string, () => Promise<Component>>, opts?: { observe?: typeof IntersectionObserver; idle?: (cb: () => void) => void; matchMedia?: (q: string) => MediaQueryList }): void`. For each `weft-island`: `load` hydrates at once, `idle` on `requestIdleCallback` (fallback `setTimeout 200`), `visible` on first intersection, `media` when the query matches. Hydration: `createVaporSSRApp(component, props).mount(islandElement)`. A rejected loader or a thrown hydration logs one `console.warn("weft: island <id> did not start: <reason>")` and leaves the server HTML in place; other islands continue.
- Size budget: `island-runtime` minified + brotli ≤ 1.5 KB, excluding Vue.

- [ ] **Step 1: Failing tests** (happy-dom, server HTML injected by hand from Task 1's output): `load` island hydrates and a click on its button changes `n is 2` → `n is 3` **on the same button node** (hydrated, not re-rendered); `visible` island does nothing until the fake observer fires; `media` island waits for the fake query; a loader that rejects leaves the HTML, warns once, and a second island on the page still hydrates (Review Focus 1); an island whose client render differs from the server HTML ends showing the client content and the page keeps working (Review Focus 5); runtime budget checked by building it with Vite and brotli-compressing the output.
- [ ] **Step 2: Run** → FAIL. **Step 3: Implement.** **Step 4: Run** → PASS. Commit `Add the island runtime`.

### Task 3: File-based pages and the route map

**Files:**
- Create: `packages/vite/src/routes.ts`; Test: `packages/vite/test/routes.test.ts` with fixtures under `test/fixtures/routes-*/`

**Interfaces:**
- Produces: `discoverRoutes(pagesDir: string): Promise<Route[]>` where `Route = { path: string; file: string; dynamic: boolean }`. Rules: `index.vue` → `/`, `about.vue` → `/about`, `blog/index.vue` → `/blog`, `blog/[slug].vue` → `/blog/:slug` (dynamic; the page exports `getPaths(): Promise<{ slug: string }[]>` used at build), `404.vue` → the not-found page (path `/404`). Output order: static before dynamic, then alphabetical.
- A collision throws `Error('pages "<a>" and "<b>" both make <path>')` (Review Focus 4), including a dynamic page whose `getPaths()` yields a path a static page owns.
- `routes.json` shape written by the build: `{ "pages": [{ "path": "/about", "html": "pages/about/index.html", "islands": ["Counter"] }], "notFound": "pages/404/index.html" | null, "app": { "prefix": "/app", "index": "app/index.html" } | null }`.

- [ ] Steps: failing tests for each rule and both collision kinds → FAIL → implement → PASS → commit `Add file-based pages and the route map`.

### Task 4: `weft build` for pages: render, inline CSS, compress, manifest

**Files:**
- Create: `packages/vite/src/build.ts` (`buildSite(opts: { root: string; outDir: string; appPrefix?: string }): Promise<Manifest>`), `packages/vite/bin/weft-build-pages.mjs` (Node entry the Go CLI calls)
- Test: `packages/vite/test/build.test.ts`

**Interfaces:**
- Produces in `outDir`: `pages/<path>/index.html` per route (dynamic routes expanded), `assets/*` (hashed client chunks for islands and the runtime), and for every text file (`.html .css .js .json .svg .txt .xml`) siblings `.br` and `.gz`; `routes.json` (Task 3 shape); `manifest.json`:

```json
{ "files": { "/pages/about/index.html": { "etag": "\"<first 16 hex of sha256>\"", "type": "text/html; charset=utf-8", "br": true, "gz": true, "immutable": false } },
  "hints": { "/about": ["</assets/inter.woff2>; rel=preload; as=font; crossorigin"] } }
```

- Each page's HTML: theme CSS and `weft.css` inlined in one `<style>` when their total is ≤ 14 KB, otherwise linked; `<script type="speculationrules">{"prefetch":[{"where":{"href_matches":"/*"},"eagerness":"moderate"}]}</script>`; `@view-transition { navigation: auto; }` in the inlined CSS; the island runtime `<script type="module">` **only** when the page has islands.

- [ ] **Step 1: Failing tests** on the Task 1 fixture site: `pages/index/index.html` has no `<script` whose type is missing or `module` (zero JS); `pages/counter/index.html` has exactly one module script; every `.html` has `.br` and `.gz` siblings that decompress to the original bytes; `manifest.json` ETags equal the hash of the file bytes; inlined `<style>` contains `--weft-color-bg` and `@view-transition`.
- [ ] Steps 2–4 → FAIL → implement → PASS. Commit `Add the page build: render, inline CSS, compress, manifest`.

### Task 5: `weft.Serve` from the manifest

**Files:**
- Modify: `serve.go`; Create: `manifest.go`; Test: `serve_test.go`, `manifest_test.go`

**Interfaces:**
- `Serve(dist fs.FS, api http.Handler) http.Handler` keeps its signature. When `dist` has `manifest.json`, it is read once at construction; without it, behaviour is exactly the foundation's (SPA mode), so the playground keeps working.
- Routing with a manifest: `/api/*` → api; exact page path (with or without trailing slash) → its `pages/…/index.html`; path under `app.prefix` → `app.index`; a file in `files` → that file; anything else → `notFound` with status 404, or plain 404.
- Encoding: parse `Accept-Encoding` with q-values; prefer `br`, then `gzip`, else identity; `q=0` excludes; `*` allows both. Always `Vary: Accept-Encoding`; `Content-Encoding` set when compressed; `Content-Length` from the stored bytes.
- Caching: `ETag` from the manifest (a compressed variant uses the same tag with suffix `-br` / `-gz` inside the quotes); `If-None-Match` matching any listed tag, weak or strong, → 304 with no body (Review Focus 3); `Cache-Control` immutable for `immutable: true`, else `no-cache`.
- Early Hints: for a page with `hints`, send `103` with those `Link` headers before the 200 (`w.Header().Add("Link", …)`, `w.WriteHeader(http.StatusEarlyHints)`), and the same `Link` headers on the 200.
- All files read into memory at construction; no `fs` access per request.

- [ ] **Step 1: Failing tests** with a `fstest.MapFS` manifest site: br chosen for `br, gzip`; gzip for `gzip;q=1, br;q=0`; identity with no header and for `identity`; `*` gets br; `Vary` on every response; 304 for exact, weak and listed ETags with empty body; `/about` and `/about/` both serve the page; `/app/x/y` serves the app index; `/nope` gets the 404 page with status 404; Early Hints observed with an `httptest` server and `httptrace.ClientTrace.Got1xxResponse`; a benchmark `BenchmarkServePage` reports allocations (target ≤ 3 allocs/op).
- [ ] Steps 2–4 → FAIL → implement → PASS. Commit `Serve pages from the build manifest`.

### Task 6: Dogfood — `weft.base.al` built with Weft

**Files:**
- Create: `site-src/` (a Weft site: `pages/index.vue` using `Page`, `Section`, `Card`, `Stack`; a copy-to-clipboard install button as the one island; the `lisa` theme), `site-src/main.go` (serves with `weft.Serve`)
- Modify: `scripts/publish-site.sh` (builds `site-src` into `site/` before staging; installer and releases unchanged)

- [ ] **Step 1:** Build it; the home page HTML has one island and a total transfer (HTML + CSS + JS, brotli) under 12 KB; without the island it would be under 6 KB. Assert both in a Go test over the build output.
- [ ] **Step 2:** Check it in Chrome headless at 1280 and 500 px, light and dark; screenshots to `docs/assets/site-*.png`.
- [ ] **Step 3:** Publish with `scripts/publish-site.sh 0.0.3` (new pages replace the placeholder; releases kept). Verify the live page, `curl --compressed -I` shows `content-encoding: br`, and `weft update` from 0.0.2 reaches 0.0.3.
- [ ] Commit `Build weft.base.al with Weft`.

### Task 7: Benchmark against Astro

**Files:**
- Create: `bench/astro/` (Astro `7.4.13` site), `bench/weft/` (Weft site), both rendering the same landing: hero, three feature cards, a pricing table of three cards, one interactive counter island, one 1600 px image; `bench/run.sh`; `bench/load/main.go` (a small Go load generator: N connections, fixed duration, reports requests/second and p50/p99); `bench/README.md` with the latest results table.

**Interfaces:**
- `bench/run.sh` builds both, serves Astro with its own production server (`astro preview`) and Weft with its binary, then records per site: JS bytes and total transfer (brotli) of the landing, TTFB (median of 50), Lighthouse 13.6.0 performance score and LCP (median of 5, mobile preset), requests/second on one core (`GOMAXPROCS=1` for Weft; Astro static output served by its preview server and, for fairness, also by a plain static file server).
- Output: `bench/results/<date>.json` and the table in `bench/README.md`.

- [ ] **Step 1:** Write both sites and the runner; run it.
- [ ] **Step 2:** Record results honestly. Rows where Weft does not win are listed as findings for the next plan, not hidden.
- [ ] Commit `Add the Astro benchmark`.

---

Not in this plan (next plans): `<Image>` pipeline, content collections, actions, env, dev overlay, `weft deploy`, primitives.
