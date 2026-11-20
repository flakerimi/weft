// Every @weft/ui example renders on the server through the page pipeline:
// each component's server render, with its icons drawn, under Node, where
// there is no window or document. Later components join by adding examples.
import { readdirSync } from 'node:fs'
import { createRequire } from 'node:module'
import { dirname, join } from 'node:path'
import type { ViteDevServer } from 'vite'
import { afterAll, beforeAll, describe, expect, it } from 'vitest'
import { createRenderServer, renderPage } from '../src/render.ts'

// The package root of @weft/ui: its main entry is src/index.ts.
const ui = dirname(dirname(createRequire(import.meta.url).resolve('@weft/ui')))
const examples = readdirSync(join(ui, 'examples'), { recursive: true, encoding: 'utf8' })
  .map((f) => f.split('\\').join('/'))
  .filter((f) => /(^|\/)[a-z0-9-]+\.vue$/.test(f))
  .sort()

let server: ViteDevServer
beforeAll(async () => {
  server = await createRenderServer(ui)
}, 60_000)
afterAll(() => server.close())

describe('@weft/ui examples', () => {
  it('include a basic example for the layout components', () => {
    expect(examples).toEqual(expect.arrayContaining(['Card/basic.vue', 'Stack/basic.vue', 'Icon/basic.vue']))
  })

  it.each(examples)('%s renders on the server', async (file) => {
    expect(typeof window).toBe('undefined')
    const { html } = await renderPage({ root: ui, page: `examples/${file}`, server })
    expect(html.trim()).not.toBe('')
    // Only JSON scripts (island props) may appear in server HTML.
    for (const m of html.matchAll(/<script\b([^>]*)>/g)) expect(m[1]).toContain('type="application/json"')
    // Every icon is drawn, none blank.
    for (const m of html.matchAll(/<svg class="weft-icon"[^>]*>([\s\S]*?)<\/svg>/g)) expect(m[1]).not.toBe('')
  })
})
