---
title: Your first page
description: Add a page to the app, see it under weft dev, build it, and see what the build writes.
order: 12
---

On this page you add a page to the app you made in [Create an app](create-an-app.md), see it while `weft dev` runs, build it, and look at what the build writes. Run every command in the app's folder, `myapp`.

## Write a page

Each `.vue` file in `web/pages/` is a page, and its path is its route: `web/pages/index.vue` is `/`, and `web/pages/about.vue` is `/about`. Write `web/pages/about.vue`:

```vue
<script setup vapor lang="ts">
import { useHead } from '@weft/ui'

useHead({ title: 'About', description: 'What this app is for.' })

const notes = [
  { title: 'Built ahead', text: 'This page is rendered when the app is built.' },
  { title: 'No script', text: 'It has no island, so it loads no JavaScript.' },
]
</script>

<template>
  <div data-weft-surface="landing">
    <Page title="About" description="What this app is for.">
      <Section title="What you see">
        <Grid>
          <Card v-for="note in notes" :key="note.title">
            <template #header>{{ note.title }}</template>
            {{ note.text }}
          </Card>
        </Grid>
      </Section>
    </Page>
  </div>
</template>
```

- `vapor` on the script makes it a Vapor component. Weft's components are all Vapor; write yours the same way.
- `Page`, `Section`, `Grid` and `Card` are Weft's components, which need no import in a `.vue` file. `useHead` is a function, so it is imported from `@weft/ui`; it sets the page's title and description.
- `data-weft-surface="landing"` gives the page the landing surface: larger text and a capped width. See [the adaptive model](../guides/adaptive.md).

## See it

```sh
weft dev
```

Open `http://localhost:5173/about`. The page's title is `About`, and it shows the two cards. `weft dev` renders each page when it is asked for, so a page you add or save shows on the next request, with no restart. A path with no page, such as `/nope`, gets `web/pages/404.vue` with status 404.

Ctrl-C stops `weft dev`.

## Build it

```sh
weft build
```

It prints `weft: built 3 pages into web/dist`, for `index`, `about` and `404`, then `Built myapp`.

## What the build writes

`web/dist/` now holds, among the rest:

| File | What it is |
| --- | --- |
| `pages/about/index.html` | The page, rendered, with its CSS inlined |
| `pages/about/index.html.br`, `.gz` | The same page, compressed with brotli and gzip |
| `assets/` | The scripts of the islands, such as `Hello` on the front page, and their compressed copies |
| `routes.json` | Each page's path and file |
| `manifest.json` | Every file to serve, with its type and ETag |
| `.weft-build` | The list of files the build wrote, so the next build can replace them safely |

The page has the theme, Weft's base styles and the CSS of the components it renders, and nothing else. It has no island, so it has no script tag that loads JavaScript. The binary embeds `web/dist`, and `./myapp` serves the page at `/about`. [Pages and islands](../guides/pages-and-islands.md) says more about pages.

Next, [add a Go route](first-go-handler.md) that the pages can call.
