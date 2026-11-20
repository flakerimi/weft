# Weft Showcase Site Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace weft.base.al with the redesigned showcase site, built by Weft and served by the Weft binary. It has four parts: a landing page with the owner's wordmark and live adaptive demos, docs written in Markdown, component pages generated from a catalogue, and the Weft skills plus `llms.txt` and `AGENTS.md` published from the repository.

**Architecture:** `site-src/` grows into a full Weft site:

- **Pages:** `pages/index.vue` is the landing page, `pages/docs/[...slug].vue` renders the docs, `pages/components/[name].vue` renders the component pages, and `pages/skills/[name].vue` renders the skills.
- **Markdown:** a small Markdown pipeline in `@weft/vite` reads `content/docs/**/*.md`. Frontmatter is checked against a schema at build time.
- **Catalogue:** `catalog/components.json` describes the existing components. It is the same source `weft catalog` will read later.
- **Skills:** `skills/*/SKILL.md` in the repository is rendered as pages and also served raw for download.
- **Binary:** releases move out of the embedded build into a directory the binary serves from disk, so the image stops growing with every release.

**Tech Stack:** the existing Weft packages (Vue `3.6.0-rc.16` Vapor, `@weft/vite`, `weft.Serve`). Markdown is rendered with `markdown-it` (check the latest with `npm view`). Fonts are self-hosted from `@fontsource-variable/bricolage-grotesque 5.3.0` and `@fontsource-variable/jetbrains-mono 5.3.0`.

**Spec:** `docs/superpowers/specs/2026-11-20-weft-design.md`, phase 2c. The visual reference is `docs/design/site-2026-11-21/` (the approved mockup: `index.html` plus screenshots).

## Global Constraints

- **Inherited:** everything in the fast-pages plan's Global Constraints, plus the final fix-wave rules on what a build may write.
- **Wordmark:** use the owner's SVG exactly, as stored in `site-src/public/weft.svg`. Do not redraw it or substitute a font for it.
- **Colours and type:**
  - Ground `#E8ECF4`, ink `#141C3A`, indigo `#2B3D8F`, saffron `#E2A11B`, warp line `#C9D0E0`.
  - Dark ground `#0F1530`, dark warp line `#1C2448`.
  - Type: Bricolage Grotesque for headlines and text, JetBrains Mono only for code.
- **Motion:** exactly one automatic motion on the page, the hero thread pulling across, and it respects reduced motion.
- **Size budgets (brotli, all bytes):**
  - Landing page without islands ≤ 12 KB (fonts excluded, since they are cached and preloaded).
  - Each docs page ≤ 10 KB.
  - Landing page with its islands ≤ 30 KB.
- **No third-party requests at runtime:** no font CDN and no analytics.

## Review Focus

1. A Markdown file with invalid frontmatter, or a link to a missing doc, fails the build. The error names the file and line.
2. The landing page reads correctly with JavaScript disabled. The islands are enhancements only.
3. The hero thread never causes horizontal scrolling at any width from 320 px to 2560 px.
4. Every release previously served (0.0.1–0.0.3+) still installs after releases move out of the binary. `releases.txt` stays the source of truth.
5. Dark mode has no unreadable pair: text and controls meet WCAG AA contrast.

---

### Task 1: Releases out of the binary
`weft.Serve` gains `WithFiles(prefix string, dir fs.FS)`, which serves a read-only disk directory under a path prefix with ETags computed at start. `site-src/main.go` serves `/releases/` and `/install` from `$WEFT_RELEASES_DIR` (default `/data/releases`). The Dockerfile copies releases into the image under that path instead of embedding them. The publish script and its tests are updated, and the binary size test asserts under 15 MB. Tests cover the ETag, the 304 response and path traversal.

### Task 2: Markdown docs pipeline
- **Rendering:** `@weft/vite` gets `content.ts`. It reads `content/<collection>/**/*.md` and checks frontmatter against a schema per collection (`title` required, `description` required, `order?` number). It renders the Markdown with `markdown-it`, turns heading anchors into ids, and highlights code blocks at build time with no client JavaScript (pick a highlighter that runs in Node only, and check the latest version).
- **Pages:** files are exposed to pages through `useContent(collection, slug)` on the server.
- **Link check:** internal links are checked at build time.
- **Tests:** invalid frontmatter, a broken link, a missing title, and heading ids.

### Task 3: Docs content
`content/docs/` holds the following:
- **Getting started:** install, first site, first Go handler.
- **Guides:** pages and islands, Go serving, themes, the adaptive model, publishing.
- **Reference:** the CLI and the `weft` Go package.

Write it all from the code as it is today. Do not document features that do not exist; list them under "Coming next" with the spec phase. Then add the docs layout: a sidebar from `order`, a page table of contents, and previous and next links.

### Task 4: Catalogue and component pages
- **Catalogue:** `catalog/components.json` covers `Stack`, `Grid`, `Page`, `Section`, `Card`, `Divider`, `WeftProvider` and `Island`, with each component's props, slots, events, adaptive behaviour, an example, and when to use it and when not.
- **Freshness:** a test fails when a component's props in its source differ from the catalogue.
- **Pages:** `pages/components/[name].vue` renders each entry with a live example. The example is an island only when it is interactive.

### Task 5: Skills, `llms.txt`, `AGENTS.md`
- **Skills:** `skills/` holds `weft-site` (build a site with pages and islands), `weft-go` (serve and publish), and `weft-theme`, each with `SKILL.md` frontmatter (`name`, `description`). Each is rendered at `/skills/<name>` and served raw at `/skills/<name>.md`.
- **`llms.txt`:** generated at build from the docs, components and skills.
- **`AGENTS.md`:** at the repository root.
- **Test:** every skill has valid frontmatter, and every skill is linked from `llms.txt`.

### Task 6: The landing page
Port the approved mockup (`docs/design/site-2026-11-21/index.html`) to Weft components and tokens:
- **Hero:** the thread is CSS. The copy button is an island.
- **Adaptive demo:** an island where switching between pointer and touch re-renders the demo with `WeftProvider`. The date picker is a labelled mock until the real component exists.
- **Benchmark bars:** read from `bench/results/<latest>.json` at build time.
- **Agents section:** reads its example from the catalogue.
- **Fonts:** self-hosted and preloaded.
- **Checks:** a budget test, plus screenshots at 1440 and 500 px, light and dark, compared by eye against the mockup.

### Task 7: Publish and verify
Publish with `scripts/publish-site.sh <next version>` (the controller does this). Then verify live:
- the landing page, a docs page, a component page and a skill's raw file;
- the installer, pinned installs of every release in `releases.txt`, and `weft update`;
- brotli, 304 responses and Early Hints;
- the size budgets, measured on the live site.

Then re-run the benchmark's Weft row against the live landing page for the record.
