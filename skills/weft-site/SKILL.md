---
name: weft-site
description: Use when building or changing a site made with Weft's page build - adding pages and routes, dynamic pages with getPaths, islands (the only parts that ship JavaScript), Markdown collections in content/, the head, raw files with useFile, and running weft-build-pages - and when a page build fails, to learn which check refused it.
---

# Building a site with Weft

A Weft site is a folder of Vue pages. `weft-build-pages` renders each page to HTML at build time, inlines the CSS of what it renders, and compresses everything. JavaScript ships only for what you wrap in an `Island`. Serving is the `weft-go` skill; choosing components is `weft-components`.

## Where you work

An app is made with `weft create <name>`: the `weft` command carries Weft's packages, which are not on npm, and unpacks them into the app's `.weft/`, which you never edit. You need Go 1.26, Node 22.18 or later and pnpm 10. The site is `web/`, its Go API `app/`:

- `weft dev` runs the page dev server beside the Go API, rendering each page when asked for and reloading the browser when a file changes;
- `weft build` runs `weft-build-pages --root web --out web/dist` with the app's theme, then builds the binary. It prints `weft: built N pages into web/dist`.

## The layout of a site

```text
myapp/
  package.json          "weft": {"version", "theme"}; weft build and weft dev read it
  web/
    pages/              one .vue file per route
    components/         your components and islands
    content/<name>/     Markdown collections (content.config.ts)
    content.config.ts   only if the site has content/
    public/             copied to the build's root as it is
    vite.config.ts      optional; the build adds Vue, icons and content itself
    tsconfig.json       written by weft create
  app/                  the Go API (see weft-go)
  main.go               the server (see weft-go)
```

The build writes icon tags in every page's `<head>` itself, one for each of `favicon.ico`, `favicon.svg` and `apple-touch-icon.png` in `web/public/`, with no setting: leave them out of `useHead`.

Inside the Weft repository, a site is a workspace package whose `build` script runs `weft-build-pages --root . --out dist` (like `site-src/`); `weft-dev --root <site>`, from `@weft/vite`, is its page dev server.

## Pages and routes

| File | Route |
| --- | --- |
| `pages/index.vue` | `/` |
| `pages/about.vue` | `/about` |
| `pages/blog/index.vue` | `/blog` |
| `pages/notes/[slug].vue` | `/notes/<slug>`, one page per value |
| `pages/docs/[...slug].vue` | `/docs` and the paths under it that its `getPaths()` returns, slashes included; any other is the 404 page |
| `pages/404.vue` | every path with no page, with status 404 |

Names starting with `_` or `.` are skipped. A page is a Vapor component:

```vue
<script setup vapor lang="ts">
import { Card, Grid, Page, Section, useHead } from '@weft/ui'

useHead({ title: 'Hello', description: 'My first Weft site.' })
</script>

<template>
  <Page title="Hello" description="My first Weft site.">
    <Section title="What you see">
      <Grid><Card>Rendered when you built it.</Card></Grid>
    </Section>
  </Page>
</template>
```

A dynamic page exports `getPaths()` from a plain `<script>` block and gets each value as a prop:

```vue
<script lang="ts">
export const getPaths = async () => [{ slug: 'first' }, { slug: 'second' }]
</script>

<script setup vapor lang="ts">
const props = defineProps<{ slug: string }>()
</script>
```

A page for records made after the build exports `runtime = true` instead: built once, it is served as a shell for every matching path no other page has. An island on it reads the params in the browser with `useParams()`, in `onMounted`.

`useHead({ title, description, image })` writes `<title>`, description and Open Graph tags. Every page also gets speculation rules and view transitions.

## Islands: the only JavaScript

Wrap only what must run in the browser, such as a form, a menu or a dialog:

```vue
<Island when="visible" :component="Counter" :props="{ start: 0 }" />
```

- `when` is `load`, `idle`, `visible` or `media`. With `media`, also give `media="(min-width: 1024px)"`.
- The island's file name is its id: two islands may not share one.
- Props must survive JSON: strings, finite numbers, booleans, `null`, arrays and plain objects; anything else fails the build, naming the prop.
- The server renders the island too, so it shows before any script runs. Read input and width through `useWeft()` and `useWidth()`, which answer `pointer` and `wide` on the server.
- `usePrePaint(id, code)` runs before the first paint: to measure width-adaptive components (`useWidth`), copy a stored preference onto `<html>`, or, as `Form` does, stop a submit before the island is live (so `Form` belongs in an island).

To list the pages that load JavaScript:

```sh
grep -rl '<script type="module"' web/dist/pages --include=index.html
```

## Markdown content

Declare each folder of `content/` in `content.config.ts`:

```ts
import type { ContentConfig } from '@weft/vite'

export default { docs: { schema: { section: 'string?' } } } satisfies ContentConfig
```

- Every page needs `title` and `description`. `order` is optional, and the `schema` adds fields of its own.
- A collection takes `base`, the route it is served at (default `/<name>`); `dir`, a folder outside `content/` such as `../skills`; `index`, the file that is its folder's page, such as `SKILL.md`; and `titleField`, which reads the title from another field, such as `name`.
- A collection's folder holds only pages and their images; any other file fails the build, so a skill kept as a collection (`skills/<name>/SKILL.md`) cannot yet carry supporting files like scripts or templates.

Pages read content on the server only, from `virtual:weft/content`:

- `useContent(collection, slug)` returns `html`, `toc`, `frontmatter`, `prev`, `next`, `parts` and `source`;
- `useCollection(collection)` lists the pages for a sidebar;
- `contentPaths(collection)` is what `getPaths()` returns.

A code block marked `sh copy` becomes a part of its own, so the page can put a copy island beside it.

## Raw files beside the pages

A page's setup can write a file into the build, such as JSON for agents, compressed and listed in the manifest:

```ts
import { useFile } from 'virtual:weft/files'

useFile('/catalog.json', text)
```

The path starts with `/` and is not under `/pages/` or `/assets/`. Two pages may write one file only with identical bytes.

## Checks that fail the build

The build stops with one sentence naming the file, and often the line, when:

- two files make the same route, or a dynamic page has neither `getPaths()` nor `runtime`, or both;
- an island's prop is not JSON, or an island is not a `.vue` file the page imports;
- an island imports `virtual:weft/content` or `virtual:weft/files`, which exist only on the server;
- Markdown frontmatter is not valid YAML, misses a field, or has one of the wrong type or one the schema lacks;
- a link to a `.md` file, a heading or a route in a collection leads nowhere, or an image is missing;
- a code block's language is unknown to the highlighter;
- a CSS `url()` names no file;
- `--out` holds files an earlier build did not write;
- `--theme` is not `neutral`, `lisa` or `weft`.

Fix the cause; never weaken the check.

## Do not

- Use `createApp`, `createVaporApp` or a client router for pages: the build renders them and `Island` hydrates.
- Make an island of content that never changes, or of a whole page.
- Read `window`, `document` or `localStorage` during setup; do it in `onMounted`.
- Add a dependency without checking the latest version with `npm view <pkg> version`.
- Commit `dist/`: it is ignored and rebuilt.

## Not yet

Planned, not built (see https://weft.base.al/docs/coming-next):

- app routes: a client-side router and `layouts/`. An app is pages rendered at build, with islands, and run-time pages for records;
- an `<Image>` component, actions, environment values, a development overlay and `weft deploy`;
- `weft scaffold`, `DataTable` and `EntityDialog`.
