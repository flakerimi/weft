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

### Pages for records made after the build

A page for a record that is made later, such as `/products/42`, cannot list its paths at build. It exports `runtime = true` in place of `getPaths()`. The page is rendered once, at build, before any address exists, so it reads its params in an [island](#islands) with `useParams()`. Write `hello/pages/products/[id].vue`:

```vue
<script lang="ts">
export const runtime = true
</script>

<script setup vapor lang="ts">
import { Island } from '@weft/ui'
import Product from '../../components/Product.vue'
</script>

<template>
  <Island when="load" :component="Product" />
</template>
```

and `hello/components/Product.vue`, which reads the id once it is mounted, the place to fetch the record from the API:

```vue
<script setup vapor lang="ts">
import { onMounted, ref } from 'vue'
import { useParams } from '@weft/ui'

const id = ref('')
// In the browser only: at build the page has no address, and useParams() is {}.
onMounted(() => {
  id.value = useParams('/products/:id').id
})
</script>

<template>
  <p>Product {{ id }}</p>
</template>
```

`useParams()` runs in the browser only; on the server, on another page or on an address the pattern does not match it returns `{}`, so reading a property on it is `undefined` outside `onMounted` on the matching page. Passing the page's pattern (`useParams('/products/:id')`) types the returned object's parameter names from the runtime pages under `pages/` (written into `.weft-types/components.d.ts` by `weftAutoImport()` and `weft-types`); converting a string parameter to a number stays the page's job, as the example shop's `productId` does.

The build makes the page once, as a shell, with every param the empty string, and writes it to `pages/products/[id]/index.html`. The Go server answers every path that matches `/products/:id` with that shell and status 200, when no other page and no file has the path: `/products/new` from `pages/products/new.vue` wins. `[id]` matches one segment that is not empty, and `[...rest]` the rest of the path, or none of it. A record that does not exist is the API's 404, which the page shows; the page itself is always 200. `weft dev` serves the page the same way.

The build writes the pattern on the page's `<html>` as `data-weft-route="/products/:id"`. `useParams()` reads `location.pathname` against it and returns each param, or `{}` on another page. It splits the path as the server does, decoded first, so an id that holds an encoded slash (`%2F`) is two segments and cannot be one param. It is not reactive: the page loads once per address. Some things fail the build:

- a dynamic page that exports both `runtime` and `getPaths()`, or neither;
- two runtime pages that match the same paths, such as `a/[x].vue` and `a/[y].vue`;
- a runtime page whose shell another page also writes, such as a `getPaths()` value of `[id]`.

When two patterns match one path, the one with more fixed segments wins; then one without a `[...name]`; then the one with fewer params, except that of two with a `[...name]`, the one with more wins. So a `[...name]` never hides a more specific page.

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
- speculation rules, so the browser fetches a same-site link when the pointer rests on it. Links under `/api` are left out, and so is any link inside an element marked `data-weft-no-prefetch` (or marked so itself), such as a demo whose links name pages that do not exist. A dialog, sheet or menu opened from inside such an element carries the mark with it, so its links are left out too.
- `@view-transition { navigation: auto; }`, so going from page to page cross-fades in browsers that support it.

A page with no island has no script that loads hydration JavaScript (a page that renders a width-adaptive component, an `AppShell` with `persist`, a closable `Banner` with an `id`, or a `Form` carries one small inline `<script data-weft-prepaint>` in `<head>`).

Files in `public/` are copied to the root of the build as they are: `public/robots.txt` is `/robots.txt`.

### Icons

Put the site's icon in `public/` (in an app `weft create` made, `web/public/`) and that is all it takes: every page's `<head>`, the not-found page's included, names the files that are there, in this order, and nothing for those that are not.

| File | Tag |
| --- | --- |
| `favicon.ico` | `<link rel="icon" href="/favicon.ico" sizes="32x32">` |
| `favicon.svg` | `<link rel="icon" href="/favicon.svg" type="image/svg+xml">` |
| `apple-touch-icon.png` | `<link rel="apple-touch-icon" href="/apple-touch-icon.png">` |

A browser takes the last icon it can draw, so one that draws SVG shows `favicon.svg` and one that does not shows `favicon.ico`. The files are served with their types and revalidated, not cached for a year, since their names carry no hash. `weft dev` writes the same tags.

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
- **Its own CSS.** The page inlines the CSS of everything the island's file uses, not only what the server drew, since the browser may draw the rest later. That includes what it loads on demand with `import()`, such as a DatePicker's popover and calendar.

A page with islands loads one small module script, the loader (about 0.5 KB), which sets each island's trigger. The first island that starts fetches the runtime, Vue and the hydration, once for the page; every island then shares it and loads its own code. A page with an island that starts on `load` preloads the runtime with the loader; one whose islands wait (`idle`, `visible`, `media`) fetches nothing more until one starts.

## Components without imports

A page, an island or any component of the site may use Weft's components without importing them. The build adds the import for each one the template uses, so this page needs no `<script>` at all:

```vue
<template>
  <Page title="About">
    <Section><Card>Made with Weft.</Card></Section>
  </Page>
</template>
```

It gets `import { Card, Page, Section } from '@weft/ui'` at the top of its `<script setup>`, or a `<script setup vapor lang="ts">` to hold it, and builds as if you had written the line: the page inlines the same CSS, and an island loads the same code. Writing the import yourself still works, as every example here does.

Your own names come first. A tag gets no import when:

- the file imports or declares that name itself, in `<script setup>` or a plain `<script>` beside it, as `import Card from '../components/Card.vue'` or `const Card = …` does;
- it is the component's own name: `<Card>` inside `components/Card.vue` is the component itself, drawn recursively, as Vue has it (the page `pages/card.vue` showing `<Card>` gets Weft's);
- it is lowercase (`<button>`, `<card>`), or inside an element marked `v-pre`;
- it is not one of `@weft/ui`'s components.

A component with only a plain `<script>`, and no `<script setup>`, is left as it is.

For your editor and `vue-tsc`, the build writes `.weft-types/components.d.ts` in the site, which tells TypeScript what each name is and maps each runtime page's pattern in `WeftRouteParams`. Add it to the `include` of your `tsconfig.json`, and keep it out of git: it is rewritten on each build, whenever the list changes. In an app, `weft create`, `weft gen` and `weft build` write it too before any type check or build, with `weft-types --root web`, so `pnpm typecheck` and `weft build` check the components' props and `useParams(pattern)` from the start.

## Before the first paint

`usePrePaint(id, code)` adds a few lines of script to the top of the `<head>`, on the pages that render the calling component, each snippet wrapped in `(function(){…})()`. It is for three things: measuring width-adaptive components (`useWidth`) before the first paint and on resize without hydration, copying a stored preference onto `<html>` for CSS to read, as `AppShell` does for its collapsed sidebar, and stopping what the server's markup would do before an island is live, as `Form` does: it cancels every submit of a Weft form until its island takes over. The server's markup must not depend on it. A site with a Content-Security-Policy allows this inline script by its hash, or with `'unsafe-inline'`; [`Form`'s page](/components/form) and [`Grid`'s page](/components/grid) have their exact texts.

A `Form` sends only through its `@submit` handler, so it belongs in an island: one the page draws with no island around it never sends anything. What a person types, or a password manager fills, before the island is live is kept when it hydrates.

## Your Vite config

If the site has a `vite.config.ts`, the build uses it. It adds Vue, Weft's icons, [Markdown content](markdown.md) and the imports of Weft's components unless your config already has them. To write the declaration file elsewhere, or not at all, add `weftAutoImport({ dts: false })` from `@weft/vite` to your config's plugins.
