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
| [`tools/`](tools) | `images.mjs` (the image), `measure.mjs` (all measurements), `table.mjs` (this table), and two side tools run by hand: `lcp-experiment.mjs` (the side experiment below) and `html-breakdown.mjs` (what the Weft page's HTML is made of). Lighthouse and sharp, with their own lockfile. |
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

| Site | Server | HTML+CSS (br) | JS files (br) | Inline JS (raw) | Total (br) | Requests | TTFB, ms | Lighthouse | LCP, ms | Req/s (Go servers at GOMAXPROCS=1) | p50 / p99, ms |
|---|---|--:|--:|--:|--:|--:|--:|--:|--:|--:|--:|
| Weft | weft binary (weft.Serve), GOMAXPROCS=1 | 2.9 KB | 20.5 KB | 0 B | 81.8 KB | 4 | 0.16 [0.13–0.17] | 100 [100–100] | 1200 [1200–1201] | 124,503 [121,830–124,957] | 0.50 / 1.06 |
| Astro, Vue island | astro preview | 3.5 KB | 26.5 KB | 5238 B | 88.5 KB | 5 | 0.35 [0.33–0.40] | 100 [100–100] | 1357 [1355–1357] | 10,875 [10,790–10,936] | 5.81 / 6.81 |
| Astro, Vue island | Go static + gzip, GOMAXPROCS=1 | 3.5 KB | 26.5 KB | 5238 B | 88.5 KB | 5 | 0.26 [0.24–0.29] | 100 [100–100] | 1353 [1351–1502] | 8,206 [8,182–8,210] | 7.96 / 12.54 |
| Astro, no framework island | astro preview | 1.7 KB | 0 B | 237 B | 60.2 KB | 2 | 0.32 [0.31–0.34] | 100 [100–100] | 752 [751–752] | 12,465 [12,375–12,542] | 5.06 / 6.25 |
| Astro, no framework island | Go static + gzip, GOMAXPROCS=1 | 1.7 KB | 0 B | 237 B | 60.2 KB | 2 | 0.23 [0.21–0.26] | 100 [100–100] | 751 [751–751] | 11,309 [11,204–11,416] | 5.77 / 8.98 |

Bytes: KB is 1024 B. Text is counted as brotli quality 11 (what the Weft build stores), whatever the server sends; the image as it is. TTFB: median of 50 requests on new connections, [p25–p75]. Lighthouse and LCP: median of 5 mobile runs, [min–max]. Load: 64 connections, 2s warm-up, 10s measured, median of 3 rounds [min–max]; p50 and p99 are the medians of the rounds' own. No errors unless noted. The Go servers (Weft, Go static) run at GOMAXPROCS=1; `astro preview` is not held to one core (Node, with gzip on libuv's thread pool).

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
  in all against 88.5 KB, four requests against five, a simulated LCP of 1200 ms against 1353–1357 ms. The
  exception is outside the results table: under real (DevTools) throttling in the side experiment below, Weft's
  LCP is about 50 ms behind Astro's Vue island. One confound: the two sides do not run the same Vue. Weft ships
  Vue 3.6.0-rc.16 with Vapor; Astro's Vue integration renders with Vue's virtual DOM, here Vue 3.5.46. Part of the
  JavaScript difference is Vapor against the virtual DOM, not Weft against Astro.
- **Lighthouse scores 100 for all five**, every run. The score does not separate these pages; LCP does. The LCP
  column is Lighthouse's simulation (Lantern), not a load on a throttled device; see the side experiment.
- **TTFB** is a fraction of a millisecond everywhere on loopback; Weft's 0.16 ms median against 0.23–0.35 ms is
  real (the quartile ranges do not overlap) but too small to matter on a real network.
- **Requests per second** is mostly a server result, not a framework one: Weft answers from memory with the brotli
  the build stored, while `astro preview` and the Go static server gzip every response as they send it. That is
  the point of `weft.Serve`, but an Astro site behind a server that serves pre-compressed files from memory
  might narrow this gap (not measured).
- **What went over the wire**: Weft sent brotli; both Astro servers sent gzip, which is why Astro's wire bytes are
  a little above its brotli column.

## Where Weft loses

Every row but the last is against **Astro, no framework island**, Astro's own default, served by either server.
Weft does not lose to Astro with a Vue island in any column of the results table; the last row is the one place it
does, in the side experiment.

