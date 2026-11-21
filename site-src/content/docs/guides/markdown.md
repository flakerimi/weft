---
title: Markdown content
description: Write pages in Markdown, checked and rendered when you build.
order: 21
---

On this page you set up a folder of Markdown pages, render them through a Vue page, and learn what the build checks. These docs are made this way. Nothing here runs in the browser.

## Declare a collection

Each folder of `content/` is a collection. Declare it in `content.config.ts` at the site's root:

```ts
import type { ContentConfig } from '@weft/vite'

export default {
  docs: { schema: { section: 'string?' } },
} satisfies ContentConfig
```

- **`base`** is where the collection's pages are served. The default is `/<name>`, so `docs` is served at `/docs`.
- **`schema`** lists the frontmatter fields the collection adds. A field is `string`, `number`, `boolean` or `string[]`, with `?` when it is optional.
- **Every collection** has `title` and `description` (required strings) and `order` (an optional number).

Every folder of `content/` must be declared, and every declared collection must have a folder.

## Write pages

A page is a `.md` file with frontmatter:

```md
---
title: Islands
description: The parts of a page that run in the browser.
order: 2
---

An island is a component that runs in the browser.
```

Its slug is its path in the collection without `.md`. `index.md` takes its folder's slug, so `content/docs/index.md` is `/docs` and `content/docs/guides/islands.md` is `/docs/guides/islands`. Names are lowercase letters, digits, `-` and `_`.

Images beside a page, such as `![Flow](./flow.png)`, are copied into the build under a name made from a hash of their bytes, so browsers cache them for good. A collection's folder may hold only pages and images.

## Render them

A page reads content through `virtual:weft/content`, which exists only on the server. This is a catch-all page for the `docs` collection, `pages/docs/[...slug].vue`:

```vue
<script lang="ts">
import { contentPaths } from 'virtual:weft/content'

export const getPaths = async () => contentPaths('docs')
</script>

<script setup vapor lang="ts">
import { useHead } from '@weft/ui'
import { useContent } from 'virtual:weft/content'
import '@weft/vite/code.css'

const props = defineProps<{ slug: string }>()
const page = useContent('docs', props.slug)
useHead({ title: page.frontmatter.title, description: page.frontmatter.description })
</script>

<template>
  <article v-html="page.html" />
</template>
```

- **`useContent(collection, slug)`** returns the page: `html`, `toc` (the h2 and h3 headings), `frontmatter`, `prev` and `next`, and `parts`.
- **`useCollection(collection)`** returns every page, ordered by `order` and then by slug, for a sidebar.
- **`contentPaths(collection)`** returns the slugs for `getPaths()`.

For TypeScript to know the module, add its types to the site's `tsconfig.json`:

```json
{ "compilerOptions": { "types": ["vite/client", "@weft/vite/virtual"] } }
```

An island cannot import `virtual:weft/content`; the build refuses it. Pass the island what it needs as props.

## What the build checks

The build stops with one sentence that names the file and line when:

- the frontmatter is not valid YAML, misses a required field, has a field of the wrong type, or has a field the schema does not list;
- a link to another `.md` file, or to a heading with `#id`, leads nowhere;
- a link from the site root into a collection, such as `/docs/nope`, names no page;
- an image is missing, or is not a `.png`, `.jpg`, `.jpeg`, `.gif`, `.webp`, `.avif` or `.svg` file;
- a code block's language is one the highlighter does not know.

Relative `.md` links become routes: `[islands](../guides/islands.md#when)` becomes `/docs/guides/islands#when`. Links to other sites are left as they are and never fetched.

## Rendering

- Headings get ids by GitHub's rules. A second "Install" on a page gets `install-1`.
- Code blocks are highlighted at build time. Their colours are CSS variables that `@weft/vite/code.css` sets from the theme, so one rendering reads in light and dark.
- Raw HTML in Markdown is shown as text, not rendered.

## Copy buttons

A code block has no copy button unless you write `copy` after its language:

````md
```sh copy
weft update
```
````

`page.parts` then cuts the page at that block: the block is a part of its own, `{ html, code, lang }`, and your page renders a copy island beside it with `code` as its prop. Only pages with such a block load a script. A `copy` block must sit at the top level of the page, not inside a list or a quote.
