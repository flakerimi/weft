// Render every @weft/ui example inside an <Island>, on the server, through
// renderPage, and print { "<Name>/<example>.vue": "<weft-island …>…" } as JSON
// on stdout. examples-hydrate.test.ts hydrates each in happy-dom.
// usage: node render-example-islands.ts
//
// It runs in a process of its own, plain Node with no window or document, as
// the page build does: the test that hydrates runs in happy-dom, whose
// globals a server render must never see.
//
// Each example's page is a module this script serves (no file is written):
// `examples/<Name>/<example>.island.vue`, an Island around the example.
import { readdirSync } from 'node:fs'
import { createRequire } from 'node:module'
import { dirname, join } from 'node:path'
import { createServer, type Plugin } from 'vite'
import { siteConfig } from '../src/config.ts'
import { renderPage } from '../src/render.ts'

const ui = dirname(dirname(createRequire(import.meta.url).resolve('@weft/ui')))
const examples = readdirSync(join(ui, 'examples'), { recursive: true, encoding: 'utf8' })
  .map((f) => f.split('\\').join('/'))
  .filter((f) => /(^|\/)[a-z0-9-]+\.vue$/.test(f))
  .sort()

const ISLAND = /\.island\.vue$/
const pages: Plugin = {
  name: 'weft-test:example-islands',
  enforce: 'pre',
  resolveId(id) {
    if (ISLAND.test(id)) return join(ui, id)
  },
  load(id) {
    if (!ISLAND.test(id)) return
    const example = id.replace(ISLAND, '.vue')
    return `<script setup vapor lang="ts">
import { Island } from '@weft/ui'
import Example from ${JSON.stringify(example)}
</script>

<template>
  <Island when="load" :component="Example" />
</template>
`
  },
}

// As createRenderServer makes it, with the pages above.
const server = await createServer(
  await siteConfig(ui, { command: 'serve', mode: 'development' }, {
    appType: 'custom',
    logLevel: 'error',
    server: { middlewareMode: true, ws: false },
    optimizeDeps: { noDiscovery: true, include: [] },
    ssr: { noExternal: ['@weft/ui'] },
    plugins: [pages],
  }),
)
const out: Record<string, string> = {}
try {
  for (const file of examples) {
    const { html } = await renderPage({ root: ui, page: `examples/${file.replace(/\.vue$/, '.island.vue')}`, server })
    out[file] = html
  }
} finally {
  await server.close()
}
process.stdout.write(JSON.stringify(out) + '\n')
