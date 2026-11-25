# Weft against Astro, Nuxt, Next.js and SvelteKit

The same landing page built with Weft, Astro, Nuxt, Next.js and SvelteKit, served, and measured on one machine:
bytes, time to first byte, Lighthouse and requests per second.

The landing: a hero, a 1600 px image, three feature cards, a pricing table of three cards and one interactive
counter. The text is one file, [`content.json`](content.json), that every site imports.

| Folder | What it is |
|---|---|
| [`weft/`](weft) | The Weft site: `pages/index.vue` with `@weft/ui`, the counter as a Vapor island, built by `weft-build-pages` and embedded in a Go binary that serves it with `weft.Serve` (like `site-src`). Part of the pnpm workspace. |
| [`astro/`](astro) | The Astro site: `/` with the counter as a Vue island (`@astrojs/vue`, `client:idle`), and `/no-framework/` with the counter as a plain `<script>` in an `.astro` component. |
| [`nuxt/`](nuxt) | The Nuxt site: `app/pages/index.vue`, the counter a Vue component. Built twice: `nuxt generate` (static) and `nuxt build` (its Node server, Nitro, with the page prerendered). |
| [`next/`](next) | The Next.js site: the App Router, the page a server component and the counter its one client component. Built twice: a static export and the build `next start` serves, the page prerendered in both. |
| [`sveltekit/`](sveltekit) | The SvelteKit site: `src/routes/+page.svelte`, the counter a Svelte component, the page prerendered. Built twice: `adapter-static` and `adapter-node`. |
| [`shared/landing.css`](shared/landing.css) | The stylesheet of the Nuxt, Next.js and SvelteKit sites: the Astro site's rules, as they are there. |
| [`static/`](static) | A plain Go file server with two modes: `net/http.FileServer` with text gzipped on the fly, and `-precompressed`, which reads a folder into memory once and sends the brotli or gzip stored beside each file. |
| [`load/`](load) | The load generator and TTFB timer, a Go `main` (standard library only), run as its own process. |
| [`tools/`](tools) | `images.mjs` (the image), `precompress.mjs` (a copy of a static build with brotli and gzip beside each text file), `measure.mjs` (all measurements), `table.mjs` (this table), and two side tools run by hand: `lcp-experiment.mjs` (the side experiment below) and `html-breakdown.mjs` (what the Weft page's HTML is made of). Lighthouse and sharp, with their own lockfile. |
| [`image/hero.jpg`](image/hero.jpg) | The one image source every site starts from. |
| [`results/`](results) | One JSON per run, with every individual measurement. |

The Astro, Nuxt, Next.js and SvelteKit sites are not part of the pnpm workspace: each has its own `package.json` and
`package-lock.json`.

Run it from the repository root with `bench/run.sh`. It needs Node 22.23.4 or later (Astro's minimum), pnpm, Go and
Google Chrome (`CHROME_PATH`, the macOS default if unset). It installs, builds every site, starts the thirteen
servers, runs `tools/measure.mjs`, writes `results/<date>.json` and rewrites the table below; it takes about half an
hour. `BENCH_TRIAL=<file> bench/run.sh --runs 1 --duration 2s --load-runs 1` is a quick trial that writes its results
to `<file>` and leaves `results/` and this page alone.

## Fairness rules

- **Same content, same image.** Every site renders `content.json`, and every image comes from `image/hero.jpg`.
- **Images.** Astro uses its own `<Image width={1600}>`. Weft has no `<Image>` yet, so `tools/images.mjs` makes the
  WebP the Weft site serves (`weft/public/hero.webp`) once, the way Astro's sharp service does: same width, same
  sharp version, sharp's default WebP quality. The two files come out byte for byte the same (59,862 B). The Nuxt,
  Next.js and SvelteKit sites serve a copy of that file in a plain `<img>`, not through `<NuxtImg>`, `next/image` or
  `@sveltejs/enhanced-img`: a static export cannot run Next's optimizer, and one file for all keeps the image out of
  the comparison. The `<img>` attributes everywhere are the ones Astro's `<Image>` writes (`width`, `height`,
  `loading="lazy"`, `decoding="async"`).
- **The counter.** In Weft and Astro it is an island: the same Vue single-file component, `<script setup vapor>` for
  Weft and without `vapor` for Astro, whose Vue integration renders with Vue's virtual DOM; both hydrate it when
  idle. Astro's own default, a plain `<script>` with no framework, is measured too, as "Astro, no framework island".
  Nuxt and SvelteKit have no islands by default: they hydrate the whole page, with the counter a component in it.
  In Next.js the page is a server component and the counter a client component, which is how the App Router keeps
  the rest of the page out of the browser's JavaScript.
- **Production builds, each with its defaults.** `weft-build-pages`, `astro build`, `nuxt generate` and `nuxt build`,
  `next build`, `vite build` with SvelteKit's adapters. No site sets an option to shrink or speed up its output.
  Each site styles itself the way its users would: the Weft page with `@weft/ui` and its theme, the others with a
  hand-written stylesheet of the same look.
- **Servers.** Every framework is measured behind three:
  1. **the server it ships with**: Weft's binary; `astro preview` (Astro's production preview server, gzip on the
     fly); Nuxt's Node server (Nitro) and `next start`, each serving the page it prerendered at build; and
     SvelteKit's `adapter-node` server, which stores brotli and gzip at build as Weft does;
  2. **`static/`, gzip on the fly**: a plain Go file server, so no framework is judged only by its own server;
  3. **`static/ -precompressed`**: the same Go server answering from memory with brotli (quality 11) and gzip made
     once by `tools/precompress.mjs`. These are the serving terms `weft.Serve` gives a Weft build, so this row
     compares the pages and not the servers.

  The Go servers run with `GOMAXPROCS=1`. The frameworks' own servers are Node: one JavaScript thread, but
  compression and I/O run on libuv's thread pool, so they can use more than one core; nothing here limits that.
  Nothing pins any server to a core either: macOS has no CPU affinity.
