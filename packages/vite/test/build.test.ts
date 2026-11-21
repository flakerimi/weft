import { execFile } from 'node:child_process'
import { createHash } from 'node:crypto'
import {
  chmodSync,
  cpSync,
  existsSync,
  lstatSync,
  mkdirSync,
  mkdtempSync,
  readdirSync,
  readFileSync,
  readlinkSync,
  realpathSync,
  renameSync,
  rmSync,
  statSync,
  symlinkSync,
  writeFileSync,
} from 'node:fs'
import { tmpdir } from 'node:os'
import { join, relative } from 'node:path'
import { fileURLToPath } from 'node:url'
import { promisify } from 'node:util'
import { brotliCompressSync, brotliDecompressSync, gunzipSync } from 'node:zlib'
import { afterAll, beforeAll, describe, expect, it, vi } from 'vitest'
import { buildSite, contentType, cssTargets, removeBuilt, type Manifest } from '../src/build.ts'
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

/** Everything under `dir` by lstat: a file's bytes, a symlink's target, a folder as "dir". */
function snapshot(dir: string): Record<string, string> {
  const out: Record<string, string> = {}
  for (const e of readdirSync(dir, { recursive: true, withFileTypes: true })) {
    const path = join(e.parentPath, e.name)
    const key = relative(dir, path)
    const stat = lstatSync(path)
    out[key] = stat.isSymbolicLink() ? `-> ${readlinkSync(path)}` : stat.isDirectory() ? 'dir' : readFileSync(path, 'base64')
  }
  return out
}

/** Staging or old folders a build left beside its outDir. */
const leftovers = (parent: string) => readdirSync(parent).filter((n) => n.includes('.weft-'))

/** A copy of a fixture in a temp folder that resolves packages through this package's node_modules. */
/**
 * For a build that must fail: Vite reports it ("✗ Build failed in …") and Vue
 * warns about an error it rethrows. Expect those, and only those, rather than
 * print them; call the result once the build has failed.
 */
function expectFailureLogs() {
  const error = vi.spyOn(console, 'error').mockImplementation(() => {})
  const warn = vi.spyOn(console, 'warn').mockImplementation(() => {})
  return () => {
    for (const [message] of [...error.mock.calls, ...warn.mock.calls]) {
      expect(String(message)).toMatch(/Build failed in |^\[Vue warn\]: Unhandled error during execution of /)
    }
    error.mockRestore()
    warn.mockRestore()
  }
}

function linkedSite(name: string): string {
  const root = join(tempDir(), name)
  cpSync(fixture(name), root, { recursive: true })
  symlinkSync(fileURLToPath(new URL('../node_modules', import.meta.url)), join(root, 'node_modules'))
  return root
}

