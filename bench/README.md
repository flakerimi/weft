# Weft against Astro

The same landing page built with Weft and with Astro, served, and measured on one machine: bytes, time to first
byte, Lighthouse and requests per second on one core.

The landing: a hero, a 1600 px image, three feature cards, a pricing table of three cards and one interactive counter
island that hydrates when the browser is idle. The text is one file, [`content.json`](content.json), that both sites
import.

| Folder | What it is |
|---|---|
| [`weft/`](weft) | The Weft site: `pages/index.vue` with `@weft/ui`, the counter as a Vapor island, built by `weft-build-pages` and embedded in a Go binary that serves it with `weft.Serve` (like `site-src`). Part of the pnpm workspace. |
| [`astro/`](astro) | The Astro site: `/` with the counter as a Vue island (`@astrojs/vue`, `client:idle`), and `/no-framework/` with the counter as a plain `<script>` in an `.astro` component. Not part of the workspace: its own `package.json` and `package-lock.json`. |
| [`static/`](static) | A plain Go static file server: `net/http.FileServer`, text gzipped on the fly. |
| [`load/`](load) | The load generator and TTFB timer, a Go `main` (standard library only), run as its own process. |
| [`tools/`](tools) | `images.mjs` (the image), `measure.mjs` (all measurements), `table.mjs` (this table). Lighthouse and sharp, with their own lockfile. |
| [`image/hero.jpg`](image/hero.jpg) | The one image source both sites start from. |
| [`results/`](results) | One JSON per run, with every individual measurement. |

Run it from the repository root with `bench/run.sh`. It needs Node 22.23.4 or later (Astro's minimum), pnpm, Go and
Google Chrome (`CHROME_PATH`, the macOS default if unset). It installs, builds both sites, starts the three servers,
runs `tools/measure.mjs`, writes `results/<date>.json` and rewrites the table below.

## Fairness rules

- **Same content, same image.** Both sites render `content.json`, and both images come from `image/hero.jpg`.
- **Images.** Astro uses its own `<Image width={1600}>`. Weft has no `<Image>` yet, so `tools/images.mjs` makes the
  WebP the Weft site serves (`weft/public/hero.webp`) once, the way Astro's sharp service does: same width, same
  sharp version, sharp's default WebP quality. The two files come out byte for byte the same (59,862 B). The `<img>`
  attributes on the Weft page are the ones Astro's `<Image>` writes (`width`, `height`, `loading="lazy"`,
  `decoding="async"`).
- **The island.** It is a Vue component in both: the same single-file component, `<script setup vapor>` for Weft and
  without `vapor` for Astro, whose Vue integration renders with Vue's virtual DOM. Both hydrate it when idle. Astro's
  own default, a plain `<script>` with no framework, is measured too, as "Astro, no framework island".
- **Production builds.** `weft-build-pages` and `astro build`, each with its defaults. Each site styles itself the way
  its users would: the Weft page with `@weft/ui` and its theme, the Astro page with a hand-written stylesheet of the
  same look.
- **Servers.** Weft runs as its own binary. Astro's static output is measured behind `astro preview` (its own
  production preview server, gzip on the fly) and behind `static/` (a plain Go file server, gzip on the fly), so
  Astro is not judged only by its preview server. The Go servers run with `GOMAXPROCS=1`. `astro preview` is Node:
  one JavaScript thread, but its gzip (Vite's preview server uses `@polka/compression`, zlib streams) runs on libuv's
  thread pool, so it can use more than one core; nothing here limits that. Nothing pins any server to a core either:
  macOS has no CPU affinity.
- **Load.** `load/` runs in its own process with all cores: 64 keep-alive connections, `Accept-Encoding: br, gzip`,
  2 s warm-up, 10 s measured, the landing's HTML only. Every request that starts and ends inside the 10 s counts.
  Three rounds per target, the targets interleaved; the median and its range.
- **TTFB.** 50 requests one after another, each on a new connection, after 5 untimed ones, timed to the final
  response's headers (past Weft's 103 Early Hints; the first byte of the 103 is in the JSON as
  `first_byte_median_ms`).