- **Load.** `load/` runs in its own process with all cores: 64 keep-alive connections, `Accept-Encoding: br, gzip`,
  2 s warm-up, 10 s measured, the landing's HTML only. Every request that starts and ends inside the 10 s counts.
  Three rounds per target, the targets interleaved; the median and its range.
- **TTFB.** 50 requests one after another, each on a new connection, after 5 untimed ones, timed to the final
  response's headers (past Weft's 103 Early Hints; the first byte of the 103 is in the JSON as
  `first_byte_median_ms`).
- **Lighthouse.** Lighthouse 13.6.0, the default mobile preset (simulated throttling), headless Chrome, a fresh
  profile each run, five runs per target interleaved so drift hits all targets alike. Median and range.
- **Bytes.** The resources a Lighthouse run loaded (status 200), each fetched again. Text is counted as brotli
  quality 11, which is what Weft stores, for every site, whatever its server sends: that is every other site's best
  case, as if its host pre-compressed. "Inline JS" is JavaScript inside the HTML (`<script>` without `src`, not a
  JSON type), counted raw; it is already in the HTML's bytes. What each server really sent is in the second table.
- **Weft is not tuned for this.** The Weft site uses only what the page build does by default.

## Results

<!-- results:start -->
Run of 2026-11-25 ([`results/2026-11-25.json`](results/2026-11-25.json)).

Machine: Apple M4, 10 cores (4 performance), 32 GB, macOS 27.2.1 (26B131). Node v26.13.1, go1.26.2, Chrome 158.0.8127.88 headless. Weft at `857f786` with Vue 3.6.0-rc.16; Astro 7.4.13, @astrojs/vue 7.1.6, Vue 3.5.46; Nuxt 4.7.0, Vue 3.5.46; Next.js 16.4.6, React 19.3.1; SvelteKit 3.4.0, Svelte 5.59.2; Lighthouse 13.6.0.

