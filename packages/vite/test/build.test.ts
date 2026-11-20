import { execFile } from 'node:child_process'
import { createHash } from 'node:crypto'
import {
  cpSync,
  existsSync,
  lstatSync,
  mkdirSync,
  mkdtempSync,
  readdirSync,
  readFileSync,
  readlinkSync,
  renameSync,
  rmSync,
  symlinkSync,
  writeFileSync,
} from 'node:fs'
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
function linkedSite(name: string): string {
  const root = join(tempDir(), name)
  cpSync(fixture(name), root, { recursive: true })
  symlinkSync(fileURLToPath(new URL('../node_modules', import.meta.url)), join(root, 'node_modules'))
  return root
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
    expect(JSON.parse(read('.weft-build'))).toEqual({ weft: 1, files: files(out).filter((f) => f !== '.weft-build') })
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
    expect(JSON.parse(readFileSync(join(out, '.weft-build'), 'utf8')).files).toEqual(files(out).filter((f) => f !== '.weft-build'))
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
    for (const marker of ['not json', '{"weft":2,"files":[]}', '{"weft":1,"files":["../x"]}', '{"weft":1}']) {
      const out = tempDir()
      writeFileSync(join(out, '.weft-build'), marker)
      await refusedOver(out, 'has a .weft-build that Weft cannot read')
    }
  })

  it('leaves outDir as it was, and no staging folder, when the build fails', async () => {
    const root = linkedSite('static-site')
    const parent = tempDir()
    const out = join(parent, 'dist')
    await buildSite({ root, outDir: out })
    const before = snapshot(out)
    // A page that throws as it renders.
    writeFileSync(join(root, 'pages/boom.vue'), '<script setup vapor lang="ts">\nthrow new Error(\'boom\')\n</script>\n<template><p /></template>\n')
    await expect(buildSite({ root, outDir: out })).rejects.toThrow('boom')
    rmSync(join(root, 'pages/boom.vue'))
    // An island the client build cannot bundle: it imports a module that does not exist.
    mkdirSync(join(root, 'components'))
    writeFileSync(
      join(root, 'components/Broken.vue'),
      "<script setup vapor lang=\"ts\">\nconst load = () => import('weft-no-such-module')\n</script>\n<template><button type=\"button\" @click=\"load\">x</button></template>\n",
    )
    writeFileSync(
      join(root, 'pages/broken.vue'),
      "<script setup vapor lang=\"ts\">\nimport { Island } from '@weft/ui'\nimport Broken from '../components/Broken.vue'\n</script>\n<template><Island :component=\"Broken\" /></template>\n",
    )
    await expect(buildSite({ root, outDir: out })).rejects.toThrow('weft-no-such-module')
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

  it('an outDir in a folder holding .vue files the build uses, writing nothing', async () => {
    const root = linkedSite('static-site')
    mkdirSync(join(root, 'lib'))
    writeFileSync(join(root, 'lib/Note.vue'), '<script setup vapor lang="ts"></script>\n<template><p>note</p></template>\n')
    writeFileSync(
      join(root, 'pages/note.vue'),
      '<script setup vapor lang="ts">\nimport Note from \'../lib/Note.vue\'\n</script>\n<template><Note /></template>\n',
    )
    for (const outDir of [join(root, 'lib'), join(root, 'lib/out')]) {
      await expect(buildSite({ root, outDir })).rejects.toThrow(refusal(outDir))
    }
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
        'usage: weft-build-pages --root <site> --out <dist> [--app-prefix /app [--app-dir <app build>]] [--theme neutral|lisa]\n',
    })
  })
})