const scripts = (html: string) => [...html.matchAll(/<script\b([^>]*)>/g)].map((m) => m[1]!)
const typeOf = (attrs: string) => /\btype="([^"]*)"/.exec(attrs)?.[1]
const styleOf = (html: string) => /<style>([\s\S]*?)<\/style>/.exec(html)?.[1]
/** The JSON of the page's one speculation rules script. */
const speculationRules = (html: string) =>
  JSON.parse(/<script type="speculationrules">([\s\S]*?)<\/script>/.exec(html)![1]!) as unknown
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
    // about.vue imports @weft/ui's base.css, which every page has anyway.
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
    expect(counter).toMatch(/button\s*\{\s*font-variant-numeric:\s*tabular-nums;?\s*\}/)
    expect(about).toMatch(/\.badge\s*\{\s*letter-spacing:\s*\.?0?\.?02em;?\s*\}/)
    // The rules themselves, not their words: base.css on every page may use those too.
    expect(index).not.toMatch(/button\s*\{\s*font-variant-numeric/)
    expect(index).not.toMatch(/\.badge\s*\{\s*letter-spacing/)
    // Component CSS comes before view transitions, after the theme and base.css.
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

  it('inlines the theme and base.css, minified, with view transitions, and adds speculation rules', () => {
    for (const p of ['index', 'counter']) {
      const html = read(`pages/${p}/index.html`)
      const css = styleOf(html)!
      expect(css).toContain('--weft-color-bg:#fafafa') // neutral, the default theme
      expect(css).toContain('box-sizing:border-box') // base.css
      expect(css).toContain('@view-transition{navigation:auto}')
      expect(css).not.toContain('/*') // no comments: minified
      expect(html).not.toContain('rel="stylesheet"')
      expect(speculationRules(html)).toEqual({
        prefetch: [{ where: { and: [{ href_matches: '/*' }, { not: { href_matches: ['/api', '/api/*'] } }] }, eagerness: 'moderate' }],
      })
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
    const served = all.filter(
      (f) => !(/\.(br|gz)$/.test(f) && all.includes(f.slice(0, -3))) && f !== 'manifest.json' && f !== '.weft-build',
    )
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

  it('marks the build with .weft-build, listing every other file it wrote', () => {
    const listed = files(out).filter((f) => f !== '.weft-build')
    expect(JSON.parse(read('.weft-build'))).toEqual({
      weft: 1,
      files: listed.map((path) => {
        const bytes = readFileSync(join(out, path))
        return { path, size: bytes.length, sha256: createHash('sha256').update(bytes).digest('hex') }
      }),
    })
    expect(manifest.files['/.weft-build']).toBeUndefined()
    expect(leftovers(join(out, '..'))).toEqual([])
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
  let app: string

  beforeAll(async () => {
    out = tempDir()
    // An app built before the pages, in a folder of its own: its files are copied into app/.
    app = tempDir()
    mkdirSync(join(app, 'assets'))
    writeFileSync(join(app, 'index.html'), '<!doctype html><div id="app"></div>')
    writeFileSync(join(app, 'assets/app.js'), 'console.log(1)')
    await buildSite({ root: fixture('static-site'), outDir: out, appPrefix: '/app', appDir: app, theme: 'lisa' })
  })

  it("uses the site's own vite.config", () => {
    expect(readFileSync(join(out, 'pages/index/index.html'), 'utf8')).toContain('<h1>Static from vite.config</h1>')
  })

  it('copies the app build into app/', () => {
    expect(readFileSync(join(out, 'app/index.html'), 'utf8')).toBe('<!doctype html><div id="app"></div>')
    expect(readFileSync(join(out, 'app/assets/app.js'), 'utf8')).toBe('console.log(1)')
  })

  it('makes no client build and no runtime script', () => {
    expect(existsSync(join(out, 'assets'))).toBe(false)
    const html = readFileSync(join(out, 'pages/index/index.html'), 'utf8')
    expect(scripts(html).map(typeOf)).toEqual(['speculationrules'])
  })

  it('prefetches no API or app URL', () => {
    const html = readFileSync(join(out, 'pages/index/index.html'), 'utf8')
    expect(speculationRules(html)).toEqual({
      prefetch: [
        {
          where: { and: [{ href_matches: '/*' }, { not: { href_matches: ['/api', '/api/*', '/app', '/app/*'] } }] },
          eagerness: 'moderate',
        },
      ],
    })
  })

  it('inlines the chosen theme', () => {
    const css = styleOf(readFileSync(join(out, 'pages/index/index.html'), 'utf8'))!
    expect(css).toContain('--weft-color-bg:#f1f5f9') // lisa
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

describe('buildSite with files its CSS names', () => {
  let out: string
  let manifest: Manifest
  const html = (page: string) => readFileSync(join(out, 'pages', page, 'index.html'), 'utf8')
  const sha = (bytes: Buffer) => createHash('sha256').update(bytes).digest('hex').slice(0, 8)
  const source = (p: string) => readFileSync(join(fixture('css-assets'), p))
  const lucide = () => readFileSync(fileURLToPath(new URL('../node_modules/lucide-static/font/lucide.woff2', import.meta.url)))

  beforeAll(async () => {
    out = tempDir()
    manifest = await buildSite({ root: fixture('css-assets'), outDir: out, theme: 'weft' })
  })

  it('copies each file to /assets/ named by a hash of its bytes, and points the CSS there', () => {
    const font = `/assets/Test-${sha(source('styles/fonts/Test.woff2'))}.woff2`
    const ext = `/assets/Test-ext-${sha(source('styles/fonts/Test-ext.woff2'))}.woff2`
    const dot = `/assets/dot-${sha(source('styles/dot.svg'))}.svg`
    const packaged = `/assets/lucide-${sha(lucide())}.woff2`
    const css = styleOf(html('index'))!
    for (const url of [font, dot, packaged]) expect(css).toContain(`url(${url})`)
    // A query and a fragment stay on the URL.
    expect(css).toContain(`url(${ext}?v=1#ext)`)
    expect(css).not.toContain('/@fs/')
    expect(readFileSync(join(out, font.slice(1)))).toEqual(source('styles/fonts/Test.woff2'))
    expect(readFileSync(join(out, packaged.slice(1)))).toEqual(lucide())
    expect(manifest.files[font]).toMatchObject({ type: 'font/woff2', immutable: true, br: false })
    expect(manifest.files[dot]).toMatchObject({ type: 'image/svg+xml', immutable: true, br: true })
  })

  it('leaves public files, data URLs and other sites as they are', () => {
    const css = styleOf(html('index'))!
    expect(css).toContain('url(/robots.txt)')
    expect(css).toContain('url(data:image/gif;base64,R0lGODlhAQABAAAAACw=)')
    expect(css).toContain('url(https://example.com/x.png)')
    expect(files(out).filter((f) => f.startsWith('assets/')).map((f) => f.replace(/-[0-9a-f]{8}\./, '-#.')).sort()).toEqual(
      ['assets/Test-#.woff2', 'assets/Test-ext-#.woff2', 'assets/dot-#.svg', 'assets/dot-#.svg.br', 'assets/dot-#.svg.gz', 'assets/lucide-#.woff2'],
    )
  })

  it('preloads the Latin faces of the families the page uses, in its head and in its Early Hints', () => {
    const font = `/assets/Test-${sha(source('styles/fonts/Test.woff2'))}.woff2`
    const head = html('index').split('</head>')[0]!
    expect(head).toContain(`<link rel="preload" href="${font}" as="font" type="font/woff2" crossorigin>`)
    expect(manifest.hints['/']).toContain(`<${font}>; rel=preload; as=font; type="font/woff2"; crossorigin`)
    // The Latin Extended face is for text this page may never show, and no rule uses the icon font.
    expect(head.match(/rel="preload"/g)).toHaveLength(1)
    expect(manifest.hints['/']!.filter((l) => l.includes('as=font'))).toHaveLength(1)
    // Before the stylesheet that names it, so the fetch starts first.
    expect(head.indexOf('rel="preload"')).toBeLessThan(head.indexOf('<style>'))
    expect(html('plain')).not.toContain('rel="preload"')
    expect(manifest.hints['/plain']).toBeUndefined()
  })

  it('preloads a font the site lists in preloadFonts, by its file name', async () => {
    const opted = tempDir()
    const built = await buildSite({ root: fixture('css-assets'), outDir: opted, theme: 'weft', preloadFonts: ['lucide.woff2'] })
    const head = readFileSync(join(opted, 'pages/index/index.html'), 'utf8').split('</head>')[0]!
    const icons = `/assets/lucide-${sha(lucide())}.woff2`
    expect(head).toContain(`<link rel="preload" href="${icons}" as="font" type="font/woff2" crossorigin>`)
    expect(head.match(/rel="preload"/g)).toHaveLength(2)
    expect(built.hints['/']).toContain(`<${icons}>; rel=preload; as=font; type="font/woff2"; crossorigin`)
  })

  it('inlines the weft theme', () => {
    expect(styleOf(html('plain'))).toContain('--weft-color-bg:#e8ecf4')
  })
})

describe('buildSite refuses a url() in CSS it cannot serve', { timeout: 30_000 }, () => {
  /** Build the css-unsafe site with `css` as its stylesheet; the error the build fails with. */
  async function refusal(css: string, setup?: (root: string) => void): Promise<string> {
    const root = linkedSite('css-unsafe')
    writeFileSync(join(root, 'styles/bad.css'), css)
    setup?.(root)
    const out = join(tempDir(), 'dist')
    const done = expectFailureLogs()
    const error = await buildSite({ root, outDir: out }).then(
      () => '(built)',
      (e: Error) => e.message,
    )
    done()
    expect(existsSync(out)).toBe(false)
    return error
  }
  const outside = (url: string) =>
    `buildSite: the CSS of pages/index.vue names url(${url}), a file outside the site, its public/ folder and the packages it uses; put the file in the site.`

  it('a file of the system, through /@fs/', async () => {
    expect(await refusal('.x { background: url(/@fs/etc/passwd); }')).toBe(outside('/@fs/etc/passwd'))
  })

  it('a path from the site root that climbs out of it', async () => {
    expect(await refusal('.x { background: url(/../../x.png); }')).toBe(outside('/../../x.png'))
  })

  it('a file of the site that is a symlink to one outside it', async () => {
    const target = join(tempDir(), 'outside.png')
    writeFileSync(target, 'x'.repeat(5000))
    const error = await refusal('.x { background: url(./outside.png); }', (root) => symlinkSync(target, join(root, 'styles/outside.png')))
    // The render server follows the link and names the file where it is.
    expect(error).toBe(outside(`/@fs${realpathSync(target)}`))
  })

  it('a path that names no file', async () => {
    expect(await refusal('.x { background: url(/nope.png); }')).toBe(
      'buildSite: the CSS of pages/index.vue names url(/nope.png), which is no file of the site or its public/ folder; fix the path, or give another site\'s file by its full URL.',
    )
  })

  it('a path whose % escapes do not decode', async () => {
    expect(await refusal('.x { background: url(/bad%zz.png); }')).toBe(
      'buildSite: the CSS of pages/index.vue names url(/bad%zz.png), whose % escapes do not decode; write a % in a name as %25.',
    )
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
    expect(css).toContain('--weft-color-bg:#fafafa')
    expect(css).toContain('@view-transition{navigation:auto}')
    expect(manifest.files[href!]).toMatchObject({ type: 'text/css; charset=utf-8', immutable: true })
    // Pages with the same CSS share a file; a page with component CSS gets its own.
    expect(hrefOf('404')).toBe(href)
    expect(hrefOf('about')).not.toBe(href)
    expect(readFileSync(join(out, hrefOf('about')!), 'utf8')).toMatch(/\.badge\s*\{\s*letter-spacing:\s*\.?0?\.?02em;?\s*\}/)
  })

  it('measures the limit in brotli bytes, as the page is sent, not in raw bytes', async () => {
    const css = readFileSync(join(out, hrefOf('index')!), 'utf8')
    const raw = Buffer.byteLength(css)
    const sent = brotliCompressSync(css).length
    expect(raw).toBeGreaterThan(sent)
    // A limit between the two: the raw CSS is over it, what is sent is not, so it is inlined.
    const between = tempDir()
    await buildSite({ root: fixture('site'), outDir: between, cssInlineLimit: Math.floor((raw + sent) / 2) })
    const html = readFileSync(join(between, 'pages/index/index.html'), 'utf8')
    expect(html).toContain('<style>')
    expect(html).not.toContain('rel="stylesheet"')
  })

  it('hints the stylesheet, and keys the not-found page\'s hints "404"', () => {
    expect(manifest.hints['/']).toEqual([`<${hrefOf('index')}>; rel=preload; as=style`])
    expect(manifest.hints['404']).toEqual([`<${hrefOf('404')}>; rel=preload; as=style`])
    expect(manifest.hints['/404']).toBeUndefined()
  })
})

// Each test makes whole builds, some of them twice.
describe('buildSite replaces only a folder it built', { timeout: 30_000 }, () => {
  /** The message for an outDir the build will not replace. */
  const notOurs = (outDir: string, why: string) =>
    `buildSite: outDir ${outDir} ${why}; Weft replaces only an empty folder or one it built, so move that out or build into a new folder.`

  it('builds into a missing outDir, and a rebuild replaces the folder it built', async () => {
    const parent = tempDir()
    const out = join(parent, 'dist')
    const first = await buildSite({ root: fixture('site'), outDir: out })
    expect(first.files['/pages/counter/index.html']).toBeDefined()
    expect(readdirSync(parent)).toEqual(['dist'])
    await buildSite({ root: fixture('static-site'), outDir: out })
    // Nothing of the first build is left: the folder was replaced as a whole.
    expect(existsSync(join(out, 'pages/counter'))).toBe(false)
    expect(existsSync(join(out, 'assets'))).toBe(false)
    expect(readFileSync(join(out, 'pages/index/index.html'), 'utf8')).toContain('<h1>Static from vite.config</h1>')
    const marker = JSON.parse(readFileSync(join(out, '.weft-build'), 'utf8')) as { files: { path: string }[] }
    expect(marker.files.map((f) => f.path)).toEqual(files(out).filter((f) => f !== '.weft-build'))
    // A folder anyone can read, as a build folder is, not mkdtemp's owner-only one.
    expect(statSync(out).mode & 0o777).toBe(0o755)
    expect(readdirSync(parent)).toEqual(['dist'])
  })

  /** A folder holding a fresh build of the static site, and what it holds. */
  const built = async () => {
    const parent = tempDir()
    const out = join(parent, 'dist')
    await buildSite({ root: fixture('static-site'), outDir: out })
    return { parent, out }
  }
  const refusedOver = async (out: string, why: string) => {
    const before = snapshot(out)
    await expect(buildSite({ root: fixture('static-site'), outDir: out })).rejects.toThrow(notOurs(out, why))
    expect(snapshot(out)).toEqual(before)
    expect(leftovers(join(out, '..'))).toEqual([])
  }

  it('refuses a built folder the owner added a file to, and leaves it as it was', async () => {
    const { out } = await built()
    mkdirSync(join(out, 'pages/owner'))
    writeFileSync(join(out, 'pages/owner/notes.txt'), 'owner')
    await refusedOver(out, 'holds pages/owner, which Weft did not build')
  })

  it('refuses a built folder with a compressed file the owner added', async () => {
    const { out } = await built()
    writeFileSync(join(out, 'pages/index/index.html.zst'), 'owner')
    await refusedOver(out, 'holds pages/index/index.html.zst, which Weft did not build')
  })

  it('refuses a built folder that lost a file', async () => {
    const { out } = await built()
    renameSync(join(out, 'routes.json.gz'), join(tempDir(), 'moved'))
    await refusedOver(out, 'no longer holds routes.json.gz, which Weft built')
  })

  it('refuses a built folder holding a symlink, and leaves what it leads to', async () => {
    const { out } = await built()
    const elsewhere = tempDir()
    writeFileSync(join(elsewhere, 'victim.txt'), 'owner')
    symlinkSync(join(elsewhere, 'victim.txt'), join(out, 'link.txt'))
    await refusedOver(out, 'holds the symlink link.txt')
    expect(readFileSync(join(elsewhere, 'victim.txt'), 'utf8')).toBe('owner')
  })

  it('refuses a built folder whose own folder became a symlink', async () => {
    const { out } = await built()
    // The owner moves pages/ out and links it back: its files are listed, but not ours to delete.
    const elsewhere = tempDir()
    renameSync(join(out, 'pages'), join(elsewhere, 'pages'))
    symlinkSync(join(elsewhere, 'pages'), join(out, 'pages'))
    const theirs = snapshot(elsewhere)
    await refusedOver(out, 'holds the symlink pages')
    expect(snapshot(elsewhere)).toEqual(theirs)
  })

  it('refuses an outDir that is a symlink to a built folder', async () => {
    const { out } = await built()
    const link = join(tempDir(), 'dist')
    symlinkSync(out, link)
    const before = snapshot(out)
    await expect(buildSite({ root: fixture('static-site'), outDir: link })).rejects.toThrow(notOurs(link, 'is a symlink'))
    expect(snapshot(out)).toEqual(before)
  })

  it('refuses a folder without a .weft-build, even one holding the generated names', async () => {
    const out = tempDir()
    const elsewhere = tempDir()
    writeFileSync(join(elsewhere, 'victim.txt'), 'owner')
    writeFileSync(join(out, 'routes.json'), 'owner routes')
    writeFileSync(join(out, 'manifest.json'), JSON.stringify({ files: { '/x': { etag: '"x"' } }, hints: {} }))
    mkdirSync(join(out, 'pages/index'), { recursive: true })
    symlinkSync(join(elsewhere, 'victim.txt'), join(out, 'pages/index/index.html'))
    await refusedOver(out, 'holds manifest.json, which Weft did not build')
    expect(readFileSync(join(elsewhere, 'victim.txt'), 'utf8')).toBe('owner')
  })

  it('refuses a .weft-build it cannot read', async () => {
    const entry = (path: string) => JSON.stringify({ path, size: 0, sha256: createHash('sha256').digest('hex') })
    for (const marker of [
      'not json',
      '{"weft":2,"files":[]}',
      '{"weft":1}',
      '{"weft":1,"files":["x"]}',
      `{"weft":1,"files":[${entry('../x')}]}`,
      `{"weft":1,"files":[${entry('/etc/x')}]}`,
      '{"weft":1,"files":[{"path":"x","size":0}]}',
    ]) {
      const out = tempDir()
      writeFileSync(join(out, '.weft-build'), marker)
      await refusedOver(out, 'has a .weft-build that Weft cannot read')
    }
  })

  it('refuses a built folder whose listed file the owner changed, and keeps the change', async () => {
    const { out } = await built()
    writeFileSync(join(out, 'routes.json'), 'OWNER HAND EDIT')
    await refusedOver(out, 'holds routes.json, changed since Weft built it')
    expect(readFileSync(join(out, 'routes.json'), 'utf8')).toBe('OWNER HAND EDIT')
  })

  it('refuses a built folder where the owner moved a file of theirs onto a listed name', async () => {
    const { out } = await built()
    const mine = join(tempDir(), 'mine.txt')
    writeFileSync(mine, 'OWNER FILE')
    renameSync(mine, join(out, 'routes.json.gz'))
    await refusedOver(out, 'holds routes.json.gz, changed since Weft built it')
    expect(readFileSync(join(out, 'routes.json.gz'), 'utf8')).toBe('OWNER FILE')
  })

  it("names a Finder .DS_Store in a built folder", async () => {
    const { out } = await built()
    writeFileSync(join(out, 'pages/.DS_Store'), 'finder')
    const before = snapshot(out)
    await expect(buildSite({ root: fixture('static-site'), outDir: out })).rejects.toThrow(
      `buildSite: outDir ${out} holds pages/.DS_Store (made by Finder); remove it or build into a new folder.`,
    )
    expect(snapshot(out)).toEqual(before)
  })

  it('refuses a .weft-build that is a symlink or a folder, naming it', async () => {
    const real = join(tempDir(), 'marker')
    writeFileSync(real, '{"weft":1,"files":[]}')
    const linked = tempDir()
    symlinkSync(real, join(linked, '.weft-build'))
    await refusedOver(linked, 'has a .weft-build that is not a regular file')
    const folder = tempDir()
    mkdirSync(join(folder, '.weft-build'))
    await refusedOver(folder, 'has a .weft-build that is not a regular file')
  })

  it.runIf(process.getuid?.() !== 0)('refuses a built folder with a folder it cannot read, in one sentence', async () => {
    const { out } = await built()
    chmodSync(join(out, 'pages'), 0o000)
    try {
      await expect(buildSite({ root: fixture('static-site'), outDir: out })).rejects.toThrow(
        notOurs(out, 'holds pages, which Weft cannot read'),
      )
    } finally {
      chmodSync(join(out, 'pages'), 0o755)
    }
    expect(leftovers(join(out, '..'))).toEqual([])
  })

  it.runIf(process.platform === 'linux')('refuses a built folder holding a second name in another Unicode form', async () => {
    const root = linkedSite('static-site')
    mkdirSync(join(root, 'public'))
    writeFileSync(join(root, 'public/café.txt'), 'built')
    const out = join(tempDir(), 'dist')
    await buildSite({ root, outDir: out })
    writeFileSync(join(out, 'café.txt'), 'built')
    const before = snapshot(out)
    await expect(buildSite({ root, outDir: out })).rejects.toThrow(/holds caf.*\.txt, whose name another entry has in another Unicode form/)
    expect(snapshot(out)).toEqual(before)
  })

  it.runIf(process.getuid?.() !== 0)(
    'succeeds, with one warning, when the earlier build cannot be removed after the swap',
    async () => {
      const { parent, out } = await built()
      chmodSync(join(out, 'pages/index'), 0o555)
      const warn = vi.spyOn(console, 'warn').mockImplementation(() => {})
      try {
        await buildSite({ root: fixture('static-site'), outDir: out })
        const [old] = leftovers(parent)
        expect(old).toMatch(/^\.dist\.weft-old-/)
        expect(warn.mock.calls).toEqual([
          [`buildSite: the build is in ${out}, but the earlier build moved to ${join(parent, old!)} could not be removed; remove that folder by hand.`],
        ])
        chmodSync(join(parent, old!, 'pages/index'), 0o755)
      } finally {
        warn.mockRestore()
      }
      expect(existsSync(join(out, 'pages/index/index.html'))).toBe(true)
      expect(statSync(join(out, 'pages/index')).mode & 0o777).not.toBe(0o555)
    },
  )

  it('leaves outDir as it was, and no staging folder, when the build fails', async () => {
    const root = linkedSite('static-site')
    const parent = tempDir()
    const out = join(parent, 'dist')
    await buildSite({ root, outDir: out })
    const before = snapshot(out)
    // A page that throws as it renders.
    writeFileSync(join(root, 'pages/boom.vue'), '<script setup vapor lang="ts">\nthrow new Error(\'boom\')\n</script>\n<template><p /></template>\n')
    let failed = expectFailureLogs()
    await expect(buildSite({ root, outDir: out })).rejects.toThrow('boom')
    failed()
    rmSync(join(root, 'pages/boom.vue'))
    // An island the client build cannot bundle: it imports a module that does not exist.
    mkdirSync(join(root, 'components'))
    writeFileSync(
      join(root, 'components/Broken.vue'),
      "<script setup vapor lang=\"ts\">\nconst load = () => import('weft-no-such-module')\n</script>\n<template><button type=\"button\" @click=\"load\">x</button></template>\n",
    )
    writeFileSync(
      join(root, 'pages/broken.vue'),
      "<script setup vapor lang=\"ts\">\nimport { Island } from '@weft/ui'\nimport Broken from '../components/Broken.vue'\n</script>\n<template><Island when=\"load\" :component=\"Broken\" /></template>\n",
    )
    failed = expectFailureLogs()
    await expect(buildSite({ root, outDir: out })).rejects.toThrow('weft-no-such-module')
    failed()
    expect(snapshot(out)).toEqual(before)
    expect(readdirSync(parent)).toEqual(['dist'])
  })

  it('builds into <root>/dist, and a rebuild leaves the site as it was', async () => {
    const root = linkedSite('site')
    const out = join(root, 'dist')
    const first = await buildSite({ root, outDir: out })
    expect(first.files['/pages/escape/index.html']).toBeDefined()
    // The page goes from the (temporary) site; its built files go with the next build.
    rmSync(join(root, 'pages/escape.vue'))
    const site = snapshot(root)
    const second = await buildSite({ root, outDir: out })
    expect(second.files['/pages/escape/index.html']).toBeUndefined()
    expect(existsSync(join(out, 'pages/escape'))).toBe(false)
    expect(existsSync(join(out, 'pages/counter/index.html'))).toBe(true)
    // Beside dist, the site is as it was: no staging folder, nothing written or removed.
    const source = (snap: Record<string, string>) => Object.entries(snap).filter(([k]) => k !== 'dist' && !k.startsWith('dist/'))
    expect(source(snapshot(root))).toEqual(source(site))
  })
})

describe('buildSite with an app build', { timeout: 30_000 }, () => {
  const appFiles = (dir: string) => {
    mkdirSync(join(dir, 'assets'), { recursive: true })
    writeFileSync(join(dir, 'index.html'), '<!doctype html><div id="app"></div>')
    writeFileSync(join(dir, 'assets/main.js'), 'console.log("app")')
    // A compressed copy the app's own build made: the page build makes its own.
    writeFileSync(join(dir, 'assets/main.js.gz'), 'app gz')
  }

  it('copies the app into app/ and leaves its own folder as it was, build after build', async () => {
    const app = tempDir()
    appFiles(app)
    const before = snapshot(app)
    const out = join(tempDir(), 'dist')
    for (let i = 0; i < 2; i++) {
      const manifest = await buildSite({ root: fixture('static-site'), outDir: out, appPrefix: '/app', appDir: app })
      expect(readFileSync(join(out, 'app/assets/main.js'), 'utf8')).toBe('console.log("app")')
      expect(gunzipSync(readFileSync(join(out, 'app/assets/main.js.gz'))).toString()).toBe('console.log("app")')
      expect(manifest.files['/app/assets/main.js']).toMatchObject({ br: true, gz: true, immutable: false })
      expect(snapshot(app)).toEqual(before)
    }
    // Without appPrefix the app is not part of the build, and is still left alone.
    const manifest = await buildSite({ root: fixture('static-site'), outDir: out, appDir: app })
    expect(existsSync(join(out, 'app'))).toBe(false)
    expect(manifest.files['/app/index.html']).toBeUndefined()
    expect(snapshot(app)).toEqual(before)
  })

  it("marks the app's content-hashed assets its Vite build emitted immutable, and nothing else of it", async () => {
    const app = tempDir()
    appFiles(app)
    writeFileSync(join(app, 'assets/index-FeUbVT--.js'), 'hashed')
    writeFileSync(join(app, 'assets/logo-D9x2kQ7a.svg'), '<svg/>')
    writeFileSync(join(app, 'assets/index-Cq3xY7pL.css'), 'a{}')
    // Hand-placed in the app's public/ folder: named like a hash, but nothing guarantees it changes when it does.
    writeFileSync(join(app, 'assets/vendor-Ab12Cd34.js'), 'hand-placed')
    writeFileSync(join(app, 'assets/app-settings.js'), 'a name, not a hash')
    writeFileSync(join(app, 'boot-Ab12Cd34.js'), 'outside assets/')
    mkdirSync(join(app, '.vite'))
    // Vite's build manifest (build.manifest: true) lists what the build emitted; main.js is emitted but not hashed.
    writeFileSync(
      join(app, '.vite/manifest.json'),
      JSON.stringify({
        'index.html': { file: 'assets/index-FeUbVT--.js', src: 'index.html', isEntry: true, css: ['assets/index-Cq3xY7pL.css'], assets: ['assets/logo-D9x2kQ7a.svg'] },
        'src/main.ts': { file: 'assets/main.js', src: 'src/main.ts', isDynamicEntry: true },
      }),
    )
    const manifest = await buildSite({ root: fixture('static-site'), outDir: tempDir(), appPrefix: '/admin', appDir: app })
    const immutable = Object.keys(manifest.files).filter((f) => manifest.files[f]!.immutable).sort()
    expect(immutable).toEqual(['/admin/assets/index-Cq3xY7pL.css', '/admin/assets/index-FeUbVT--.js', '/admin/assets/logo-D9x2kQ7a.svg'])
    expect(manifest.files['/admin/assets/vendor-Ab12Cd34.js']).toMatchObject({ immutable: false })
    expect(manifest.files['/admin/assets/main.js']).toMatchObject({ immutable: false })
    expect(manifest.files['/admin/index.html']).toMatchObject({ immutable: false })
  })

  it("neither copies nor serves the app's .vite folder: its build manifest is for the page build only", async () => {
    const app = tempDir()
    appFiles(app)
    mkdirSync(join(app, '.vite'))
    writeFileSync(join(app, '.vite/manifest.json'), JSON.stringify({ 'index.html': { file: 'assets/main.js' } }))
    writeFileSync(join(app, '.vite/ssr-manifest.json'), '{}')
    const out = tempDir()
    const manifest = await buildSite({ root: fixture('static-site'), outDir: out, appPrefix: '/admin', appDir: app })
    expect(existsSync(join(out, 'admin/.vite'))).toBe(false)
    expect(Object.keys(manifest.files).filter((f) => f.includes('.vite'))).toEqual([])
    expect(manifest.files['/admin/assets/main.js']).toBeDefined()
  })

  it("marks nothing of the app immutable without its Vite build manifest: a hashed-looking name may be hand-placed", async () => {
    const app = tempDir()
    appFiles(app)
    writeFileSync(join(app, 'assets/index-FeUbVT--.js'), 'hashed')
    const manifest = await buildSite({ root: fixture('static-site'), outDir: tempDir(), appPrefix: '/admin', appDir: app })
    expect(Object.keys(manifest.files).filter((f) => manifest.files[f]!.immutable)).toEqual([])
  })

  it("refuses an app build manifest it cannot read, in one sentence", async () => {
    const app = tempDir()
    appFiles(app)
    mkdirSync(join(app, '.vite'))
    writeFileSync(join(app, '.vite/manifest.json'), '{"index.html": {"file": 3}}')
    await expect(buildSite({ root: fixture('static-site'), outDir: tempDir(), appPrefix: '/admin', appDir: app })).rejects.toThrow(
      new Error(`buildSite: ${join(app, '.vite/manifest.json')} is not a Vite build manifest (each entry has a file name); rebuild the app, or remove it.`),
    )
  })

  it('copies the app into the folder its prefix names, so its URLs are its files', async () => {
    const app = tempDir()
    appFiles(app)
    for (const [prefix, folder] of [
      ['/admin', 'admin'],
      ['/tools/admin', 'tools/admin'],
      ['/constructor', 'constructor'],
    ]) {
      const out = tempDir()
      const manifest = await buildSite({ root: fixture('static-site'), outDir: out, appPrefix: prefix, appDir: app })
      expect(readFileSync(join(out, folder, 'index.html'), 'utf8')).toBe('<!doctype html><div id="app"></div>')
      expect(readFileSync(join(out, folder, 'assets/main.js'), 'utf8')).toBe('console.log("app")')
      expect(existsSync(join(out, 'app'))).toBe(false)
      expect(manifest.files[`${prefix}/index.html`]).toMatchObject({ type: 'text/html; charset=utf-8', br: true, gz: true })
      expect(manifest.files[`${prefix}/assets/main.js`]).toMatchObject({ br: true, gz: true })
      const routes = JSON.parse(readFileSync(join(out, 'routes.json'), 'utf8')) as RoutesJson
      expect(routes.app).toEqual({ prefix, index: `${folder}/index.html` })
    }
  })

  it('refuses an appPrefix that is not a folder of its own, in one sentence', async () => {
    const app = tempDir()
    appFiles(app)
    const build = (appPrefix: string) => buildSite({ root: fixture('static-site'), outDir: tempDir(), appPrefix, appDir: app })
    for (const prefix of ['app', '/app/', '//app']) {
      await expect(build(prefix)).rejects.toThrow(`buildSite: appPrefix "${prefix}" must start with "/" and not end with one, like "/app".`)
    }
    for (const prefix of ['/a//b', '/a/./b', '/a/../b', '/a\\b']) {
      await expect(build(prefix)).rejects.toThrow(`buildSite: appPrefix "${prefix}" has an empty, ".", ".." or backslashed segment.`)
    }
    for (const prefix of ['/a?b', '/a#b', '/a%20b', '/a b', '/app/x?y']) {
      await expect(build(prefix)).rejects.toThrow(
        `buildSite: appPrefix "${prefix}" has a character other than A-Z, a-z, 0-9, ".", "_", "~" and "-"; use a plain path, like "/app".`,
      )
    }
    for (const [prefix, why] of [
      ['/pages', 'holds the built pages'],
      ['/Pages/x', 'holds the built pages'],
      ['/api', 'the Go server sends to the API'],
      ['/api/app', 'the Go server sends to the API'],
    ]) {
      const first = prefix.split('/')[1]
      await expect(build(prefix)).rejects.toThrow(`buildSite: appPrefix "${prefix}" is under /${first}, which ${why}; use another prefix, like "/app".`)
    }
  })

  it('takes the app from <root>/app-dist unless told otherwise', async () => {
    const root = linkedSite('static-site')
    appFiles(join(root, 'app-dist'))
    const before = snapshot(join(root, 'app-dist'))
    const out = join(root, 'dist')
    await buildSite({ root, outDir: out, appPrefix: '/app' })
    await buildSite({ root, outDir: out, appPrefix: '/app' })
    expect(readFileSync(join(out, 'app/index.html'), 'utf8')).toBe('<!doctype html><div id="app"></div>')
    expect(snapshot(join(root, 'app-dist'))).toEqual(before)
  })

  it('refuses an app build that is missing its index.html', async () => {
    const app = tempDir()
    await expect(buildSite({ root: fixture('static-site'), outDir: tempDir(), appPrefix: '/app', appDir: app })).rejects.toThrow(
      `buildSite: appPrefix is /app but ${join(app, 'index.html')} is missing; build the app into ${app} first, or set appDir.`,
    )
  })

  it('refuses an app build inside outDir, or an outDir inside the app build', async () => {
    const parent = tempDir()
    const out = join(parent, 'dist')
    const app = join(parent, 'app')
    appFiles(app)
    await buildSite({ root: fixture('static-site'), outDir: out, appPrefix: '/app', appDir: app })
    const before = snapshot(out)
    // The copy in outDir/app taken for the app build, and an outDir inside the app build.
    for (const [outDir, appDir] of [
      [out, join(out, 'app')],
      [join(app, 'dist'), app],
    ] as const) {
      await expect(buildSite({ root: fixture('static-site'), outDir, appPrefix: '/app', appDir })).rejects.toThrow(
        `buildSite: outDir ${outDir} and the app build ${appDir} overlap; keep the app build in a folder of its own.`,
      )
    }
    expect(snapshot(out)).toEqual(before)
  })
})

describe('buildSite with site build options that name output files', { timeout: 30_000 }, () => {
  it('writes the client build only into the build, whatever the site config says', async () => {
    const root = linkedSite('site')
    const elsewhere = tempDir()
    writeFileSync(
      join(root, 'vite.config.ts'),
      `import { defineConfig } from 'vite'
export default defineConfig({
  build: {
    outDir: ${JSON.stringify(elsewhere)},
    assetsDir: '../escaped',
    rollupOptions: {
      external: ['weft-test-external'],
      output: {
        dir: ${JSON.stringify(elsewhere)},
        entryFileNames: '../escaped/[name].js',
        chunkFileNames: '../escaped/[name].js',
        assetFileNames: '../escaped/[name][extname]',
      },
    },
  },
})
`,
    )
    const parent = tempDir()
    const out = join(parent, 'dist')
    const manifest = await buildSite({ root, outDir: out })
    expect(readdirSync(elsewhere)).toEqual([])
    expect(readdirSync(parent)).toEqual(['dist'])
    expect(existsSync(join(root, 'escaped'))).toBe(false)
    const js = Object.keys(manifest.files).filter((f) => f.endsWith('.js'))
    expect(js.length).toBeGreaterThan(0)
    for (const f of js) expect(f).toMatch(/^\/assets\/[\w-]+-[\w-]{8}\.js$/)
    // The site's other options stay: the external is still left to the browser.
    const lazy = js.find((f) => f.startsWith('/assets/Lazy-'))!
    expect(readFileSync(join(out, lazy), 'utf8')).toMatch(/import\(\s*["'`]weft-test-external["'`]\s*\)/)
  })
})

const refusal = (outDir: string) =>
  `buildSite: outDir ${outDir} is the site root, contains it, or is one of its source folders; build into a folder of its own, like dist.`

describe('buildSite next to the .vue files it uses', { timeout: 30_000 }, () => {
  it('builds into <root>/dist when a page uses <root>/Layout.vue', async () => {
    const root = linkedSite('static-site')
    writeFileSync(join(root, 'Layout.vue'), '<script setup vapor lang="ts"></script>\n<template><main><slot /></main></template>\n')
    writeFileSync(
      join(root, 'pages/index.vue'),
      "<script setup vapor lang=\"ts\">\nimport Layout from '../Layout.vue'\n</script>\n<template><Layout><h1>hi</h1></Layout></template>\n",
    )
    await buildSite({ root, outDir: join(root, 'dist') })
    expect(readFileSync(join(root, 'dist/pages/index/index.html'), 'utf8')).toMatch(/<main>[\s\S]*<h1>hi<\/h1>[\s\S]*<\/main>/)
  })
})

describe('buildSite with a site config that moves the client build', { timeout: 30_000 }, () => {
  const sentence = (setting: string) =>
    `buildSite: the site's Vite config or a plugin sets ${setting} for the client build; the page build alone decides where that goes, so remove it.`

  // Each sends the client build to another folder of the owner's, and asks Vite to empty it first.
  const routes: [string, (owner: string) => string, string][] = [
    [
      'an environment outDir',
      (o) => `environments: { client: { build: { outDir: ${JSON.stringify(o)}, emptyOutDir: true } } }`,
      'build.outDir',
    ],
    [
      'an environment output.dir',
      (o) => `environments: { client: { build: { emptyOutDir: true, rollupOptions: { output: { dir: ${JSON.stringify(o)} } } } } }`,
      'build.rollupOptions.output.dir',
    ],
    [
      "a plugin's config hook",
      (o) => `plugins: [{ name: 'redirect', config: () => ({ build: { outDir: ${JSON.stringify(o)}, emptyOutDir: true } }) }]`,
      'build.outDir',
    ],
    [
      "a plugin's outputOptions hook",
      (o) => `build: { emptyOutDir: true }, plugins: [{ name: 'redirect', outputOptions: (x) => ({ ...x, dir: ${JSON.stringify(o)} }) }]`,
      'output.dir',
    ],
  ]

  for (const [name, config, setting] of routes) {
    it(`refuses ${name}, and leaves the owner's folder and outDir as they were`, async () => {
      const root = linkedSite('site')
      const parent = tempDir()
      const out = join(parent, 'dist')
      await buildSite({ root, outDir: out })
      const before = snapshot(out)
      const owner = tempDir()
      writeFileSync(join(owner, 'keep.txt'), 'owner')
      writeFileSync(join(owner, 'thesis.docx'), 'owner')
      writeFileSync(
        join(root, 'vite.config.ts'),
        `import { defineConfig } from 'vite'\nexport default defineConfig({ build: { rollupOptions: { external: ['weft-test-external'] } }, ${config(owner)} })\n`,
      )
      const failed = expectFailureLogs()
      await expect(buildSite({ root, outDir: out })).rejects.toThrow(sentence(setting))
      failed()
      expect(snapshot(owner)).toEqual({ 'keep.txt': Buffer.from('owner').toString('base64'), 'thesis.docx': Buffer.from('owner').toString('base64') })
      expect(snapshot(out)).toEqual(before)
      expect(readdirSync(parent)).toEqual(['dist'])
    })
  }
})

describe('buildSite with a plugin that changes the client build late', { timeout: 30_000 }, () => {
  const sentence = (setting: string) =>
    `buildSite: the site's Vite config or a plugin sets ${setting} for the client build; the page build alone decides where that goes, so remove it.`

  /** A copy of the site whose vite.config has this plugin besides the external, built once into outDir. */
  const withPlugin = async (plugin: (owner: string) => string) => {
    const root = linkedSite('site')
    const parent = tempDir()
    const out = join(parent, 'dist')
    await buildSite({ root, outDir: out })
    // The owner's folder beside outDir.
    const owner = join(parent, 'owner')
    mkdirSync(owner)
    writeFileSync(join(owner, 'keep.txt'), 'owner')
    writeFileSync(
      join(root, 'vite.config.ts'),
      `import { defineConfig } from 'vite'\nexport default defineConfig({ build: { rollupOptions: { external: ['weft-test-external'] } }, plugins: [${plugin(owner)}] })\n`,
    )
    return { root, parent, out, owner, before: snapshot(out) }
  }

  for (const [name, hook, setting] of [
    ['outDir and emptyOutDir', (o: string) => `this.environment.config.build.outDir = ${JSON.stringify(o)}; this.environment.config.build.emptyOutDir = true`, 'build.outDir'],
    ['emptyOutDir', () => 'this.environment.config.build.emptyOutDir = true', 'build.emptyOutDir'],
    ['copyPublicDir', () => 'this.environment.config.build.copyPublicDir = true', 'build.copyPublicDir'],
    ['publicDir', (o: string) => `this.environment.config.publicDir = ${JSON.stringify(o)}`, 'publicDir'],
  ] as const) {
    it(`refuses ${name} set in buildStart, and leaves the owner's folder and outDir as they were`, async () => {
      const { root, parent, out, owner, before } = await withPlugin((o) => `{ name: 'late', buildStart() { ${hook(o)} } }`)
      const failed = expectFailureLogs()
      await expect(buildSite({ root, outDir: out })).rejects.toThrow(new Error(sentence(setting)))
      failed()
      expect(snapshot(owner)).toEqual({ 'keep.txt': Buffer.from('owner').toString('base64') })
      expect(snapshot(out)).toEqual(before)
      expect(readdirSync(parent).sort()).toEqual(['dist', 'owner'])
    })
  }

  it('refuses a watching client build, in one sentence', async () => {
    const root = linkedSite('site')
    writeFileSync(
      join(root, 'vite.config.ts'),
      `import { defineConfig } from 'vite'\nexport default defineConfig({ build: { watch: {}, rollupOptions: { external: ['weft-test-external'] } } })\n`,
    )
    const parent = tempDir()
    await expect(buildSite({ root, outDir: join(parent, 'dist') })).rejects.toThrow(
      new Error(`buildSite: the site's Vite config or a plugin sets build.watch for the client build; the page build builds once, so remove it.`),
    )
    expect(readdirSync(parent)).toEqual([])
  })

  for (const [name, plugin] of [
    [
      "an environment's whole build",
      (o: string) => `{ name: 'late', buildStart() { this.environment.config.build = { ...this.environment.config.build, outDir: ${JSON.stringify(o)}, emptyOutDir: true } } }`,
    ],
    [
      "the top-level config's whole build",
      (o: string) =>
        `(() => { let c; return { name: 'late', configResolved(x) { c = x }, buildStart() { c.build = { ...c.build, outDir: ${JSON.stringify(o)}, emptyOutDir: true } } } })()`,
    ],
  ] as const) {
    it(`refuses ${name} replaced in buildStart, and leaves the owner's folder and outDir as they were`, async () => {
      const { root, parent, out, owner, before } = await withPlugin(plugin)
      const failed = expectFailureLogs()
      await expect(buildSite({ root, outDir: out })).rejects.toThrow(new Error(sentence('build')))
      failed()
      expect(snapshot(owner)).toEqual({ 'keep.txt': Buffer.from('owner').toString('base64') })
      expect(snapshot(out)).toEqual(before)
      expect(readdirSync(parent).sort()).toEqual(['dist', 'owner'])
    })
  }

  it('allows a setting to be assigned the value it has', async () => {
    const { root, out } = await withPlugin(
      () => `{ name: 'same', buildStart() { const c = this.environment.config; c.build.outDir = c.build.outDir; c.build.emptyOutDir = c.build.emptyOutDir; c.publicDir = c.publicDir; c.build = c.build } }`,
    )
    const manifest = await buildSite({ root, outDir: out })
    expect(manifest.files['/pages/counter/index.html']).toBeDefined()
  })

  it("gives the build's own error, not a refusal a plugin caught and went on from", async () => {
    const { root, out, owner, before } = await withPlugin(
      (o) =>
        `{ name: 'caught', buildStart() { try { this.environment.config.build.outDir = ${JSON.stringify(o)} } catch {} }, renderChunk() { throw new Error('the chunk is broken') } }`,
    )
    const failed = expectFailureLogs()
    const error = await buildSite({ root, outDir: out }).then(
      () => null,
      (e: Error) => e,
    )
    failed()
    expect(error?.message).toContain('the chunk is broken')
    expect(error?.message).not.toContain('buildSite: ')
    expect(snapshot(owner)).toEqual({ 'keep.txt': Buffer.from('owner').toString('base64') })
    expect(snapshot(out)).toEqual(before)
  })

  it('builds when a plugin caught a refusal and went on, into outDir alone', async () => {
    const { root, out, owner } = await withPlugin(
      (o) => `{ name: 'caught', buildStart() { try { this.environment.config.build.outDir = ${JSON.stringify(o)} } catch {} } }`,
    )
    const manifest = await buildSite({ root, outDir: out })
    expect(manifest.files['/pages/counter/index.html']).toBeDefined()
    expect(snapshot(owner)).toEqual({ 'keep.txt': Buffer.from('owner').toString('base64') })
  })

  it('checks the environment Vite builds: ssr, when build.ssr is set, and names it', async () => {
    const root = linkedSite('site')
    const parent = tempDir()
    const owner = join(parent, 'owner')
    mkdirSync(owner)
    writeFileSync(join(owner, 'keep.txt'), 'owner')
    writeFileSync(
      join(root, 'vite.config.ts'),
      `import { defineConfig } from 'vite'
export default defineConfig({
  build: { ssr: true, rollupOptions: { external: ['weft-test-external'] } },
  environments: { ssr: { build: { outDir: ${JSON.stringify(owner)}, emptyOutDir: true } } },
})
`,
    )
    const failed = expectFailureLogs()
    await expect(buildSite({ root, outDir: join(parent, 'dist') })).rejects.toThrow(
      new Error(sentence('build.outDir').replace('for the client build', 'for the ssr build')),
    )
    failed()
    expect(snapshot(owner)).toEqual({ 'keep.txt': Buffer.from('owner').toString('base64') })
    expect(readdirSync(parent)).toEqual(['owner'])
  })

  it("does not refuse another environment's output settings: only the client is built", async () => {
    const root = linkedSite('site')
    const elsewhere = tempDir()
    writeFileSync(join(elsewhere, 'keep.txt'), 'owner')
    writeFileSync(
      join(root, 'vite.config.ts'),
      `import { defineConfig } from 'vite'
export default defineConfig({
  build: { rollupOptions: { external: ['weft-test-external'] } },
  environments: { ssr: { build: { outDir: ${JSON.stringify(elsewhere)}, emptyOutDir: true } } },
})
`,
    )
    const out = join(tempDir(), 'dist')
    const manifest = await buildSite({ root, outDir: out })
    expect(manifest.files['/pages/counter/index.html']).toBeDefined()
    expect(readdirSync(elsewhere)).toEqual(['keep.txt'])
  })
})

describe('removeBuilt', { timeout: 30_000 }, () => {
  it('deletes only the listed files that still match, then their emptied folders', async () => {
    const out = join(tempDir(), 'dist')
    await buildSite({ root: fixture('static-site'), outDir: out })
    const { files: listed } = JSON.parse(readFileSync(join(out, '.weft-build'), 'utf8')) as { files: { path: string; size: number; sha256: string }[] }
    // Between the check and the delete, the owner adds a file and changes one.
    writeFileSync(join(out, 'pages/index/owner.txt'), 'owner')
    writeFileSync(join(out, 'routes.json'), 'OWNER EDIT')
    expect(await removeBuilt(out, listed)).toBe(false)
    expect(snapshot(out)).toEqual({
      '.weft-build': readFileSync(join(out, '.weft-build'), 'base64'),
      pages: 'dir',
      'pages/index': 'dir',
      'pages/index/owner.txt': Buffer.from('owner').toString('base64'),
      'routes.json': Buffer.from('OWNER EDIT').toString('base64'),
    })
  })

  it('removes the whole folder when everything in it is as listed', async () => {
    const parent = tempDir()
    const out = join(parent, 'dist')
    await buildSite({ root: fixture('static-site'), outDir: out })
    const { files: listed } = JSON.parse(readFileSync(join(out, '.weft-build'), 'utf8')) as { files: { path: string; size: number; sha256: string }[] }
    expect(await removeBuilt(out, listed)).toBe(true)
    expect(readdirSync(parent)).toEqual([])
  })
})

describe('buildSite with public files', { timeout: 30_000 }, () => {
  /** Build a copy of the static site with these public files; return its outDir's parent. */
  const withPublic = (make: (pub: string) => void) => {
    const root = linkedSite('static-site')
    mkdirSync(join(root, 'public'))
    make(join(root, 'public'))
    return { root, parent: tempDir() }
  }
  const clash = async (root: string, parent: string, message: string) => {
    await expect(buildSite({ root, outDir: join(parent, 'dist') })).rejects.toThrow(message)
    expect(readdirSync(parent)).toEqual([])
  }

  it('refuses one that would overwrite a generated file, and leaves no staging folder', async () => {
    await clash(fixture('public-clash'), tempDir(), 'buildSite: public/routes.json would overwrite the generated routes.json; rename or move it.')
  })

  it('refuses one named like the compressed copy of another', async () => {
    const { root, parent } = withPublic((pub) => {
      writeFileSync(join(pub, 'x.txt'), 'hello')
      writeFileSync(join(pub, 'x.txt.gz'), 'owner gz')
    })
    await clash(root, parent, 'buildSite: public/x.txt.gz would overwrite the generated x.txt.gz; rename or move it.')
  })

  it.runIf(process.platform === 'darwin' || process.platform === 'win32')(
    'refuses one that differs from a generated name only in case, where the file system ignores case',
    async () => {
      const upper = withPublic((pub) => writeFileSync(join(pub, 'ROUTES.json'), 'owner'))
      await clash(upper.root, upper.parent, 'buildSite: public/ROUTES.json would overwrite the generated routes.json; rename or move it.')
      const gz = withPublic((pub) => {
        writeFileSync(join(pub, 'x.txt'), 'hello')
        writeFileSync(join(pub, 'x.txt.GZ'), 'owner gz')
      })
      await clash(gz.root, gz.parent, 'buildSite: public/x.txt.GZ would overwrite the generated x.txt.gz; rename or move it.')
    },
  )

  it('refuses one named like the build marker', async () => {
    const { root, parent } = withPublic((pub) => writeFileSync(join(pub, '.weft-build'), '{}'))
    await clash(root, parent, 'buildSite: public/.weft-build would overwrite the generated .weft-build; rename or move it.')
  })

  it('copies a symlink to a file inside public/ as that file, and refuses one that leads out', async () => {
    const inside = withPublic((pub) => {
      mkdirSync(join(pub, 'real'))
      writeFileSync(join(pub, 'real/a.txt'), 'inside')
      symlinkSync(join(pub, 'real/a.txt'), join(pub, 'a.txt'))
    })
    const out = join(inside.parent, 'dist')
    await buildSite({ root: inside.root, outDir: out })
    expect(lstatSync(join(out, 'a.txt')).isFile()).toBe(true)
    expect(readFileSync(join(out, 'a.txt'), 'utf8')).toBe('inside')

    const secret = join(tempDir(), 'secret.txt')
    writeFileSync(secret, 'secret')
    const outside = withPublic((pub) => symlinkSync(secret, join(pub, 'leak.txt')))
    await clash(
      outside.root,
      outside.parent,
      'buildSite: public/leak.txt is a symlink that does not lead to a file inside public; put the file itself there.',
    )
  })
})

describe('buildSite refuses', { timeout: 30_000 }, () => {

  // Each case works on a copy of a fixture, so that a build which failed to
  // refuse could not write into, or delete from, the fixtures.
  const site = () => {
    const root = join(tempDir(), 'site')
    cpSync(fixture('static-site'), root, { recursive: true })
    return root
  }
  const refused = async (root: string, outDir: string, fixtureFiles = ['env.d.ts', 'pages/index.vue', 'vite.config.ts']) => {
    await expect(buildSite({ root, outDir })).rejects.toThrow(refusal(outDir))
    expect(files(root)).toEqual(fixtureFiles)
  }

  it('an outDir that is the site root, contains it, or is one of its source folders', async () => {
    const root = site()
    const sources = ['pages', 'public', 'components', 'src', 'node_modules', 'pages/blog'].map((d) => join(root, d))
    for (const outDir of [root, join(root, '..'), ...sources]) await refused(root, outDir)
  })

  it('an outDir holding .vue files the build uses, writing nothing', async () => {
    const root = linkedSite('static-site')
    mkdirSync(join(root, 'lib'))
    writeFileSync(join(root, 'lib/Note.vue'), '<script setup vapor lang="ts"></script>\n<template><p>note</p></template>\n')
    writeFileSync(
      join(root, 'pages/note.vue'),
      '<script setup vapor lang="ts">\nimport Note from \'../lib/Note.vue\'\n</script>\n<template><Note /></template>\n',
    )
    await expect(buildSite({ root, outDir: join(root, 'lib') })).rejects.toThrow(refusal(join(root, 'lib')))
    expect(readdirSync(join(root, 'lib'))).toEqual(['Note.vue'])
  })

  it('an outDir that reaches the site root through a symlink', async () => {
    const root = site()
    const link = join(root, '..', 'link')
    symlinkSync(root, link)
    await refused(root, link)
    await refused(root, join(link, 'pages'))
  })

  it('an outDir inside the folder a symlinked source folder leads to', async () => {
    for (const dir of ['public', 'pages', 'components']) {
      const root = linkedSite('static-site')
      const shared = tempDir()
      writeFileSync(join(shared, 'keep.txt'), 'owner')
      if (dir === 'pages') {
        renameSync(join(root, 'pages'), join(shared, 'pages'))
        symlinkSync(join(shared, 'pages'), join(root, 'pages'))
        for (const outDir of [join(shared, 'pages'), join(shared, 'pages/out')]) {
          await expect(buildSite({ root, outDir })).rejects.toThrow(refusal(outDir))
        }
      } else {
        symlinkSync(shared, join(root, dir))
        for (const outDir of [shared, join(shared, 'out')]) {
          await expect(buildSite({ root, outDir })).rejects.toThrow(refusal(outDir))
        }
      }
      expect(readdirSync(shared).sort(), dir).toEqual(dir === 'pages' ? ['keep.txt', 'pages'] : ['keep.txt'])
    }
  })

  it.runIf(process.platform === 'darwin')(
    'an outDir that names the root in another Unicode form, where the file system ignores it',
    async () => {
      const parent = tempDir()
      const root = join(parent, 'caf\u00e9')
      cpSync(fixture('static-site'), root, { recursive: true })
      const nfd = join(parent, 'cafe\u0301')
      await refused(root, nfd)
      await refused(root, join(nfd, 'pages'))
      await refused(nfd, join(root, 'pages'))
    },
  )

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

describe('buildSite without pages', () => {
  it('refuses a site with no pages folder, in one sentence', async () => {
    const root = tempDir()
    await expect(buildSite({ root, outDir: join(root, 'dist') })).rejects.toThrow(
      new Error(`buildSite: ${join(root, 'pages')} does not exist; a site's pages are .vue files in it, like pages/index.vue.`),
    )
    writeFileSync(join(root, 'pages'), 'not a folder')
    await expect(buildSite({ root, outDir: join(root, 'dist') })).rejects.toThrow(
      new Error(`buildSite: ${join(root, 'pages')} is not a folder; a site's pages are .vue files in it, like pages/index.vue.`),
    )
    expect(readdirSync(root)).toEqual(['pages'])
  })
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
    // As a site runs it: no NODE_ENV (vitest sets "test").
    const env = { ...process.env, NODE_ENV: undefined }
    const { stdout } = await promisify(execFile)(process.execPath, [bin, '--root', fixture('site'), '--out', out], { env })
    expect(stdout).toBe(`weft: built 8 pages into ${out}\n`)
    const js = files(out).filter((f) => f.endsWith('.js'))
    expect(js.length).toBeGreaterThan(0)
    // Vue's development build carries its warnings; the production build does not.
    for (const f of js) expect(readFileSync(join(out, f), 'utf8').includes('[Vue warn]'), f).toBe(false)
  })

  it('copies the app build from --app-dir', async () => {
    const out = join(tempDir(), 'dist')
    const app = tempDir()
    writeFileSync(join(app, 'index.html'), '<!doctype html><div id="app"></div>')
    const bin = fileURLToPath(new URL('../bin/weft-build-pages.mjs', import.meta.url))
    const args = [bin, '--root', fixture('static-site'), '--out', out, '--app-prefix', '/app', '--app-dir', app]
    await promisify(execFile)(process.execPath, args)
    expect(readFileSync(join(out, 'app/index.html'), 'utf8')).toBe('<!doctype html><div id="app"></div>')
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
      stderr:
        'weft-build-pages: --out is required.\n' +
        'usage: weft-build-pages --root <site> --out <dist> [--app-prefix /app [--app-dir <app build>]] [--theme neutral|lisa|weft] [--preload-font <file.woff2>]...\n',
    })
  })
})

describe('buildSite on a page that imports weft.css', () => {
  it('refuses it in one sentence: a page gets the CSS of what it renders', async () => {
    const out = tempDir()
    const error = await buildSite({ root: fixture('weft-css-import'), outDir: join(out, 'dist') }).catch((e: Error) => e)
    expect(error).toBeInstanceOf(Error)
    const message = (error as Error).message
    expect(message).toMatch(/^renderPage: pages\/index\.vue imports .*weft\.css, which holds every Weft component's CSS; /)
    expect(message).toContain('remove the import')
    expect(message.trim().split(/(?<=\.) /)).toHaveLength(1)
    expect(existsSync(join(out, 'dist'))).toBe(false)
  })
})

describe('cssTargets', () => {
  it("converts Vite's build.cssTarget to Lightning CSS targets as Vite does", () => {
    expect(cssTargets(['chrome111', 'edge111', 'firefox114', 'safari16.4', 'ios16.4'])).toEqual({
      chrome: 111 << 16,
      edge: 111 << 16,
      firefox: 114 << 16,
      safari: (16 << 16) | (4 << 8),
      ios_saf: (16 << 16) | (4 << 8),
    })
    expect(cssTargets(['safari15', 'safari16.4'])).toEqual({ safari: 15 << 16 })
    expect(cssTargets(false)).toBeUndefined()
    expect(cssTargets('esnext')).toBeUndefined()
    expect(cssTargets(['node20', 'chrome120'])).toEqual({ chrome: 120 << 16 })
  })

  it('reads a JavaScript language target (es2020, the default cssTarget of build.target) as no browser constraint', () => {
    expect(cssTargets('es2020')).toBeUndefined()
    expect(cssTargets(['es2015', 'esnext'])).toBeUndefined()
    expect(cssTargets(['es2020', 'safari16.4'])).toEqual({ safari: (16 << 16) | (4 << 8) })
  })

  it('refuses a target that names no browser, in one sentence', () => {
    expect(() => cssTargets('netscape4')).toThrow(/^buildSite: build\.cssTarget "netscape4" is not a browser version; name browsers, like "safari16\.4"\.$/)
  })

  it('builds a site whose Vite config sets build.target to es2020', { timeout: 60_000 }, async () => {
    const root = linkedSite('site')
    writeFileSync(
      join(root, 'vite.config.ts'),
      `import { defineConfig } from 'vite'\nexport default defineConfig({ build: { target: 'es2020', rollupOptions: { external: ['weft-test-external'] } } })\n`,
    )
    const out = tempDir()
    const manifest = await buildSite({ root, outDir: out })
    expect(existsSync(join(out, 'pages', 'index', 'index.html'))).toBe(true)
    expect(Object.keys(manifest).length).toBeGreaterThan(0)
  })
})