| Site | Server | HTML+CSS (br) | JS files (br) | Inline JS (raw) | Total (br) | Requests | TTFB, ms | Lighthouse | LCP, ms | Req/s (Go servers at GOMAXPROCS=1) | p50 / p99, ms |
|---|---|--:|--:|--:|--:|--:|--:|--:|--:|--:|--:|
| Weft | weft binary (weft.Serve), GOMAXPROCS=1 | 3.1 KB | 20.8 KB | 0 B | 82.4 KB | 5 | 0.20 [0.18–0.25] | 100 [100–100] | 1276 [1276–1276] | 111,249 [108,824–123,643] | 0.57 / 1.13 |
| Astro, Vue island | astro preview | 3.5 KB | 26.5 KB | 5238 B | 88.5 KB | 5 | 0.31 [0.28–0.34] | 100 [100–100] | 1356 [1354–1357] | 19,354 [18,635–20,943] | 3.20 / 4.66 |
| Astro, Vue island | Go static + gzip, GOMAXPROCS=1 | 3.5 KB | 26.5 KB | 5238 B | 88.5 KB | 5 | 0.28 [0.25–0.30] | 100 [100–100] | 1353 [1351–1354] | 7,964 [7,924–7,988] | 8.19 / 13.49 |
| Astro, Vue island | Go, pre-compressed in memory, GOMAXPROCS=1 | 3.5 KB | 26.5 KB | 5238 B | 88.5 KB | 5 | 0.12 [0.11–0.14] | 100 [100–100] | 1351 [1350–1500] | 126,285 [122,218–136,192] | 0.48 / 1.19 |
| Astro, no framework island | astro preview | 1.7 KB | 0 B | 237 B | 60.2 KB | 2 | 0.26 [0.24–0.28] | 100 [100–100] | 752 [752–753] | 21,304 [21,258–22,036] | 2.89 / 4.25 |
| Astro, no framework island | Go static + gzip, GOMAXPROCS=1 | 1.7 KB | 0 B | 237 B | 60.2 KB | 2 | 0.21 [0.20–0.22] | 100 [100–100] | 751 [750–751] | 11,131 [11,090–11,271] | 5.86 / 9.13 |
| Astro, no framework island | Go, pre-compressed in memory, GOMAXPROCS=1 | 1.7 KB | 0 B | 237 B | 60.2 KB | 2 | 0.12 [0.11–0.14] | 100 [100–100] | 750 [750–750] | 129,955 [129,382–132,615] | 0.47 / 1.10 |
| Nuxt | its Node server (Nitro) | 1.9 KB | 62.3 KB | 153 B | 122.9 KB | 8 | 0.21 [0.19–0.23] | 99 [99–99] | 2252 [2251–2274] | 50,085 [49,515–51,533] | 1.23 / 2.18 |
| Nuxt | Go static + gzip, GOMAXPROCS=1 | 1.9 KB | 62.3 KB | 153 B | 122.8 KB | 7 | 0.20 [0.19–0.23] | 100 [100–100] | 1651 [1651–1653] | 13,061 [13,040–13,156] | 4.92 / 8.19 |
| Nuxt | Go, pre-compressed in memory, GOMAXPROCS=1 | 1.9 KB | 62.3 KB | 153 B | 122.8 KB | 7 | 0.12 [0.11–0.13] | 100 [100–100] | 1575 [1575–1576] | 129,176 [128,765–132,404] | 0.47 / 1.10 |
| Next.js | next start | 3.4 KB | 112.5 KB | 9255 B | 174.4 KB | 9 | 0.48 [0.46–0.52] | 99 [99–99] | 2014 [2009–2022] | 10,053 [9,976–10,233] | 6.20 / 9.17 |
| Next.js | Go static + gzip, GOMAXPROCS=1 | 3.4 KB | 112.6 KB | 9255 B | 174.5 KB | 9 | 0.27 [0.24–0.30] | 99 [99–99] | 2009 [1953–2151] | 7,797 [7,770–7,830] | 8.29 / 13.32 |
| Next.js | Go, pre-compressed in memory, GOMAXPROCS=1 | 3.4 KB | 112.6 KB | 9255 B | 174.5 KB | 9 | 0.12 [0.11–0.15] | 100 [99–100] | 1934 [1932–1935] | 131,974 [130,131–132,155] | 0.46 / 1.08 |
| SvelteKit | its Node server (adapter-node) | 2.0 KB | 34.9 KB | 530 B | 95.3 KB | 13 | 0.17 [0.15–0.20] | 100 [100–100] | 1505 [1503–1656] | 38,099 [37,977–38,630] | 1.45 / 3.77 |
| SvelteKit | Go static + gzip, GOMAXPROCS=1 | 2.0 KB | 34.9 KB | 532 B | 95.3 KB | 13 | 0.21 [0.20–0.22] | 100 [100–100] | 1504 [1502–1655] | 12,681 [12,593–12,758] | 5.04 / 8.70 |
| SvelteKit | Go, pre-compressed in memory, GOMAXPROCS=1 | 2.0 KB | 34.9 KB | 532 B | 95.3 KB | 13 | 0.12 [0.11–0.14] | 100 [100–100] | 1501 [1501–1501] | 129,418 [126,812–130,162] | 0.47 / 1.10 |

