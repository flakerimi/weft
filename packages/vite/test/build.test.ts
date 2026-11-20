import { execFile } from 'node:child_process'
import { createHash } from 'node:crypto'
import { cpSync, existsSync, mkdirSync, renameSync, symlinkSync, mkdtempSync, readdirSync, readFileSync, rmSync, writeFileSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join, relative } from 'node:path'
import { fileURLToPath } from 'node:url'
import { promisify } from 'node:util'
import { brotliDecompressSync, gunzipSync } from 'node:zlib'
import { afterAll, beforeAll, describe, expect, it } from 'vitest'
import { buildSite, contentType, type Manifest } from '../src/build.ts'
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
    for (const p of ['index', 'about', 'assets', 'counter', 'escape', '404', 'blog/first', 'blog/second']) {
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

  it('builds a client chunk for each island the pages use, and nothing from the site config input', () => {
    const assets = files(out).filter((f) => f.startsWith('assets/') && f.endsWith('.js'))
    expect(assets.some((f) => /^assets\/Counter-[\w-]+\.js$/.test(f))).toBe(true)
    // The fixture's vite.config sets input { main: 'index.html' }; only the client entry is built.
    expect(assets.filter((f) => /^assets\/main-/.test(f))).toEqual([])
    expect(existsSync(join(out, 'index.html'))).toBe(false)
  })

  it("keeps the site config's other build options, such as an external", () => {
    const lazy = files(out).find((f) => /^assets\/Lazy-[\w-]+\.js$/.test(f))!
    expect(read(lazy)).toMatch(/import\(\s*["'`]weft-test-external["'`]\s*\)/)
  })

  it('ships no island CSS from the client build: it is inlined in the pages', () => {
    expect(files(out).filter((f) => f.endsWith('.css'))).toEqual([])
    for (const f of files(out).filter((f) => f.endsWith('.js'))) {
      expect(read(f), f).not.toMatch(/[\w-]+-[\w-]{8}\.css/)
    }
  })

  it('inlines a stylesheet a page imports itself only once', () => {
    const about = styleOf(read('pages/about/index.html'))!
    expect(about.split('box-sizing:border-box').length - 1).toBe(1)
  })

  it('marks only hashed build output immutable, not a page named assets', () => {
    expect(manifest.files['/pages/assets/index.html']).toMatchObject({ immutable: false })
    const immutable = Object.keys(manifest.files).filter((f) => manifest.files[f]!.immutable)
    expect(immutable.length).toBeGreaterThan(0)
    for (const f of immutable) expect(f).toMatch(/^\/assets\/[\w-]+-[\w-]{8}\.(js|css)$/)
  })

  it("inlines the CSS of the components each page renders, islands' included", () => {
    const counter = styleOf(read('pages/counter/index.html'))!
    const about = styleOf(read('pages/about/index.html'))!
    const index = styleOf(read('pages/index/index.html'))!
    expect(counter).toMatch(/button \{\s*font-variant-numeric: tabular-nums;\s*\}/)
    expect(about).toMatch(/\.badge \{\s*letter-spacing: 0\.02em;\s*\}/)
    expect(index).not.toContain('tabular-nums')
    expect(index).not.toContain('.badge')
    // Component CSS comes before view transitions, after the theme and weft.css.
    expect(about.indexOf('.badge')).toBeGreaterThan(about.indexOf('box-sizing:border-box'))
  })

  it('writes the head a page sets with useHead, escaped', () => {
    const html = read('pages/about/index.html')
    const head = html.slice(0, html.indexOf('</head>'))
    expect(head).toContain('<title>About &lt;Weft&gt;</title>')
    expect(head).toContain('<meta name="description" content="A &quot;quoted&quot; &amp; &lt;b&gt;page&lt;/b&gt;">')
    expect(head).toContain('<meta property="og:title" content="About &lt;Weft&gt;">')
    expect(head).toContain('<meta property="og:description" content="A &quot;quoted&quot; &amp; &lt;b&gt;page&lt;/b&gt;">')
    expect(head).toContain('<meta property="og:image" content="/og.png?a=1&amp;b=2">')
  })

  it('writes no title or description for a page without useHead', () => {
    const html = read('pages/index/index.html')
    expect(html).not.toContain('<title')
    expect(html).not.toContain('name="description"')
    expect(html).not.toContain('property="og:')
  })

  it("copies the site's public files to the root of the build, in the manifest", () => {
    expect(read('robots.txt')).toBe('User-agent: *\nAllow: /\n')
    expect(manifest.files['/robots.txt']).toMatchObject({ type: 'text/plain; charset=utf-8', br: true, gz: true, immutable: false })
    expect(manifest.files['/icon.bin']).toMatchObject({ type: 'application/octet-stream', br: false, gz: false, immutable: false })
    expect(existsSync(join(out, 'icon.bin.br'))).toBe(false)
    // A public file named like a compressed variant is copied like any other.
    expect(readFileSync(join(out, 'archive.tar.gz'))).toEqual(readFileSync(fixture('site/public/archive.tar.gz')))
    expect(manifest.files['/archive.tar.gz']).toMatchObject({ type: 'application/octet-stream', br: false, gz: false })
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
    expect(text.filter((f) => f.endsWith('.html'))).toHaveLength(8)
    for (const f of text) {
      const bytes = readFileSync(join(out, f))
      expect(brotliDecompressSync(readFileSync(join(out, f + '.br'))).equals(bytes), f).toBe(true)
      expect(gunzipSync(readFileSync(join(out, f + '.gz'))).equals(bytes), f).toBe(true)
    }
  })

  it('lists every served file in manifest.json with the hash of its bytes as ETag', () => {
    const written = JSON.parse(read('manifest.json')) as Manifest
    expect(written).toEqual(manifest)
    const all = files(out)
    // Leave out compressed variants (a file named like one, from public/, is served).
    const served = all.filter((f) => !(/\.(br|gz)$/.test(f) && all.includes(f.slice(0, -3))) && f !== 'manifest.json')
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
        { path: '/about', html: 'pages/about/index.html', islands: ['Lazy'] },
        { path: '/assets', html: 'pages/assets/index.html', islands: [] },
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
    // Files the owner put there: no manifest lists them, so the build leaves them.
    mkdirSync(join(out, 'pages/old'), { recursive: true })
    writeFileSync(join(out, 'pages/old/index.html'), 'owner')
    writeFileSync(join(out, 'notes.txt'), 'owner')
    await buildSite({ root: fixture('static-site'), outDir: out, appPrefix: '/app', theme: 'lisa' })
  })

  it("uses the site's own vite.config", () => {
    expect(readFileSync(join(out, 'pages/index/index.html'), 'utf8')).toContain('<h1>Static from vite.config</h1>')
  })

  it('leaves the files it did not write, and the app build', () => {
    expect(readFileSync(join(out, 'pages/old/index.html'), 'utf8')).toBe('owner')
    expect(readFileSync(join(out, 'notes.txt'), 'utf8')).toBe('owner')
    expect(readFileSync(join(out, 'app/index.html'), 'utf8')).toBe('<!doctype html><div id="app"></div>')
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
  let out: string
  let manifest: Manifest
  const hrefOf = (page: string) =>
    /<link rel="stylesheet" href="([^"]+)">/.exec(readFileSync(join(out, 'pages', page, 'index.html'), 'utf8'))?.[1]

  beforeAll(async () => {
    out = tempDir()
    manifest = await buildSite({ root: fixture('site'), outDir: out, cssInlineLimit: 100 })
  })

  it('links one hashed stylesheet per CSS set that keeps view transitions', () => {
    const html = readFileSync(join(out, 'pages/index/index.html'), 'utf8')
    expect(html).not.toContain('<style>')
    const href = hrefOf('index')
    expect(href).toMatch(/^\/assets\/weft-[0-9a-f]{8}\.css$/)
    const css = readFileSync(join(out, href!), 'utf8')
    expect(css).toContain('--weft-color-bg: #FAFAFA')
    expect(css).toContain('@view-transition { navigation: auto; }')
    expect(manifest.files[href!]).toMatchObject({ type: 'text/css; charset=utf-8', immutable: true })
    // Pages with the same CSS share a file; a page with component CSS gets its own.
    expect(hrefOf('404')).toBe(href)
    expect(hrefOf('about')).not.toBe(href)
    expect(readFileSync(join(out, hrefOf('about')!), 'utf8')).toMatch(/\.badge \{\s*letter-spacing: 0\.02em;\s*\}/)
  })

  it('hints the stylesheet, and keys the not-found page\'s hints "404"', () => {
    expect(manifest.hints['/']).toEqual([`<${hrefOf('index')}>; rel=preload; as=style`])
    expect(manifest.hints['404']).toEqual([`<${hrefOf('404')}>; rel=preload; as=style`])
    expect(manifest.hints['/404']).toBeUndefined()
  })
})

describe('buildSite into the outDir of an earlier build', () => {
  let out: string
  let first: Manifest

  beforeAll(async () => {
    out = tempDir()
    first = await buildSite({ root: fixture('site'), outDir: out })
    mkdirSync(join(out, 'pages/owner'), { recursive: true })
    writeFileSync(join(out, 'pages/owner/notes.txt'), 'owner')
    writeFileSync(join(out, 'assets/owner.css'), 'owner')
    writeFileSync(join(out, 'owner.txt'), 'owner')
    await buildSite({ root: fixture('static-site'), outDir: out })
  })

  it('removes the files of the earlier build that the new one does not make', () => {
    const now = files(out)
    for (const f of Object.keys(first.files).map((k) => k.slice(1))) {
      const kept = ['pages/index/index.html', 'routes.json'].includes(f)
      for (const v of [f, f + '.br', f + '.gz']) expect(now.includes(v), v).toBe(kept && !v.endsWith('.bin.br') && !v.endsWith('.bin.gz'))
    }
    // Directories it emptied go too; ones holding the owner's files stay.
    for (const d of ['pages/counter', 'pages/blog', 'pages/404']) expect(existsSync(join(out, d)), d).toBe(false)
    expect(existsSync(join(out, 'assets'))).toBe(true)
  })

  it('never touches what the owner put there', () => {
    for (const f of ['pages/owner/notes.txt', 'assets/owner.css', 'owner.txt']) {
      expect(readFileSync(join(out, f), 'utf8'), f).toBe('owner')
    }
  })

  it('leaves files that a symlink in outDir leads to', async () => {
    const dir = tempDir()
    const elsewhere = tempDir()
    const built = await buildSite({ root: fixture('site'), outDir: dir })
    const client = Object.keys(built.files).find((f) => f.startsWith('/assets/client-'))!.slice(1)
    // The owner moves the assets out and links them back.
    renameSync(join(dir, 'assets'), join(elsewhere, 'assets'))
    symlinkSync(join(elsewhere, 'assets'), join(dir, 'assets'))
    await buildSite({ root: fixture('static-site'), outDir: dir })
    expect(existsSync(join(elsewhere, client))).toBe(true)
  })

  it('refuses an outDir whose manifest.json is not a Weft manifest, and deletes nothing', async () => {
    const dir = tempDir()
    writeFileSync(join(dir, 'manifest.json'), '{"name":"not weft"}')
    writeFileSync(join(dir, 'routes.json'), '{}')
    await expect(buildSite({ root: fixture('static-site'), outDir: dir })).rejects.toThrow(
      `buildSite: ${join(dir, 'manifest.json')} is not a Weft build manifest; move it or choose another outDir.`,
    )
    expect(files(dir)).toEqual(['manifest.json', 'routes.json'])
  })
})

describe('buildSite refuses', () => {
  it('a public file that would overwrite a generated one', async () => {
    await expect(buildSite({ root: fixture('public-clash'), outDir: tempDir() })).rejects.toThrow(
      'buildSite: public/routes.json would overwrite the generated routes.json; rename or move it.',
    )
  })

  // Each case works on a copy of a fixture, so that a build which failed to
  // refuse could not write into, or delete from, the fixtures.
  const site = () => {
    const root = join(tempDir(), 'site')
    cpSync(fixture('static-site'), root, { recursive: true })
    return root
  }
  const refused = async (root: string, outDir: string) => {
    await expect(buildSite({ root, outDir })).rejects.toThrow(
      `buildSite: outDir ${outDir} is the site root, inside it, or contains it; build into a directory of its own.`,
    )
    expect(files(root)).toEqual(['env.d.ts', 'pages/index.vue', 'vite.config.ts'])
  }

  it('an outDir that is the site root, contains it, or is inside it', async () => {
    const root = site()
    for (const outDir of [root, join(root, '..'), join(root, 'pages'), join(root, 'dist')]) await refused(root, outDir)
  })

  it('an outDir that reaches the site root through a symlink', async () => {
    const root = site()
    const link = join(root, '..', 'link')
    symlinkSync(root, link)
    await refused(root, link)
    await refused(root, join(link, 'pages'))
  })

  it('a site root reached through a symlink', async () => {
    const root = site()
    const link = join(root, '..', 'link')
    symlinkSync(root, link)
    await refused(link, join(root, 'pages'))
  })

  it('an outDir that contains a root whose name starts with ".."', async () => {
    const parent = tempDir()
    const root = join(parent, '..site')
    cpSync(fixture('static-site'), root, { recursive: true })
    await refused(root, parent)
  })

  it.runIf(process.platform === 'darwin' || process.platform === 'win32')(
    'an outDir that differs from the root only in case, where the file system ignores case',
    async () => {
      const root = site()
      await refused(root, join(root, '..', 'SITE'))
    },
  )
})

describe('contentType', () => {
  it('knows the types a site serves', () => {
    expect(
      ['a.mjs', 'a.jpeg', 'a.gif', 'a.woff', 'a.woff2', 'a.ttf', 'a.wasm', 'a.webmanifest', 'a.ico', 'a.avif', 'a.webp'].map(
        contentType,
      ),
    ).toEqual([
      'text/javascript; charset=utf-8',
      'image/jpeg',
      'image/gif',
      'font/woff',
      'font/woff2',
      'font/ttf',
      'application/wasm',
      'application/manifest+json',
      'image/x-icon',
      'image/avif',
      'image/webp',
    ])
    expect(contentType('a.unknown')).toBe('application/octet-stream')
  })
})

describe('bin/weft-build-pages.mjs', () => {
  it('makes a production build from the command line', async () => {
    const out = tempDir()
    const bin = fileURLToPath(new URL('../bin/weft-build-pages.mjs', import.meta.url))
    // As `weft build` runs it: no NODE_ENV (vitest sets "test").
    const env = { ...process.env, NODE_ENV: undefined }
    const { stdout } = await promisify(execFile)(process.execPath, [bin, '--root', fixture('site'), '--out', out], { env })
    expect(stdout).toBe(`weft: built 8 pages into ${out}\n`)
    const js = files(out).filter((f) => f.endsWith('.js'))
    expect(js.length).toBeGreaterThan(0)
    // Vue's development build carries its warnings; the production build does not.
    for (const f of js) expect(readFileSync(join(out, f), 'utf8').includes('[Vue warn]'), f).toBe(false)
  })

  it('fails with one sentence when Node does not strip types', async () => {
    const bin = fileURLToPath(new URL('../bin/weft-build-pages.mjs', import.meta.url))
    const args = ['--no-experimental-strip-types', bin, '--root', fixture('static-site'), '--out', tempDir()]
    await expect(promisify(execFile)(process.execPath, args)).rejects.toMatchObject({
      code: 1,
      stderr: `weft-build-pages: Node ${process.version} does not strip TypeScript types; use Node 22.18 or later.\n`,
    })
  })

  it('fails with one sentence when an argument is missing', async () => {
    const bin = fileURLToPath(new URL('../bin/weft-build-pages.mjs', import.meta.url))
    await expect(promisify(execFile)(process.execPath, [bin, '--root', fixture('static-site')])).rejects.toMatchObject({
      code: 1,
      stderr: 'weft-build-pages: --out is required.\nusage: weft-build-pages --root <site> --out <dist> [--app-prefix /app] [--theme neutral|lisa]\n',
    })
  })
})