- **Lighthouse.** Lighthouse 13.6.0, the default mobile preset (simulated throttling), headless Chrome, a fresh
  profile each run, five runs per target interleaved so drift hits all targets alike. Median and range.
- **Bytes.** The resources a Lighthouse run loaded (status 200), each fetched again. Text is counted as brotli
  quality 11, which is what Weft stores, for every site, whatever its server sends: that is Astro's best case, as if
  its host pre-compressed. "Inline JS" is JavaScript inside the HTML (`<script>` without `src`, not a JSON type),
  counted raw; it is already in the HTML's bytes. What each server really sent is in the second table.
- **Weft is not tuned for this.** The Weft site uses only what the page build does by default.

## Results

<!-- results:start -->
Run of 2026-11-20 ([`results/2026-11-20.json`](results/2026-11-20.json)).

Machine: Apple M4, 10 cores (4 performance), 32 GB, macOS 27.2.1 (26B131). Node v26.0.0, go1.26.2, Chrome 157.0.8103.79 headless. Weft at `8964daa` with Vue 3.6.0-rc.16; Astro 7.4.13, @astrojs/vue 7.1.6, Vue 3.5.46; Lighthouse 13.6.0.

| Site | Server | HTML+CSS (br) | JS files (br) | Inline JS (raw) | Total (br) | Requests | TTFB, ms | Lighthouse | LCP, ms | Req/s, one core | p50 / p99, ms |
|---|---|--:|--:|--:|--:|--:|--:|--:|--:|--:|--:|
| Weft | weft binary (weft.Serve), GOMAXPROCS=1 | 2.9 KB | 20.5 KB | 0 B | 81.8 KB | 4 | 0.16 [0.13–0.17] | 100 [100–100] | 1200 [1200–1201] | 124,503 [121,830–124,957] | 0.50 / 1.06 |
| Astro, Vue island | astro preview | 3.5 KB | 26.5 KB | 5238 B | 88.5 KB | 5 | 0.35 [0.33–0.40] | 100 [100–100] | 1357 [1355–1357] | 10,875 [10,790–10,936] | 5.81 / 6.81 |
| Astro, Vue island | Go static + gzip, GOMAXPROCS=1 | 3.5 KB | 26.5 KB | 5238 B | 88.5 KB | 5 | 0.26 [0.24–0.29] | 100 [100–100] | 1353 [1351–1502] | 8,206 [8,182–8,210] | 7.96 / 12.54 |
| Astro, no framework island | astro preview | 1.7 KB | 0 B | 237 B | 60.2 KB | 2 | 0.32 [0.31–0.34] | 100 [100–100] | 752 [751–752] | 12,465 [12,375–12,542] | 5.06 / 6.25 |
| Astro, no framework island | Go static + gzip, GOMAXPROCS=1 | 1.7 KB | 0 B | 237 B | 60.2 KB | 2 | 0.23 [0.21–0.26] | 100 [100–100] | 751 [751–751] | 11,309 [11,204–11,416] | 5.77 / 8.98 |

Bytes: KB is 1024 B. Text is counted as brotli quality 11 (what the Weft build stores), whatever the server sends; the image as it is. TTFB: median of 50 requests on new connections, [p25–p75]. Lighthouse and LCP: median of 5 mobile runs, [min–max]. Load: 64 connections, 2s warm-up, 10s measured, median of 3 rounds [min–max]; p50 and p99 are the medians of the rounds' own. No errors unless noted.

What each server really sent (Accept-Encoding: br, gzip):