Bytes: KB is 1024 B. Text is counted as brotli quality 11 (what the Weft build stores), whatever the server sends; the image as it is. TTFB: median of 50 requests on new connections, [p25–p75]. Lighthouse and LCP: median of 5 mobile runs, [min–max]. Load: 64 connections, 2s warm-up, 10s measured, median of 3 rounds [min–max]; p50 and p99 are the medians of the rounds' own. No errors unless noted. The Go servers (Weft, Go static, Go pre-compressed) run at GOMAXPROCS=1; the frameworks' own servers are Node and are not held to one core.

What each server really sent (Accept-Encoding: br, gzip):

| Site | Server | Page encoding | JS on the wire | Total on the wire | LCP element |
|---|---|---|--:|--:|---|
| Weft | weft binary (weft.Serve), GOMAXPROCS=1 | br | 20.8 KB | 82.4 KB | `div.bench > main.weft-page > div.weft-page__body > img.bench-image` |
| Astro, Vue island | astro preview | gzip | 29.4 KB | 92.1 KB | `body > main.page > img.image` |
| Astro, Vue island | Go static + gzip, GOMAXPROCS=1 | gzip | 29.2 KB | 91.8 KB | `body > main.page > img.image` |
| Astro, Vue island | Go, pre-compressed in memory, GOMAXPROCS=1 | br | 26.5 KB | 88.5 KB | `body > main.page > img.image` |
| Astro, no framework island | astro preview | gzip | 0 B | 60.7 KB | `body > main.page > img.image` |
| Astro, no framework island | Go static + gzip, GOMAXPROCS=1 | gzip | 0 B | 60.7 KB | `body > main.page > img.image` |
| Astro, no framework island | Go, pre-compressed in memory, GOMAXPROCS=1 | br | 0 B | 60.2 KB | `body > main.page > img.image` |
| Nuxt | its Node server (Nitro) | identity | 188.1 KB | 252.2 KB | `body > div#__nuxt > main.page > img.image` |
| Nuxt | Go static + gzip, GOMAXPROCS=1 | gzip | 70.2 KB | 131.2 KB | `body > div#__nuxt > main.page > img.image` |
| Nuxt | Go, pre-compressed in memory, GOMAXPROCS=1 | br | 62.3 KB | 122.8 KB | `body > div#__nuxt > main.page > img.image` |
| Next.js | next start | gzip | 132.2 KB | 194.8 KB | `body > main.page > img.image` |
| Next.js | Go static + gzip, GOMAXPROCS=1 | gzip | 132.6 KB | 195.2 KB | `body > main.page > img.image` |
| Next.js | Go, pre-compressed in memory, GOMAXPROCS=1 | br | 112.6 KB | 174.5 KB | `body > main.page > img.image` |
| SvelteKit | its Node server (adapter-node) | br | 34.9 KB | 95.3 KB | `body > div > main.page > img.image` |
| SvelteKit | Go static + gzip, GOMAXPROCS=1 | gzip | 38.9 KB | 99.9 KB | `body > div > main.page > img.image` |
| SvelteKit | Go, pre-compressed in memory, GOMAXPROCS=1 | br | 34.9 KB | 95.3 KB | `body > div > main.page > img.image` |
<!-- results:end -->

## Reading the results

The numbers in the prose below are from the run in the table (2026-11-25).

- **Requests per second is a result about servers, not frameworks.** Behind the same Go server, answering from
  memory with brotli stored at build, the four other frameworks' pages do 126,285 to 131,974 requests per second.
  Weft's own binary does 111,249 [108,824–123,643]: the same order, and in this run below all of them. What
  `weft.Serve` gives is that serving in the binary, with nothing to set up. It does not make pages faster to send
  than a pre-compressed static site behind an equally careful server.
- **Each framework behind the server it ships with** is where the gap is: Weft 111,249; Nuxt's Node server 50,085;
  SvelteKit's 38,099; `astro preview` 19,354 (Vue island) and 21,304 (no framework); `next start` 10,053. Weft's
  server is held to one core and the Node servers are not. Nuxt's server sent the page and its scripts with no
  compression at all (252.2 KB on the wire against 122.8 KB as brotli), which is part of why it is the fastest of
  the Node servers; SvelteKit's sent brotli stored at build; `next start` and `astro preview` gzip each response.
