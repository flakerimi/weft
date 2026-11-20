import { execFile } from 'node:child_process'
import { createHash } from 'node:crypto'
import { existsSync, mkdirSync, mkdtempSync, readdirSync, readFileSync, rmSync, writeFileSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join, relative } from 'node:path'
import { fileURLToPath } from 'node:url'
import { promisify } from 'node:util'
import { brotliDecompressSync, gunzipSync } from 'node:zlib'
import { afterAll, beforeAll, describe, expect, it } from 'vitest'
import { buildSite, type Manifest } from '../src/build.ts'
import type { RoutesJson } from '../src/routes.ts'

const fixture = (n: string) => fileURLToPath(new URL(`./fixtures/${n}`, import.meta.url))
const temps: string[] = []
const tempDir = () => {
  const dir = mkdtempSync(join(tmpdir(), 'weft-build-'))
  temps.push(dir)
  return dir
}

afterAll(() => {
  for (const dir of temps) rmSync(dir, { recursive: true, force: true })
})

/** Every file under `dir`, as dist-relative posix paths. */
function files(dir: string): string[] {
  return readdirSync(dir, { recursive: true, withFileTypes: true })
    .filter((e) => e.isFile())
    .map((e) => relative(dir, join(e.parentPath, e.name)).split('\\').join('/'))
    .sort()
}

const scripts = (html: string) => [...html.matchAll(/<script\b([^>]*)>/g)].map((m) => m[1]!)
const typeOf = (attrs: string) => /\btype="([^"]*)"/.exec(attrs)?.[1]
const styleOf = (html: string) => /<style>([\s\S]*?)<\/style>/.exec(html)?.[1]
const etagOf = (bytes: Buffer) => `"${createHash('sha256').update(bytes).digest('hex').slice(0, 16)}"`

describe('buildSite on a site with islands', () => {
  let out: string
  let manifest: Manifest
  const read = (p: string) => readFileSync(join(out, p), 'utf8')

  beforeAll(async () => {
    out = tempDir()
    manifest = await buildSite({ root: fixture('site'), outDir: out })
  })

  it('writes one HTML file per route, dynamic routes expanded with their params as props', () => {
    for (const p of ['index', 'counter', 'escape', '404', 'blog/first', 'blog/second']) {
      expect(existsSync(join(out, 'pages', p, 'index.html')), p).toBe(true)
    }
    expect(read('pages/blog/first/index.html')).toContain('<h1>Post first</h1>')
    expect(read('pages/blog/second/index.html')).toContain('<h1>Post second</h1>')
  })

  it('ships no JavaScript on a page without islands', () => {
    const html = read('pages/index/index.html')
    expect(html).toContain('<h1>Hello from Weft</h1>')
    // Only JSON script types: no classic (type missing) and no module script.
    expect(scripts(html).map(typeOf)).toEqual(['speculationrules'])
    expect(html).not.toContain('modulepreload')
  })

  it('loads the island runtime with exactly one module script on a page with an island', () => {
    const html = read('pages/counter/index.html')
    const modules = scripts(html).filter((a) => typeOf(a) === 'module')
    expect(modules).toHaveLength(1)
    const src = /\bsrc="([^"]+)"/.exec(modules[0]!)?.[1]
    expect(src).toMatch(/^\/assets\/[\w-]+\.js$/)
    expect(existsSync(join(out, src!))).toBe(true)
    expect(manifest.files[src!]).toMatchObject({ type: 'text/javascript; charset=utf-8', immutable: true })
    // The island itself and its props still come from the server render.
    expect(html).toContain('<weft-island data-id="Counter" data-when="visible">')
  })

  it('builds a client chunk for each island the pages use', () => {
    const assets = files(out).filter((f) => f.startsWith('assets/') && f.endsWith('.js'))
    expect(assets.some((f) => /^assets\/Counter-[\w-]+\.js$/.test(f))).toBe(true)
  })

  it('inlines the theme and weft.css with view transitions, and adds speculation rules', () => {
    for (const p of ['index', 'counter']) {
      const html = read(`pages/${p}/index.html`)
      const css = styleOf(html)!
      expect(css).toContain('--weft-color-bg: #FAFAFA') // neutral, the default theme
      expect(css).toContain('box-sizing:border-box') // weft.css
      expect(css).toContain('@view-transition { navigation: auto; }')
      expect(html).not.toContain('rel="stylesheet"')
      expect(html).toContain(
        '<script type="speculationrules">{"prefetch":[{"where":{"href_matches":"/*"},"eagerness":"moderate"}]}</script>',
      )
    }
  })

  it('writes .br and .gz siblings of every text file that decompress to its bytes', () => {
    const all = files(out)
    const text = all.filter((f) => /\.(html|css|js|json|svg|txt|xml)$/.test(f) && f !== 'manifest.json')
    expect(text.filter((f) => f.endsWith('.html'))).toHaveLength(6)
    for (const f of text) {
      const bytes = readFileSync(join(out, f))
      expect(brotliDecompressSync(readFileSync(join(out, f + '.br'))).equals(bytes), f).toBe(true)
      expect(gunzipSync(readFileSync(join(out, f + '.gz'))).equals(bytes), f).toBe(true)
    }
  })

  it('lists every served file in manifest.json with the hash of its bytes as ETag', () => {
    const written = JSON.parse(read('manifest.json')) as Manifest
    expect(written).toEqual(manifest)
    const served = files(out).filter((f) => !/\.(br|gz)$/.test(f) && f !== 'manifest.json')
    expect(Object.keys(manifest.files).sort()).toEqual(served.map((f) => '/' + f))
    for (const [path, entry] of Object.entries(manifest.files)) {
      expect(entry.etag, path).toBe(etagOf(readFileSync(join(out, path))))
    }
    expect(manifest.files['/pages/index/index.html']).toEqual({
      etag: etagOf(readFileSync(join(out, 'pages/index/index.html'))),
      type: 'text/html; charset=utf-8',
      br: true,
      gz: true,
      immutable: false,
    })
    expect(manifest.files['/routes.json']).toMatchObject({ type: 'application/json', immutable: false })
  })

  it('hints the runtime modules for island pages only', () => {
    const entry = /<script type="module" src="([^"]+)"/.exec(read('pages/counter/index.html'))![1]
    expect(manifest.hints['/counter']).toContain(`<${entry}>; rel=modulepreload`)
    expect(manifest.hints['/']).toBeUndefined()
  })

  it('writes routes.json in the route-map shape', () => {
    const routes = JSON.parse(read('routes.json')) as RoutesJson
    expect(routes).toEqual({
      pages: [
        { path: '/', html: 'pages/index/index.html', islands: [] },
        { path: '/counter', html: 'pages/counter/index.html', islands: ['Counter'] },
        { path: '/escape', html: 'pages/escape/index.html', islands: ['Counter'] },
        { path: '/blog/first', html: 'pages/blog/first/index.html', islands: [] },
        { path: '/blog/second', html: 'pages/blog/second/index.html', islands: [] },
      ],
      notFound: 'pages/404/index.html',
      app: null,
    })
  })
})

