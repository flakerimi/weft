// A component may register a pre-paint snippet (usePrePaint): a few lines of
// script the page build writes, inline and blocking, at the top of <head>,
// on the pages that render that component and no others. AppShell uses it to
// copy its stored collapsed state to <html> before the first paint.
import { mkdtempSync, readFileSync, rmSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { afterAll, beforeAll, describe, expect, it } from 'vitest'
import { buildSite } from '../src/build.ts'
import { renderPage } from '../src/render.ts'

const root = fileURLToPath(new URL('./fixtures/prepaint-site', import.meta.url))
let out: string
const page = (name: string) => readFileSync(join(out, 'pages', name, 'index.html'), 'utf8')
const head = (name: string) => page(name).slice(0, page(name).indexOf('</head>'))
const snippets = (html: string) => [...html.matchAll(/<script data-weft-prepaint>([\s\S]*?)<\/script>/g)].map((m) => m[1]!)

beforeAll(async () => {
  out = mkdtempSync(join(tmpdir(), 'weft-prepaint-'))
  await buildSite({ root, outDir: join(out, 'dist') })
  out = join(out, 'dist')
}, 120_000)
afterAll(() => rmSync(join(out, '..'), { recursive: true, force: true }))

describe('pre-paint snippets', () => {
  it('are written once, at the top of <head> before the CSS, on a page that renders an AppShell', () => {
    const h = head('shell')
    expect(snippets(h)).toHaveLength(1)
    expect(h.indexOf('data-weft-prepaint')).toBeLessThan(h.indexOf('<style>'))
    expect(snippets(h)[0]).toContain('"weft:app-shell:collapsed"')
  })

  it('are left out of a page that renders none, and of an AppShell that keeps nothing', () => {
    expect(page('plain')).not.toContain('data-weft-prepaint')
    expect(page('unkept')).not.toContain('data-weft-prepaint')
  })

  it('come from an AppShell inside an island too, under its own key', () => {
    expect(snippets(head('island'))[0]).toContain('"acme:sidebar"')
  })

  it('mark the root before paint when the sidebar was left collapsed, and survive storage that throws', () => {
    const code = snippets(head('shell'))[0]!
    const run = (getItem: () => string | null) => {
      const dataset: Record<string, string> = {}
      new Function('localStorage', 'document', code)({ getItem }, { documentElement: { dataset } })
      return dataset.weftSidebar
    }
    expect(run(() => 'true')).toBe('collapsed')
    expect(run(() => 'false')).toBeUndefined()
    expect(run(() => null)).toBeUndefined()
    expect(run(() => { throw new Error('denied') })).toBeUndefined()
  })

  it('reach renderPage as prePaint, and nothing else of the page changes', async () => {
    const { html, prePaint } = await renderPage({ root, page: 'pages/shell.vue' })
    expect(prePaint).toHaveLength(1)
    expect(html).not.toContain('<script')
  })
})