- **A plain file server that gzips on the fly is the slowest way to serve any of them**: 7,797 to 13,061 requests
  per second, below every framework's own server but Next's.
- **JavaScript is the result about frameworks.** Weft's page loads 20.8 KB of script files; Astro with a Vue island
  26.5 KB plus 5.2 KB inline; SvelteKit 34.9 KB; Nuxt 62.3 KB; Next.js 112.5 KB plus 9.3 KB inline. Astro with no
  framework island loads none, only 237 B inline. Two confounds: the sites do not run the same framework runtime
  (Weft ships Vue 3.6.0-rc.16 with Vapor; Astro and Nuxt run Vue 3.5.46 with the virtual DOM; Next runs React;
  SvelteKit runs Svelte), and Nuxt and SvelteKit hydrate the whole page where Weft, Astro and Next hydrate only the
  counter, because that is each one's default.
- **Total transfer** follows the JavaScript, since the image is the same 59,862 B everywhere: 60.2 KB (Astro, no
  framework), 82.4 KB (Weft), 88.5 KB (Astro, Vue island), 95.3 KB (SvelteKit), 122.8 KB (Nuxt), 174.5 KB (Next.js).
- **Lighthouse scores 99 or 100 for all of them.** The score does not separate these pages; LCP does: 750 ms
  (Astro, no framework), 1276 ms (Weft), 1351–1356 ms (Astro, Vue island), 1501–1505 ms (SvelteKit), 1575–2252 ms
  (Nuxt, slowest behind its own uncompressed server), 1934–2014 ms (Next.js). The LCP column is Lighthouse's
  simulation (Lantern), not a load on a throttled device; see the side experiment.
- **TTFB** is a fraction of a millisecond everywhere on loopback. The pre-compressed Go server is fastest at
  0.12 ms; Weft's binary is at 0.20 ms, level with the Node servers of Nuxt (0.21) and SvelteKit (0.17). Too small
  to matter on a real network.
- **Weft's page changed since the run of 2026-11-20**: it now loads a third script, `boot-*.js` (553 B as brotli),
  so it makes five requests where it made four, and its simulated LCP is 1276 ms where it was 1200 ms.

## Where Weft loses

In the run of 2026-11-25:

| Measure | Weft | Who is ahead | Why, as best we can tell | Next step |
|---|--:|---|---|---|
| Requests per second | 111,249 | Every other site behind the pre-compressed Go server: 126,285 to 131,974 | Not measured. That server does less per request than `weft.Serve`: it sends no 103 Early Hints, no ETag and does no `If-None-Match` check. Hypothesis: the 103 before each page, an extra write, is most of the difference. | Run `load` against Weft with Early Hints off, and profile `weft.Serve` under load, before changing anything. |
| TTFB | 0.20 ms | The pre-compressed Go server, 0.12 ms, and SvelteKit's Node server, 0.17 ms | The same hypothesis: the timer runs to the final response's headers, past the 103. | As above. |
| HTML and CSS | 3.1 KB br | Astro with no framework (1.7 KB), Nuxt (1.9 KB), SvelteKit (2.0 KB) | Weft inlines its theme and the stylesheets of the components the page renders; the others carry one hand-written stylesheet. See the HTML and CSS row below. | As below. |
| JavaScript, total transfer, requests, LCP | 20.8 KB, 82.4 KB, 5, 1276 ms | Astro with no framework island: 237 B inline, 60.2 KB, 2, 750 ms | Astro's default island ships no framework. See the table below. | As below. |

Weft does not lose to Astro with a Vue island, Nuxt, Next.js or SvelteKit in JavaScript, total transfer or LCP, and
to Nuxt and SvelteKit only in HTML and CSS.

The analysis below is from the run of 2026-11-20 and was not repeated: its Weft figures are that run's (20.5 KB of
JavaScript in two files, four requests, 1200 ms), from before the page loaded `boot-*.js`. Every row but the last is
against **Astro, no framework island**, Astro's own default, served by either server; the last row is the one place
Weft loses to Astro with a Vue island, in the side experiment.

