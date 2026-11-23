---
name: weft-site
description: Use when building or changing a site made with Weft's page build - adding pages and routes, dynamic pages with getPaths, islands (the only parts that ship JavaScript), Markdown collections in content/, the head, raw files with useFile, and running weft-build-pages - and when a page build fails and you need to know which check refused it.
---

# Building a site with Weft

A Weft site is a folder of Vue pages. `weft-build-pages` renders each page to HTML at build time, inlines the CSS of what it renders, and compresses everything. JavaScript ships only for what you wrap in an `Island`. Serving is the `weft-go` skill; choosing components is `weft-components`.

## Where you work today

The packages are not on npm, so a site lives in a clone of the Weft repository, a pnpm workspace. You need Node 22.18 or later, pnpm 10 and Go 1.26. Run commands from the repository root.

```sh
pnpm install
pnpm --filter @weft/themes build
pnpm --filter @weft/ui build
```

## The layout of a site

```text
hello/
  package.json        "build": "weft-build-pages --root . --out dist"
  pages/              one .vue file per route
  components/         your components and islands
  content/<name>/     Markdown collections, declared in content.config.ts
  content.config.ts   only if the site has content/
  public/             copied to the build's root as it is
  vite.config.ts      optional; the build adds Vue, icons and content itself
  tsconfig.json       "types": ["vite/client", "@weft/vite/virtual"]
  main.go             the server (see weft-go)
```

To add a site:

1. Add the folder to `pnpm-workspace.yaml`.
2. Give it a `package.json` with:
   - dependencies `"@weft/ui": "workspace:*"` and `"vue": "3.6.0-rc.16"`, pinned exactly;
   - dev dependencies `"@weft/vite": "workspace:*"`, `"@vitejs/plugin-vue": "^6.0.11"` and `"vite": "^8.3.8"`.
3. Run `pnpm install`.
4. Run `pnpm --filter hello build`. It prints `weft: built N pages into dist`.

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

`useHead({ title, description, image })` writes the `<title>`, the description and the Open Graph tags. Every page also gets speculation rules and view transitions.

## Islands: the only JavaScript

Render static content directly. Wrap only what must run in the browser, such as a form, a menu or a dialog:

```vue
<Island when="visible" :component="Counter" :props="{ start: 0 }" />
```

- `when` is `load`, `idle`, `visible` or `media`. With `media`, also give `media="(min-width: 1024px)"`.
- The island's file name is its id. Two islands may not share a file name.
- Props must survive JSON: strings, finite numbers, booleans, `null`, arrays and plain objects. Anything else fails the build, which names the prop.
- The server renders the island too, so it shows before any script runs. Read input and width through `useWeft()` and `useWidth()`, which answer `pointer` and `wide` on the server.
- `usePrePaint(id, code)` is only for copying a stored preference onto `<html>` before the first paint.

A page with no island loads no JavaScript. To list the pages that do load it:

```sh
grep -rl '<script type="module"' hello/dist/pages --include=index.html
```

## Markdown content

Declare each folder of `content/` in `content.config.ts`:

```ts
import type { ContentConfig } from '@weft/vite'

export default { docs: { schema: { section: 'string?' } } } satisfies ContentConfig
```

- Every page needs `title` and `description`. `order` is optional, and the `schema` adds fields of its own.
- A collection takes `base`, the route it is served at (default `/<name>`).
- It also takes `dir`, a folder outside `content/` such as `../skills`.
- `index` names the file that is its folder's page, such as `SKILL.md`.
- A collection's folder holds only pages and their images; any other file fails the build. So a skill kept as a collection (`skills/<name>/SKILL.md`) cannot carry supporting files, such as scripts or templates, beside it yet.
- `titleField` reads the title from another field, such as `name`.

Pages read the content on the server only, from `virtual:weft/content`:

- `useContent(collection, slug)` returns `html`, `toc`, `frontmatter`, `prev`, `next`, `parts` and `source`;
- `useCollection(collection)` lists the pages for a sidebar;
- `contentPaths(collection)` is what `getPaths()` returns.

A code block marked `sh copy` becomes a part of its own, so the page can put a copy island beside it.

## Raw files beside the pages

A page's setup can write a file into the build, such as JSON for agents. It is compressed and listed in the manifest:

```ts
import { useFile } from 'virtual:weft/files'

useFile('/catalog.json', text)
```

The path starts with `/` and may not be under `/pages/` or `/assets/`. Two pages may write the same file only with the same bytes.

## Checks that fail the build

The build stops with one sentence naming the file, and often the line, when:

- two files make the same route, or a dynamic page has no `getPaths()`;
- an island's prop is not JSON, or an island is not a `.vue` file the page imports;
- an island imports `virtual:weft/content` or `virtual:weft/files`, which exist only on the server;
- Markdown frontmatter is not valid YAML, misses a field, or has one of the wrong type or one the schema lacks;
- a link to a `.md` file, a heading or a route in a collection leads nowhere, or an image is missing;
- a code block's language is unknown to the highlighter;
- a CSS `url()` names no file;
- `--out` holds files an earlier build did not write;
- `--theme` is not `neutral`, `lisa` or `weft`.

Fix the cause. Never weaken the check.

## Do not

- Use `createApp`, `createVaporApp` or a client router for pages: the build renders them and `Island` hydrates.
- Make an island of content that never changes, or of a whole page.
- Read `window`, `document` or `localStorage` during setup; do it in `onMounted`.
- Add a dependency without checking the latest version with `npm view <pkg> version`.
- Commit `dist/`: it is ignored and rebuilt.

## Not yet

These are planned but not built (see https://weft.base.al/docs/coming-next):

- `weft create`, `weft dev` and `weft build`. Today there is no dev server for a page build: rebuild with `pnpm --filter <site> build`;
- an `<Image>` component, actions, environment values, a development overlay and `weft deploy`;
- `weft add <block>`.