describe('buildSite on a site without islands', () => {
  let out: string

  beforeAll(async () => {
    out = tempDir()
    // An app built before the pages: its files join the manifest.
    mkdirSync(join(out, 'app'))
    writeFileSync(join(out, 'app/index.html'), '<!doctype html><div id="app"></div>')
    await buildSite({ root: fixture('static-site'), outDir: out, appPrefix: '/app', theme: 'lisa' })
  })

  it("uses the site's own vite.config", () => {
    expect(readFileSync(join(out, 'pages/index/index.html'), 'utf8')).toContain('<h1>Static from vite.config</h1>')
  })

  it('makes no client build and no runtime script', () => {
    expect(existsSync(join(out, 'assets'))).toBe(false)
    const html = readFileSync(join(out, 'pages/index/index.html'), 'utf8')
    expect(scripts(html).map(typeOf)).toEqual(['speculationrules'])
  })

  it('inlines the chosen theme', () => {
    const css = styleOf(readFileSync(join(out, 'pages/index/index.html'), 'utf8'))!
    expect(css).toContain('--weft-color-bg: #F1F5F9') // lisa
  })

  it('records the app and a missing 404 page in routes.json', () => {
    const routes = JSON.parse(readFileSync(join(out, 'routes.json'), 'utf8')) as RoutesJson
    expect(routes).toEqual({
      pages: [{ path: '/', html: 'pages/index/index.html', islands: [] }],
      notFound: null,
      app: { prefix: '/app', index: 'app/index.html' },
    })
    const manifest = JSON.parse(readFileSync(join(out, 'manifest.json'), 'utf8')) as Manifest
    expect(manifest.files['/app/index.html']).toMatchObject({ type: 'text/html; charset=utf-8', br: true, gz: true })
    expect(existsSync(join(out, 'app/index.html.br'))).toBe(true)
  })
})

describe('buildSite with CSS over the inline limit', () => {
  it('links one hashed stylesheet that keeps view transitions, and hints it', async () => {
    const out = tempDir()
    const manifest = await buildSite({ root: fixture('static-site'), outDir: out, cssInlineLimit: 100 })
    const html = readFileSync(join(out, 'pages/index/index.html'), 'utf8')
    expect(html).not.toContain('<style>')
    const href = /<link rel="stylesheet" href="([^"]+)">/.exec(html)?.[1]
    expect(href).toMatch(/^\/assets\/weft-[0-9a-f]{8}\.css$/)
    const css = readFileSync(join(out, href!), 'utf8')
    expect(css).toContain('--weft-color-bg: #FAFAFA')
    expect(css).toContain('@view-transition { navigation: auto; }')
    expect(manifest.files[href!]).toMatchObject({ type: 'text/css; charset=utf-8', immutable: true })
    expect(manifest.hints['/']).toEqual([`<${href}>; rel=preload; as=style`])
  })
})

describe('bin/weft-build-pages.mjs', () => {
  it('makes a production build from the command line', async () => {
    const out = tempDir()
    const bin = fileURLToPath(new URL('../bin/weft-build-pages.mjs', import.meta.url))
    // As `weft build` runs it: no NODE_ENV (vitest sets "test").
    const env = { ...process.env, NODE_ENV: undefined }
    const { stdout } = await promisify(execFile)(process.execPath, [bin, '--root', fixture('site'), '--out', out], { env })
    expect(stdout).toBe(`weft: built 6 pages into ${out}\n`)
    const js = files(out).filter((f) => f.endsWith('.js'))
    expect(js.length).toBeGreaterThan(0)
    // Vue's development build carries its warnings; the production build does not.
    for (const f of js) expect(readFileSync(join(out, f), 'utf8').includes('[Vue warn]'), f).toBe(false)
  })

  it('fails with one sentence when an argument is missing', async () => {
    const bin = fileURLToPath(new URL('../bin/weft-build-pages.mjs', import.meta.url))
    await expect(promisify(execFile)(process.execPath, [bin, '--root', fixture('static-site')])).rejects.toMatchObject({
      code: 1,
      stderr: 'weft-build-pages: --out is required.\nusage: weft-build-pages --root <site> --out <dist> [--app-prefix /app] [--theme neutral|lisa]\n',
    })
  })
})
