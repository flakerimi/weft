---
title: Pages and islands
description: Add pages and routes, set the head, and make parts of a page interactive.
order: 20
---

On this page you learn how files in `pages/` become routes, how a page sets its head, and how to make part of a page run in the browser with an island. The examples continue the `hello` site from [Your first site](../getting-started/first-site.md).

## Routes

Every `.vue` file under `pages/` is a page, and its path is its route:

| File | Route |
| --- | --- |
| `pages/index.vue` | `/` |
| `pages/about.vue` | `/about` |
| `pages/blog/index.vue` | `/blog` |
| `pages/notes/[slug].vue` | `/notes/<slug>`, one page per slug |
| `pages/docs/[...slug].vue` | `/docs`, and every path under it |
| `pages/404.vue` | the page for paths with no page, sent with status 404 |

Files and folders whose names start with `_` or `.` are skipped. Two files that make the same route fail the build, which names both.

### Pages with params

A page with `[name]` in its file name is built once per value. It exports `getPaths()` from a plain `<script>` block, and gets each value as a prop. Write `hello/pages/notes/[slug].vue`:

```vue
<script lang="ts">
export const getPaths = async () => [{ slug: 'first' }, { slug: 'second' }]
</script>

<script setup vapor lang="ts">
import { Page } from '@weft/ui'

const props = defineProps<{ slug: string }>()
</script>

<template>
  <Page :title="`Note: ${props.slug}`" />
</template>
```

The build makes `/notes/first` and `/notes/second`. A `[name]` value is one path segment; a `[...name]` value may hold several, and `''` is the folder itself.

## The head

`useHead` sets the `<title>`, the description and the Open Graph image:

```ts
import { useHead } from '@weft/ui'

useHead({ title: 'Notes', description: 'Things I wrote down.', image: '/notes.png' })
```

The build writes them into the page's `<head>`, with `og:title` and `og:description`.

## What a page gets

Every page the build writes has:

- its CSS inlined in a `<style>`: the theme, Weft's base styles, and the stylesheets of the components the page renders. If that is over 14 KB brotli, it goes in one stylesheet the page links instead.
- speculation rules, so the browser fetches a same-site link when the pointer rests on it. Links under `/api` are left out, and so is any link inside an element marked `data-weft-no-prefetch` (or marked so itself), such as a demo whose links name pages that do not exist.
- `@view-transition { navigation: auto; }`, so going from page to page cross-fades in browsers that support it.

A page with no island has no script that loads JavaScript.

Files in `public/` are copied to the root of the build as they are: `public/robots.txt` is `/robots.txt`.

### Files a page writes

A page can also write a file of its own into the build, such as data it shows, for agents or other programs to read. Call `useFile(path, text)` from `virtual:weft/files` in the page's setup:

```ts
import { useFile } from 'virtual:weft/files'

useFile('/catalog.json', catalogText)
```

The file is compressed and listed in the manifest like any other, and revalidated rather than cached for good. Its path starts with `/` and may not be under `/pages/` or `/assets/`. Two pages may write the same file only with the same bytes. Like `virtual:weft/content`, it exists only on the server. This site writes its component catalogue, `/llms.txt` and its skills' raw files this way.

## Islands

An island is a component that runs in the browser. The rest of the page is HTML. Write the component, `hello/components/Counter.vue`:

```vue
<script setup vapor lang="ts">
import { ref } from 'vue'
import { Button } from '@weft/ui'

const props = defineProps<{ start: number }>()
const count = ref(props.start)
</script>

<template>
  <Button @click="count++">Pressed {{ count }} times</Button>
</template>
```

Put it on a page with `Island`, in `hello/pages/counter.vue`:

```vue
<script setup vapor lang="ts">
import { Island, Page } from '@weft/ui'
import Counter from '../components/Counter.vue'
</script>

<template>
  <Page title="Counter">
    <Island when="visible" :component="Counter" :props="{ start: 0 }" />
  </Page>
</template>
```

The server renders the button, so it shows before any script runs. The island runtime then makes it interactive when `when` says:

| `when` | Hydrates |
| --- | --- |
| `load` | as soon as the page loads |
| `idle` | when the browser is idle |
| `visible` | when the island scrolls into view |
| `media` | when the query in `media` matches, such as `(min-width: 1024px)` |

### What an island needs

- **A `<script setup>` component.** Its file name is the island's id, so two islands may not share a file name.
- **Props that JSON can carry:** strings, finite numbers, booleans, `null`, arrays and plain objects. The build refuses anything else and names the prop.
- **Its own CSS.** The page inlines the CSS of everything the island's file uses, not only what the server drew, since the browser may draw the rest later.

A page with islands loads one small module script, the loader (about 0.5 KB), which sets each island's trigger. The first island that starts fetches the runtime, Vue and the hydration, once for the page; every island then shares it and loads its own code. A page with an island that starts on `load` preloads the runtime with the loader; one whose islands wait (`idle`, `visible`, `media`) fetches nothing more until one starts.

## Before the first paint

`usePrePaint(id, code)` adds a few lines of script to the top of the `<head>`, on the pages that render the calling component. Use it only to copy a stored preference onto `<html>` for CSS to read, as `AppShell` does for its collapsed sidebar. The server's markup must not depend on it.

## Your Vite config

If the site has a `vite.config.ts`, the build uses it. It adds Vue, Weft's icons and [Markdown content](markdown.md) unless your config already has them.
