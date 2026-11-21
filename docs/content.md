# Markdown content

A Weft site can keep pages as Markdown under `content/`. `@weft/vite` reads them at build time. It checks them, renders them to HTML and hands them to the site's pages. Nothing in this pipeline runs in the browser.

## Collections

Each folder of `content/` is a collection, and `content.config.ts` at the site root declares them:

```ts
import type { ContentConfig } from '@weft/vite'

export default {
  docs: { schema: { section: 'string?' } },
} satisfies ContentConfig
```

- **`base`** is where the collection's pages are served, from the site root. The default is `/<name>`, so `docs` is served at `/docs`.
- **`schema`** lists the frontmatter fields the collection adds, each with a type: `string`, `number`, `boolean` or `string[]`. A `?` after the type makes the field optional.
- **Fields every collection has:** `title` (a string, required), `description` (a string, required) and `order` (a number, optional).

A page's slug is its path in the collection without `.md`. `index.md` takes its folder's slug, so `content/docs/index.md` is `''` (served at `/docs`) and `content/docs/guides/islands.md` is `guides/islands`. File and folder names must be plain URL segments: lowercase letters, digits, `-` and `_`. Names that start with `.` or `_` are skipped.

## Checks

The build fails with one sentence naming the file and line when:

- **The frontmatter is wrong:** it is not valid YAML, a required field is missing, a field has the wrong type, or a field is not in the schema.
- **A link leads nowhere:**
  - a relative `.md` link names a file that does not exist;
  - a link names a heading the target page does not have, on another page or with `#id` on the same page;
  - a link from the site root into a collection (`/docs/...`) names a page or heading that is not there.
- **A link or image is relative but cannot be served:**
  - a relative link that is not to a `.md` file;
  - a relative link that leads outside `content/`;
  - a relative image. Put images in `public/` and give their path from the site root.
- **A code block cannot be highlighted:** its language is one Shiki does not know, or its fence has a word after the language other than `copy`.
- **The files are wrong:** a file in a collection is not `.md`, two files make one page, a folder of `content/` is not declared, or a declared collection has no folder.

Relative `.md` links become the routes of their pages: `[islands](../guides/islands.md#when)` becomes `/docs/guides/islands#when`. External links are left as they are and never fetched. A path from the site root outside every collection (such as `/components/Button`) is not checked.

## Rendering

- **Markdown** is rendered with markdown-it (CommonMark plus tables and strikethrough). Raw HTML is not rendered, so it shows as text.
- **Headings** get ids by GitHub's slug rules, unique on the page: a second "Install" gets `install-1`. The h2 and h3 headings make the table of contents.
- **Code blocks** are highlighted at build time with Shiki. The colours are CSS variables (`--weft-code-token-keyword` and the rest). `@weft/vite/code.css` sets them from the theme's colours, so one rendering reads in light and dark, and every token colour meets AA contrast on the code's ground. A block with no language is plain text.

## Pages

Pages read content through `virtual:weft/content`, which exists only on the server:

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

- **`useContent(collection, slug)`** returns the page: `html`, `toc` (`{ depth, id, text }` for each h2 and h3), `frontmatter`, `prev` and `next` (`{ slug, path, title, description, order }`, or `null`), and `parts`.
- **`useCollection(collection)`** returns every page of the collection in order, for a sidebar. Pages are ordered by `order`, with unordered pages last, then by slug.
- **`contentPaths(collection, param = 'slug')`** returns the params for `getPaths()`.

`pages/docs/[...slug].vue` is a catch-all route. It takes the rest of the path, slashes included, and a slug of `''` makes `/docs` itself.

An island runs in the browser and cannot import `virtual:weft/content`; the build refuses it. Pass the island what it needs as props.

### Copy buttons

Code blocks have no copy button by default, so a docs page ships no script. To give a block one, write `copy` after its language:

````md
```sh copy
pnpm add @weft/ui
```
````

The page is then cut at that block. In `page.parts`, the block is a part of its own, `{ html, code, lang }`, and the rest are `{ html }` parts. The page renders the block's HTML and puts its copy island beside it, passing `code` as a prop. A `copy` block must be at the top level of the page, not inside a list or a quote.

## Not yet

Weft components inside Markdown (MDX-style tags such as `<Button>` in a `.md` file) are out of scope for now. A page that needs a component renders it in its `.vue` file, around the Markdown.
