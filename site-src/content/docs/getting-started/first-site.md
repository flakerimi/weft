---
title: Your first site
description: Make a site with one page, build it, and see what the build writes.
order: 12
---

On this page you make a site with one page, build it, and look at what the build writes. You need the repository set up as [Install](install.md) describes. Run every command from the repository's root.

## Make the site

A Weft site is a folder with a `package.json` and a `pages/` folder. Make one called `hello`:

```sh
mkdir -p hello/pages
```

Add it to the workspace, at the end of `pnpm-workspace.yaml`:

```yaml
packages:
  - packages/*
  - playground/web
  - site-src
  - bench/weft
  - hello
```

Write `hello/package.json`:

```json
{
  "name": "hello",
  "private": true,
  "type": "module",
  "scripts": {
    "build": "weft-build-pages --root . --out dist"
  },
  "dependencies": {
    "@weft/ui": "workspace:*",
    "vue": "3.6.0-rc.16"
  },
  "devDependencies": {
    "@vitejs/plugin-vue": "^6.0.11",
    "@weft/vite": "workspace:*",
    "vite": "^8.3.8"
  }
}
```

`weft-build-pages` comes from `@weft/vite`. Vue is pinned to the release candidate Weft is built on.

## Write a page

Each `.vue` file in `pages/` is a page. `pages/index.vue` is `/`. Write `hello/pages/index.vue`:

```vue
<script setup vapor lang="ts">
import { Card, Grid, Page, Section, useHead } from '@weft/ui'

useHead({ title: 'Hello', description: 'My first Weft site.' })

const notes = [
  { title: 'Built ahead', text: 'This page was rendered when you built it.' },
  { title: 'No script', text: 'It has no island, so it loads no JavaScript.' },
]
</script>

<template>
  <div data-weft-surface="landing">
    <Page title="Hello" description="My first Weft site.">
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
- `useHead` sets the page's title and description.
- `data-weft-surface="landing"` gives the page the landing surface: larger text and a capped width. See [the adaptive model](../guides/adaptive.md).

## Build it

Install, so pnpm links the new site, then build:

```sh
pnpm install
pnpm --filter hello build
```

It prints `weft: built 1 page into dist`.

## What the build writes

`hello/dist/` now holds:

| File | What it is |
| --- | --- |
| `pages/index/index.html` | The page, rendered, with its CSS inlined |
| `pages/index/index.html.br`, `.gz` | The same page, compressed with brotli and gzip |
| `routes.json` | Each page's path and file |
| `manifest.json` | Every file to serve, with its type and ETag |
| `.weft-build` | The list of files the build wrote, so the next build can replace them safely |

The page has the theme, Weft's base styles and the CSS of the components it renders, and nothing else. It has no island, so it has no script tag that loads JavaScript.

Nothing serves `dist/` yet. Next, [add a Go handler](first-go-handler.md) that does.
