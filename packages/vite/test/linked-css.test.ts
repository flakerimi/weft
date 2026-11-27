// A stylesheet a page links by its URL (`import href from './x.css?url'`),
// rather than importing it for the page's own CSS: the page build writes it
// as a file of its own under /assets/, minified, named by a hash of its
// bytes and cached for good, with the files it names copied in as a page's
// are, and gives the server render that URL; the dev server serves it as
// Vite does. It is never inlined, and its fonts are never preloaded: it is
// for CSS a page fetches only when it needs it (a theme a visitor picks).
import { createHash } from 'node:crypto'
import { cpSync, existsSync, mkdirSync, mkdtempSync, readFileSync, rmSync, symlinkSync, writeFileSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { afterAll, beforeAll, describe, expect, it, vi } from 'vitest'
import { buildSite, type Manifest } from '../src/build.ts'
import { devSite } from '../src/dev.ts'

const fixture = fileURLToPath(new URL('./fixtures/css-linked', import.meta.url))
const temps: string[] = []
const tempDir = () => {
  const dir = mkdtempSync(join(tmpdir(), 'weft-linked-'))
  temps.push(dir)
  return dir
}
afterAll(() => {
  for (const dir of temps) rmSync(dir, { recursive: true, force: true })
})

/** A copy of the fixture that resolves packages through this package's node_modules. */
function copy(): string {
  const root = join(tempDir(), 'css-linked')
  cpSync(fixture, root, { recursive: true })
  symlinkSync(fileURLToPath(new URL('../node_modules', import.meta.url)), join(root, 'node_modules'))
  return root
}

const sha = (bytes: Buffer | string) => createHash('sha256').update(bytes).digest('hex').slice(0, 8)
const hrefOf = (html: string, id: string) => new RegExp(`<a id="${id}" href="([^"]+)"`).exec(html)?.[1]
const styleOf = (html: string) => /<style>([\s\S]*?)<\/style>/.exec(html)?.[1] ?? ''

describe('buildSite with a stylesheet a page links by its URL', () => {
  let out: string
  let manifest: Manifest
  const page = (name: string) => readFileSync(join(out, 'pages', name, 'index.html'), 'utf8')

  beforeAll(async () => {
    out = join(tempDir(), 'dist')
    manifest = await buildSite({ root: fixture, outDir: out })
  }, 60_000)

  it('writes it under /assets/, minified, named by a hash of its bytes, cached for good', () => {
    const href = hrefOf(page('index'), 'linked')!
    expect(href).toMatch(/^\/assets\/linked-[0-9a-f]{8}\.css$/)
    const css = readFileSync(join(out, href.slice(1)), 'utf8')
    expect(href).toBe(`/assets/linked-${sha(css)}.css`)
    expect(css).toContain(':root[data-linked] .linked{')
    expect(css).not.toContain('/*')
    expect(manifest.files[href]).toMatchObject({ type: 'text/css; charset=utf-8', immutable: true, br: true, gz: true })
  })

  it('copies the files it names in, under hashed names, and points it there', () => {
    const css = readFileSync(join(out, hrefOf(page('index'), 'linked')!.slice(1)), 'utf8')
    const font = `/assets/Linked-${sha(readFileSync(join(fixture, 'styles/fonts/Linked.woff2')))}.woff2`
    expect(css).toContain(`url(${font})`)
    expect(css).toMatch(/url\(\/assets\/lucide-[0-9a-f]{8}\.woff2\)/)
    expect(css).not.toContain('/@fs/')
    expect(readFileSync(join(out, font.slice(1)))).toEqual(readFileSync(join(fixture, 'styles/fonts/Linked.woff2')))
    expect(manifest.files[font]).toMatchObject({ type: 'font/woff2', immutable: true })
  })

  it('neither inlines it in the page nor preloads its fonts', () => {
    const html = page('index')
    expect(styleOf(html)).toContain('.page{color:green}')
    expect(styleOf(html)).not.toContain('rebeccapurple')
    expect(styleOf(html)).not.toContain('Linked')
    expect(html).not.toContain('rel="preload"')
    expect(manifest.hints['/'] ?? []).toEqual([])
  })

  it('gives the server render one URL, on every page and in pre-paint code', () => {
    const href = hrefOf(page('index'), 'linked')!
    expect(hrefOf(page('other'), 'linked')).toBe(href)
    expect(page('index')).toContain(`document.documentElement.dataset.linked=${JSON.stringify(href)}`)
    expect(Object.keys(manifest.files).filter((f) => /^\/assets\/linked-.*\.css$/.test(f))).toEqual([href])
  })

  it('builds a stylesheet a plugin makes, which answers every query on its file', () => {
    const href = hrefOf(page('index'), 'generated')!
    expect(href).toMatch(/^\/assets\/sheet-[0-9a-f]{8}\.css$/)
    expect(readFileSync(join(out, href.slice(1)), 'utf8')).toBe('.generated{color:#00f}\n')
    expect(styleOf(page('index'))).not.toContain('.generated')
  })
})

describe('buildSite refuses a stylesheet an island links by its URL', () => {
  it('in a sentence that names the file and says what to do', async () => {
    const root = copy()
    mkdirSync(join(root, 'components'))
    writeFileSync(
      join(root, 'components/Picker.vue'),
      `<script setup vapor lang="ts">\nimport linked from '../styles/linked.css?url'\n</script>\n\n<template><a :href="linked">Picker</a></template>\n`,
    )
    writeFileSync(
      join(root, 'pages/island.vue'),
      `<script setup vapor lang="ts">\nimport { Island } from '@weft/ui'\nimport Picker from '../components/Picker.vue'\n</script>\n\n<template><main><Island when="load" :component="Picker" /></main></template>\n`,
    )
    const out = join(tempDir(), 'dist')
    const error = vi.spyOn(console, 'error').mockImplementation(() => {})
    const warn = vi.spyOn(console, 'warn').mockImplementation(() => {})
    const message = await buildSite({ root, outDir: out }).then(
      () => '(built)',
      (e: Error) => e.message,
    )
    error.mockRestore()
    warn.mockRestore()
    expect(message).toContain(
      'buildSite: an island imports styles/linked.css?url, a stylesheet the browser would get no URL for; import it in a page or a component only the server renders, and give the island its URL as a prop.',
    )
    expect(existsSync(out)).toBe(false)
  }, 60_000)
})

describe('weft dev with a stylesheet a page links by its URL', () => {
  it('serves it as CSS at the URL the page has', async () => {
    const dev = await devSite({ root: copy(), port: 0 })
    try {
      const html = await (await fetch(dev.url)).text()
      const href = hrefOf(html, 'linked')!
      expect(href).toBeDefined()
      expect(styleOf(html)).not.toContain('rebeccapurple')
      const res = await fetch(new URL(href, dev.url), { headers: { accept: 'text/css,*/*;q=0.1', 'sec-fetch-dest': 'style' } })
      expect(res.status).toBe(200)
      expect(res.headers.get('content-type')).toMatch(/^text\/css/)
      const css = await res.text()
      expect(css).toContain('rebeccapurple')
      expect(css).toContain('Linked.woff2')
    } finally {
      await dev.close()
    }
  }, 60_000)
})