| Site | Server | Page encoding | JS on the wire | Total on the wire | LCP element |
|---|---|---|--:|--:|---|
| Weft | weft binary (weft.Serve), GOMAXPROCS=1 | br | 20.5 KB | 81.8 KB | `div.bench > main.weft-page > div.weft-page__body > img.bench-image` |
| Astro, Vue island | astro preview | gzip | 29.4 KB | 92.1 KB | `body > main.page > img.image` |
| Astro, Vue island | Go static + gzip, GOMAXPROCS=1 | gzip | 29.2 KB | 91.8 KB | `body > main.page > img.image` |
| Astro, no framework island | astro preview | gzip | 0 B | 60.7 KB | `body > main.page > img.image` |
| Astro, no framework island | Go static + gzip, GOMAXPROCS=1 | gzip | 0 B | 60.7 KB | `body > main.page > img.image` |
<!-- results:end -->

## Reading the results

The numbers in the prose below are from the run in the table (2026-11-20).

- **Against Astro with a Vue island**, the like-for-like comparison, Weft is ahead in every column it can be ahead
  in: 2.9 KB of HTML and CSS against 3.5 KB, 20.5 KB of JavaScript files against 26.5 KB plus 5.2 KB inline, 81.8 KB
  in all against 88.5 KB, four requests against five, an LCP of 1200 ms against 1353–1357 ms.
- **Lighthouse scores 100 for all five**, every run. The score does not separate these pages; LCP does.
- **TTFB** is a fraction of a millisecond everywhere on loopback; Weft's 0.16 ms median against 0.23–0.35 ms is
  real (the quartile ranges do not overlap) but too small to matter on a real network.
- **Requests per second** is mostly a server result, not a framework one: Weft answers from memory with the brotli
  the build stored, while `astro preview` and the Go static server gzip every response as they send it. That is
  the point of `weft.Serve`, but an Astro site behind a server that serves pre-compressed files from memory would
  close much of this gap; it is not measured here.
- **What went over the wire**: Weft sent brotli; both Astro servers sent gzip, which is why Astro's wire bytes are
  a little above its brotli column.

## Where Weft loses

Every row is against **Astro, no framework island**, Astro's own default, served by either server. Weft does not
lose to Astro with a Vue island in any column.

| Measure | Weft | Astro, no framework island | Why, as best we can tell | Next step |
|---|--:|--:|---|---|
| JavaScript | 20.5 KB br in 2 files | 237 B inline | The island is a Vue component, so the page loads Weft's client chunk (`client-*.js`, 60,570 B raw): the Vapor runtime, reactivity and the island runtime. The counter's own chunk is 538 B. Astro's default island ships no framework at all. | Find out which part of those 60,570 B a Vapor island really needs (a bundle report of the client chunk) and cut the runtime to it; add a JS budget test for the benchmark page as `site-src` has. Consider an island form with no framework for trivial interactivity. |
| LCP (Lighthouse, mobile) | 1200 ms | 751 ms | The LCP element is the hero image in every case. Weft's page puts the client chunk in the head as a module script, so it is fetched at high priority while the lazy image waits, and Lighthouse's simulated mobile network and CPU charge it to the LCP. Astro with its Vue island (26.5 KB of JS files, loaded by 5.2 KB of inline loader) lands at 1353–1357 ms, so the cost follows the JavaScript on the load path, not Weft itself. | For `when="idle"` and `"visible"` islands, do not load the runtime from the head: a small inline loader that imports it when the first island's trigger fires. Then re-run this benchmark. The `<Image>` plan should also offer an eager, high-priority image for the one above the fold. |
| HTML and CSS | 2.9 KB br | 1.7 KB br | The page inlines 7,989 B of CSS (about 1.8 KB of the 2.9 KB brotli): the whole neutral theme, light and dark tokens, and `weft.css` with the rules of every layout component, used or not. The Astro page carries only its own hand-written stylesheet, so part of this is the price of a UI kit, not of the page build. Vue's hydration markers in the static parts are another 622 B raw, about 50 B brotli. | Inline only the CSS of the components a page renders, and only the tokens it uses; drop hydration markers outside islands, since nothing hydrates there. |
| Total transfer | 81.8 KB br | 60.2 KB br | The JavaScript above; the image is the same 59,862 B in both. | As for JavaScript. |
| Requests | 4 | 2 | The client chunk and the island's chunk. | As for JavaScript. |