| Measure | Weft | Astro (no framework island, but for the last row) | Why, as best we can tell | Next step |
|---|--:|--:|---|---|
| JavaScript | 20.5 KB br in 2 files | 237 B inline | The island is a Vue component, so the page loads Weft's client chunk (`client-*.js`, 60,570 B raw): the Vapor runtime, reactivity and the island runtime. The counter's own chunk is 538 B. Astro's default island ships no framework at all. | Find out which part of those 60,570 B a Vapor island really needs (a bundle report of the client chunk) and cut the runtime to it; add a JS budget test for the benchmark page as `site-src` has. Consider an island form with no framework for trivial interactivity. |
| LCP (Lighthouse, mobile, simulated, Lantern) | 1200 ms | 751 ms | Hypothesis, supported by the side experiment: Lantern charges the LCP for every script the page loads, whenever it loads it. Moving Weft's module script to the end of the body (1201 ms) or importing it at idle (1206 ms) changes nothing; removing it gives 751 ms, the same as Astro with no framework. Astro's Vue island, already loaded at idle by an inline loader, is at 1353–1357 ms. So the simulated loss is the cost of loading about 20 KB of JavaScript at all, not of where the script sits. Under real (DevTools) throttling the gap shrinks to about 90 ms (1633 against 1546 and 1543 ms). | Not an inline loader: the experiment shows it does not help. Test first, then build: (1) the same page with `fetchpriority="high"` and `loading="eager"` on the LCP image, which the `<Image>` plan would write, under both throttling methods; (2) a smaller client chunk (see the JavaScript row). Read Lighthouse's LCP breakdown for each. |
| HTML and CSS | 2.9 KB br | 1.7 KB br | The page inlined 7,989 B of CSS (about 1.8 KB of the 2.9 KB brotli): the whole neutral theme, light and dark tokens, and `weft.css` with the rules of every layout component, used or not. Since then each component ships its own stylesheet and a page inlines only the theme, `base.css` and the stylesheets of the components it renders, minified for Vite's default browser targets (2026-11-21, measured on the built page, not in a new run, which needs network installs): 8,410 B of CSS, 1,763 B brotli on its own, and 2,940 B brotli for the page. That is no smaller than on the run's day, because this page renders five of the eight layout components (Page, Section, Stack, Grid, Card) and the theme has grown (3,276 B of the CSS); without per-page CSS, the `weft.css` of today, with the basic and form components, would have made it 35,320 B of CSS and 6,494 B brotli. The Astro page carries only its own hand-written stylesheet, so part of this is the price of a UI kit, not of the page build. Vue's hydration markers are another 80 markers, 622 B raw, 53 B of the brotli: 46 `<!--[-->`/`<!--]-->`, 18 `<!---->` and 16 Vapor `<!--{-->`/`<!--}-->` (without the Vapor kind, 64 markers and 494 B). All 80 are outside the island, where nothing hydrates. How each number was measured is in `weft_html_breakdown` in `results/2026-11-20.json`. | Inline only the tokens a page uses (the CSS of the components it renders is done); drop hydration markers outside islands, since nothing hydrates there. |
| Total transfer | 81.8 KB br | 60.2 KB br | The JavaScript above; the image is the same 59,862 B in both. | As for JavaScript. |
| Requests | 4 | 2 | The client chunk and the island's chunk. | As for JavaScript. |
| LCP against **Astro, Vue island** (side experiment, DevTools throttling, both behind `static/`) | 1633 ms [1616–1637] | 1582 ms [1565–1598] | Hypothesis: on a really throttled network the JavaScript shares bandwidth with the hero image. Lighthouse's LCP breakdown puts the difference in the image's load duration (median 970 ms for Weft against 931 ms for Astro's Vue page and 896–899 ms with no JS; the other subparts are within about 20 ms). Weft's module script is fetched at once, at high priority; Astro fetches its Vue runtime later, from its idle loader. Weft's idle-import variant was slower still (1679 ms), so later loading alone does not explain it. | The same tests as the LCP row above, read under DevTools throttling: a high-priority image first, then a smaller client chunk. |

## Side experiment: where the island script loads, and LCP

A labelled side experiment, run after the main run of 2026-11-20 on the same machine, with no change to Weft's code
([`tools/lcp-experiment.mjs`](tools/lcp-experiment.mjs); every run is in `results/2026-11-20.json` under
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
