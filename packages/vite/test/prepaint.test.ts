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
import { prePaintTag } from '../src/document.ts'
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
  it('are written once, wrapped in their own function scope at the top of <head> before the CSS, on a page that renders an AppShell', () => {
    const h = head('shell')
    expect(snippets(h)).toHaveLength(1)
    expect(h.indexOf('data-weft-prepaint')).toBeLessThan(h.indexOf('<style>'))
    expect(snippets(h)[0]).toMatch(/^\(function\(\)\{[\s\S]*\}\)\(\)$/)
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

  it('come from a closable Banner with an id, and from no other Banner', () => {
    const h = head('banner')
    expect(snippets(h)).toHaveLength(1)
    expect(h.indexOf('data-weft-prepaint')).toBeLessThan(h.indexOf('<style>'))
    expect(snippets(h)[0]).toContain('weft-banner:news-1')
    expect(page('banner-unkept')).not.toContain('data-weft-prepaint')
  })

  it('hide a Banner closed on an earlier visit before paint, and survive storage that throws', () => {
    const code = snippets(head('banner'))[0]!
    const run = (getItem: (key: string) => string | null) => {
      const added: { textContent: string }[] = []
      const document = {
        createElement: () => ({ textContent: '' }),
        head: { appendChild: (el: { textContent: string }) => added.push(el) },
      }
      new Function('localStorage', 'document', code)({ getItem }, document)
      return added.map((s) => s.textContent)
    }
    expect(run((k) => (k === 'weft-banner:news-1' ? 'closed' : null))).toEqual(['.weft-banner[data-id="news-1"]{display:none}'])
    expect(run(() => null)).toEqual([])
    expect(run(() => { throw new Error('denied') })).toEqual([])
  })

  it('reach renderPage as prePaint, and nothing else of the page changes', async () => {
    const { html, prePaint } = await renderPage({ root, page: 'pages/shell.vue' })
    expect(prePaint).toHaveLength(1)
    expect(html).not.toContain('<script')
  })

  it('N8: separates multiple wrapped snippets with ";\\n" so every snippet runs in order without throwing', () => {
    const tag = prePaintTag(['var x=1;ran.push("a",x)', 'var x=2;ran.push("b",x)', 'var x=3;ran.push("c",x)'])[0]!
    const code = tag.replace(/^<script data-weft-prepaint>|<\/script>$/g, '')
    expect(code).toBe('(function(){var x=1;ran.push("a",x)})();\n(function(){var x=2;ran.push("b",x)})();\n(function(){var x=3;ran.push("c",x)})()')
    const ran: (string | number)[] = []
    new Function('ran', code)(ran)
    expect(ran).toEqual(['a', 1, 'b', 2, 'c', 3])
  })
})