| Measure | Weft | Astro (no framework island, but for the last row) | Why, as best we can tell | Next step |
|---|--:|--:|---|---|
| JavaScript | 20.5 KB br in 2 files | 237 B inline | The island is a Vue component, so the page loads Weft's client chunk (`client-*.js`, 60,570 B raw): the Vapor runtime, reactivity and the island runtime. The counter's own chunk is 538 B. Astro's default island ships no framework at all. | Find out which part of those 60,570 B a Vapor island really needs (a bundle report of the client chunk) and cut the runtime to it; add a JS budget test for the benchmark page as `site-src` has. Consider an island form with no framework for trivial interactivity. |
| LCP (Lighthouse, mobile, simulated, Lantern) | 1200 ms | 751 ms | Hypothesis, supported by the side experiment: Lantern charges the LCP for every script the page loads, whenever it loads it. Moving Weft's module script to the end of the body (1201 ms) or importing it at idle (1206 ms) changes nothing; removing it gives 751 ms, the same as Astro with no framework. Astro's Vue island, already loaded at idle by an inline loader, is at 1353–1357 ms. So the simulated loss is the cost of loading about 20 KB of JavaScript at all, not of where the script sits. Under real (DevTools) throttling the gap shrinks to about 90 ms (1633 against 1546 and 1543 ms). | Not an inline loader: the experiment shows it does not help. Test first, then build: (1) the same page with `fetchpriority="high"` and `loading="eager"` on the LCP image, which the `<Image>` plan would write, under both throttling methods; (2) a smaller client chunk (see the JavaScript row). Read Lighthouse's LCP breakdown for each. |
| HTML and CSS | 2.9 KB br | 1.7 KB br | The page inlined 7,989 B of CSS (about 1.8 KB of the 2.9 KB brotli): the whole neutral theme, light and dark tokens, and `weft.css` with the rules of every layout component, used or not. Since then each component ships its own stylesheet and a page inlines only the theme, `base.css` and the stylesheets of the components it renders, minified (2026-11-21, measured on the built page, not in a new run, which needs network installs): 8,329 B of CSS, 1,739 B brotli on its own, and 2,926 B brotli for the page. That is no smaller than on the run's day, because this page renders five of the eight layout components (Page, Section, Stack, Grid, Card) and the theme has grown (3,276 B of the CSS); without per-page CSS, the `weft.css` of today, with the basic and form components, would have made it 35,320 B of CSS and 6,494 B brotli. The Astro page carries only its own hand-written stylesheet, so part of this is the price of a UI kit, not of the page build. Vue's hydration markers are another 80 markers, 622 B raw, 53 B of the brotli: 46 `<!--[-->`/`<!--]-->`, 18 `<!---->` and 16 Vapor `<!--{-->`/`<!--}-->` (without the Vapor kind, 64 markers and 494 B). All 80 are outside the island, where nothing hydrates. How each number was measured is in `weft_html_breakdown` in the results JSON. | Inline only the tokens a page uses (the CSS of the components it renders is done); drop hydration markers outside islands, since nothing hydrates there. |
| Total transfer | 81.8 KB br | 60.2 KB br | The JavaScript above; the image is the same 59,862 B in both. | As for JavaScript. |
| Requests | 4 | 2 | The client chunk and the island's chunk. | As for JavaScript. |
| LCP against **Astro, Vue island** (side experiment, DevTools throttling, both behind `static/`) | 1633 ms [1616–1637] | 1582 ms [1565–1598] | Hypothesis: on a really throttled network the JavaScript shares bandwidth with the hero image. Lighthouse's LCP breakdown puts the difference in the image's load duration (median 970 ms for Weft against 931 ms for Astro's Vue page and 896–899 ms with no JS; the other subparts are within about 20 ms). Weft's module script is fetched at once, at high priority; Astro fetches its Vue runtime later, from its idle loader. Weft's idle-import variant was slower still (1679 ms), so later loading alone does not explain it. | The same tests as the LCP row above, read under DevTools throttling: a high-priority image first, then a smaller client chunk. |

## Side experiment: where the island script loads, and LCP

A labelled side experiment, run after the main run on the same machine, with no change to Weft's code
([`tools/lcp-experiment.mjs`](tools/lcp-experiment.mjs); every run is in the results JSON under
`side_experiments.lcp_script_placement`). Hand-edited copies of the built Weft page and the two Astro pages are all
served by `static/` (gzip on the fly, so here Weft's files go out as gzip, not the brotli `weft.Serve` sends). Each
page gets five Lighthouse mobile runs, interleaved, with the default simulated throttling (Lantern) and with
DevTools throttling, where the page really loads on a throttled network and CPU. LCP, median [min–max]:

| Page | Simulated (Lantern) | DevTools throttling |
|---|--:|--:|
| Weft as built: module script in the head | 1201 [1201–1202] | 1633 [1616–1637] |
| Weft, the same script at the end of the body | 1201 [1126–1201] | 1630 [1612–1634] |
| Weft, no script element; an inline module imports it in `requestIdleCallback` | 1206 [1204–1206] | 1679 [1666–1696] |
| Weft, no script at all (the island never starts) | 751 [750–751] | 1546 [1532–1561] |
| Astro, Vue island | 1354 [1353–1356] | 1582 [1565–1598] |
| Astro, no framework island | 751 [751–751] | 1543 [1521–1561] |

Every run scored 100. What it shows:

- In the simulation, the LCP depends on whether the page loads JavaScript, not on where or when the script is
  loaded.
- Under real throttling the cost of the JavaScript is about 90 ms, not 450 ms, and Astro's Vue page is ahead of
  Weft's by about 50 ms.
