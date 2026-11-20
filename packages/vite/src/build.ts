import { createHash } from 'node:crypto'
import { existsSync, lstatSync, realpathSync } from 'node:fs'
import { copyFile, mkdir, readFile, readdir, rm, rmdir, writeFile } from 'node:fs/promises'
import { createRequire } from 'node:module'
import { basename, dirname, extname, join, relative, resolve, sep } from 'node:path'
import { fileURLToPath } from 'node:url'
import { brotliCompressSync, constants as zlib, gzipSync } from 'node:zlib'
import { build, isCSSRequest, mergeConfig, type Plugin, type Rolldown } from 'vite'
import { siteConfig } from './config.ts'
import { weft } from './plugin.ts'
import { createRenderServer, renderPage, type PageHead, type RenderResult } from './render.ts'
import { discoverRoutes, expandRoutes, type ExpandedRoute, type PageModule, type RoutesJson } from './routes.ts'

export type Theme = 'neutral' | 'lisa'

export interface BuildOptions {
  /** The site's root directory; its pages are `<root>/pages/**.vue`. */
  root: string
  /**
   * Where the build goes (the `dist` the Go binary embeds). The build removes
   * the files its previous build there listed in `manifest.json` (keeping `app/`
   * when `appPrefix` is set) and nothing else. It may not be the site root, be
   * inside it, or contain it.
   */
  outDir: string
  /** The client-rendered app's path prefix, e.g. `/app`; its build must already be in `<outDir>/app`. */
  appPrefix?: string
  /** The theme stylesheet the pages use. Default `neutral`. */
  theme?: Theme
  /** A page's CSS up to this many bytes is inlined; more is linked. Default 14 KB. */
  cssInlineLimit?: number
}

/** One served file in `manifest.json`. */
export interface ManifestFile {
  /** Strong ETag, quotes included: the first 16 hex digits of the SHA-256 of the bytes. */
  etag: string
  type: string
  /** Whether `<file>.br` / `<file>.gz` exist beside it. */
  br: boolean
  gz: boolean
  /** Hashed file name: cache forever. */
  immutable: boolean
}

/**
 * `manifest.json`: every served file by its URL path, and the Early Hints
 * `Link` values per page path (`"404"` for the not-found page).
 */
export interface Manifest {
  files: Record<string, ManifestFile>
  hints: Record<string, string[]>
}

const SPECULATION_RULES =
  '<script type="speculationrules">{"prefetch":[{"where":{"href_matches":"/*"},"eagerness":"moderate"}]}</script>'
const VIEW_TRANSITION = '@view-transition { navigation: auto; }'
const THEMES: Theme[] = ['neutral', 'lisa']
const CLIENT_ENTRY = fileURLToPath(new URL('./client.ts', import.meta.url))

const TYPES: Record<string, string> = {
  '.html': 'text/html; charset=utf-8',
  '.css': 'text/css; charset=utf-8',
  '.js': 'text/javascript; charset=utf-8',
  '.mjs': 'text/javascript; charset=utf-8',
  '.json': 'application/json',
  '.webmanifest': 'application/manifest+json',
  '.svg': 'image/svg+xml',
  '.txt': 'text/plain; charset=utf-8',
  '.xml': 'application/xml',
  '.woff': 'font/woff',
  '.woff2': 'font/woff2',
  '.ttf': 'font/ttf',
  '.wasm': 'application/wasm',
  '.png': 'image/png',
  '.jpg': 'image/jpeg',
  '.jpeg': 'image/jpeg',
  '.gif': 'image/gif',
  '.webp': 'image/webp',
  '.avif': 'image/avif',
  '.ico': 'image/x-icon',
}
/** Files stored with .br and .gz siblings. */
const TEXT = new Set(['.html', '.css', '.js', '.mjs', '.json', '.webmanifest', '.svg', '.txt', '.xml'])

/** The Content-Type a file is served with, from its extension. */
export const contentType = (file: string) => TYPES[extname(file).toLowerCase()] ?? 'application/octet-stream'

const sha256 = (bytes: Buffer | string) => createHash('sha256').update(bytes).digest('hex')
const posix = (p: string) => p.split('\\').join('/')

/**
 * Build a site's pages: render every page to HTML with its CSS inlined (or
 * linked when large), build the island runtime and island chunks when some page
 * has an island, pre-compress every text file with brotli and gzip, and write
 * `routes.json` and `manifest.json`, which the Go server serves from.
 */
export async function buildSite(opts: BuildOptions): Promise<Manifest> {
  const root = resolve(opts.root)
  const outDir = resolve(opts.outDir)
  if (overlaps(realish(outDir), realish(root))) {
    throw new Error(
      `buildSite: outDir ${opts.outDir} is the site root, inside it, or contains it; build into a directory of its own.`,
    )
  }
  const theme = opts.theme ?? 'neutral'
  if (!THEMES.includes(theme)) throw new Error(`buildSite: unknown theme "${theme}"; use ${THEMES.join(' or ')}.`)
  const app = opts.appPrefix === undefined ? null : appRoute(opts.appPrefix, outDir)
  const previous = await previousBuild(outDir, app !== null)
  const { css: baseCSS, files: baseFiles } = await readStyles(root, theme)
  const routes = await discoverRoutes(join(root, 'pages'))

  // A production build, as `vite build` makes one: the first Vite server sets
  // NODE_ENV for the whole process, so it must not be the dev server's default.
  process.env.NODE_ENV ??= 'production'

  // One server renders every page, and loads the dynamic pages for getPaths().
  const pages: (ExpandedRoute & RenderResult)[] = []
  const server = await createRenderServer(root, 'production')
  try {
    const load = (file: string) => server.ssrLoadModule(`/pages/${file}`) as Promise<PageModule>
    for (const route of await expandRoutes(routes, load)) {
      const page = join('pages', route.file)
      const rendered = await renderPage({ root, page, props: route.params, server, providedCSS: baseFiles })
      pages.push({ ...route, ...rendered })
    }
  } finally {
    await server.close()
  }

  // Every page rendered, so the previous build can go.
  await removeFiles(outDir, previous)

  const written = new Set<string>()
  const immutable = new Set<string>()
  const write = async (file: string, data: string | Buffer) => {
    await mkdir(dirname(join(outDir, file)), { recursive: true })
    await writeFile(join(outDir, file), data)
    written.add(file)
  }

  const islandFiles = [...new Set(pages.flatMap((p) => Object.values(p.islandFiles)))]
  const client = islandFiles.length ? await buildClient(root, outDir, islandFiles) : null
  for (const file of client?.files ?? []) {
    written.add(file)
    immutable.add(file)
  }

  const hints: Manifest['hints'] = {}
  const routesJson: RoutesJson = { pages: [], notFound: null, app }
  for (const page of pages) {
    // The theme, weft.css, the page's component CSS, then view transitions.
    const css = [baseCSS, page.css, VIEW_TRANSITION].filter(Boolean).join('\n') + '\n'
    let stylesheet: string | null = null
    if (Buffer.byteLength(css) > (opts.cssInlineLimit ?? 14 * 1024)) {
      const file = `assets/weft-${sha256(css).slice(0, 8)}.css`
      if (!written.has(file)) await write(file, css)
      immutable.add(file)
      stylesheet = '/' + file
    }
    const scripts = page.islands.length > 0 ? client : null
    const file = htmlFile(page.path)
    await write(file, pageHTML({ body: page.html, head: page.head, css, stylesheet, scripts }))

    const links = [
      ...(stylesheet ? [`<${stylesheet}>; rel=preload; as=style`] : []),
      ...(scripts ? [scripts.entry, ...scripts.imports].map((m) => `<${m}>; rel=modulepreload`) : []),
    ]
    const notFound = page.file === '404.vue'
    if (links.length) hints[notFound ? '404' : page.path] = links
    if (notFound) routesJson.notFound = file
    else routesJson.pages.push({ path: page.path, html: file, islands: page.islands })
  }
  await write('routes.json', JSON.stringify(routesJson, null, 2) + '\n')
  if (app) for (const file of await filesUnder(outDir, 'app')) written.add(file)
  await copyPublic(root, outDir, written)

  const files: Manifest['files'] = {}
  for (const file of [...written].sort()) {
    const bytes = await readFile(join(outDir, file))
    const text = TEXT.has(extname(file).toLowerCase())
    if (text) {
      const br = brotliCompressSync(bytes, {
        params: {
          [zlib.BROTLI_PARAM_QUALITY]: 11,
          [zlib.BROTLI_PARAM_MODE]: zlib.BROTLI_MODE_TEXT,
          [zlib.BROTLI_PARAM_SIZE_HINT]: bytes.length,
        },
      })
      await writeFile(join(outDir, file + '.br'), br)
      await writeFile(join(outDir, file + '.gz'), gzipSync(bytes, { level: 9 }))
    }
    files['/' + file] = {
      etag: `"${sha256(bytes).slice(0, 16)}"`,
      type: contentType(file),
      br: text,
      gz: text,
      immutable: immutable.has(file),
    }
  }
  const manifest: Manifest = { files, hints }
  await writeFile(join(outDir, 'manifest.json'), JSON.stringify(manifest, null, 2) + '\n')
  return manifest
}

const foldCase = process.platform === 'darwin' || process.platform === 'win32'

/** The real path of `p` (symlinks resolved), through its nearest existing ancestor if it does not exist. */
function realish(p: string): string {
  const rest: string[] = []
  for (let dir = resolve(p); ; dir = dirname(dir)) {
    try {
      return join(realpathSync(dir), ...rest)
    } catch {
      if (dirname(dir) === dir) return resolve(p)
      rest.unshift(basename(dir))
    }
  }
}

const segments = (p: string) => (foldCase ? p.toLowerCase() : p).split(sep).filter(Boolean)

/** Whether `inner` is `outer` or inside it, by whole segments; case-insensitive where the file system is. */
function contains(outer: string, inner: string): boolean {
  const [o, i] = [segments(outer), segments(inner)]
  return o.length <= i.length && o.every((s, n) => s === i[n])
}

/** Whether one path is the other or inside it. */
const overlaps = (a: string, b: string) => contains(a, b) || contains(b, a)

/** A manifest path that stays inside outDir: `/` then segments, none empty, `.` or `..`. */
const safePath = (p: string) =>
  p.startsWith('/') && !/[\\\0]/.test(p) && p.slice(1).split('/').every((s) => s !== '' && s !== '.' && s !== '..')

/**
 * The files the previous build in `outDir` wrote, from its `manifest.json`:
 * each listed file with its `.br` and `.gz`, then `routes.json` and the
 * manifest. Files under `app/` are the app's when `keepApp`. None when there is
 * no manifest; a manifest that is not Weft's stops the build.
 */
async function previousBuild(outDir: string, keepApp: boolean): Promise<string[]> {
  const file = join(outDir, 'manifest.json')
  if (!existsSync(file)) return []
  let listed: string[]
  try {
    const m = JSON.parse(await readFile(file, 'utf8')) as Partial<Manifest> | null
    const entries = Object.entries(m?.files ?? 0)
    const ok =
      typeof m?.hints === 'object' &&
      m.hints !== null &&
      entries.length > 0 &&
      entries.every(([k, v]) => safePath(k) && typeof (v as ManifestFile | null)?.etag === 'string')
    if (!ok) throw new Error()
    listed = entries.map(([k]) => k.slice(1))
  } catch {
    throw new Error(`buildSite: ${file} is not a Weft build manifest; move it or choose another outDir.`)
  }
  const out = listed
    .filter((f) => !(keepApp && f.startsWith('app/')))
    .flatMap((f) => [f, f + '.br', f + '.gz'])
  return [...new Set([...out, 'routes.json', 'manifest.json'])]
}

/**
 * Remove `files` (relative to `dir`), then the directories that left empty,
 * deepest first, never `dir` itself. A file reached through a symlink that
 * leads out of `dir`, or a directory where a file was, is left alone.
 */
async function removeFiles(dir: string, files: string[]) {
  const real = realish(dir)
  const dirs = new Set<string>()
  for (const file of files) {
    const path = join(dir, file)
    const stat = lstatSync(path, { throwIfNoEntry: false })
    if (!stat || stat.isDirectory() || !contains(real, realish(dirname(path)))) continue
    await rm(path, { force: true })
    for (let d = dirname(file); d !== '.'; d = dirname(d)) dirs.add(d)
  }
  const deepest = [...dirs].sort((a, b) => b.split('/').length - a.split('/').length)
  for (const d of deepest) {
    try {
      await rmdir(join(dir, d))
    } catch (e) {
      const code = (e as NodeJS.ErrnoException).code
      if (code !== 'ENOTEMPTY' && code !== 'EEXIST' && code !== 'ENOENT') throw e
    }
  }
}

/** Copy `<root>/public` to the root of the build; a file the build made itself is not overwritten. */
async function copyPublic(root: string, outDir: string, written: Set<string>) {
  const generated = new Set(['manifest.json'])
  for (const f of written) generated.add(f).add(f + '.br').add(f + '.gz')
  for (const file of await filesUnder(root, 'public', { all: true })) {
    const to = file.slice('public/'.length)
    if (generated.has(to)) throw new Error(`buildSite: ${file} would overwrite the generated ${to}; rename or move it.`)
    await mkdir(dirname(join(outDir, to)), { recursive: true })
    await copyFile(join(root, file), join(outDir, to))
    written.add(to)
  }
}

function appRoute(prefix: string, outDir: string): RoutesJson['app'] {
  if (!/^\/[^/]/.test(prefix) || prefix.endsWith('/')) {
    throw new Error(`buildSite: appPrefix "${prefix}" must start with "/" and not end with one, like "/app".`)
  }
  if (!existsSync(join(outDir, 'app/index.html'))) {
    throw new Error(`buildSite: appPrefix is ${prefix} but ${join(outDir, 'app/index.html')} is missing; build the app first.`)
  }
  return { prefix, index: 'app/index.html' }
}

/**
 * The files under `<base>/<dir>` as posix paths relative to `base`; none when
 * it does not exist. Unless `all`, `.br` and `.gz` files are left out: in a
 * build directory they are the compressed variants an earlier build wrote.
 */
async function filesUnder(base: string, dir: string, { all = false } = {}): Promise<string[]> {
  if (!existsSync(join(base, dir))) return []
  const entries = await readdir(join(base, dir), { recursive: true, withFileTypes: true })
  return entries
    .filter((e) => e.isFile() && (all || !/\.(br|gz)$/.test(e.name)))
    .map((e) => posix(relative(base, join(e.parentPath, e.name))))
    .sort()
}

/** The theme stylesheet, then `weft.css`: the CSS every page starts with, and their files. */
async function readStyles(root: string, theme: Theme): Promise<{ css: string; files: string[] }> {
  const read = async (from: string, spec: string, pkg: string) => {
    try {
      const file = createRequire(from).resolve(spec)
      return [file, await readFile(file, 'utf8')] as const
    } catch (e) {
      const why = e instanceof Error ? e.message.split('\n')[0] : String(e)
      throw new Error(`buildSite: cannot read ${spec} (${why}); run pnpm --filter ${pkg} build.`)
    }
  }
  const [uiFile, ui] = await read(join(root, 'package.json'), '@weft/ui/weft.css', '@weft/ui')
  // The theme comes from @weft/ui's own @weft/themes, the version its components expect.
  const [themeFile, themeCss] = await read(uiFile, `@weft/themes/${theme}.css`, '@weft/themes')
  return { css: `${themeCss}\n${ui}`, files: [themeFile, uiFile] }
}

/** `/` → `pages/index/index.html`, `/blog/a` → `pages/blog/a/index.html`. */
function htmlFile(path: string): string {
  const segments = path === '/' ? ['index'] : path.slice(1).split('/')
  if (segments.some((s) => s === '' || s === '.' || s === '..' || s.includes('\\'))) {
    throw new Error(`buildSite: page path ${path} has an empty, "." or ".." segment.`)
  }
  return `pages/${segments.join('/')}/index.html`
}

interface ClientBuild {
  /** URL of the runtime entry, the page's one module script. */
  entry: string
  /** URLs of the chunks the entry imports statically. */
  imports: string[]
  /** Every file the build wrote, relative to outDir. */
  files: string[]
}

/**
 * Island CSS is inlined into the pages that use the island, so the client build
 * carries none: every stylesheet loads empty, and Vite emits and preloads nothing.
 */
const noCSS: Plugin = {
  name: 'weft:no-client-css',
  enforce: 'pre',
  load(id) {
    if (isCSSRequest(id)) return { code: '' }
  },
}

/** Build the island runtime and one chunk per island into `<outDir>/assets`. */
async function buildClient(root: string, outDir: string, islands: string[]): Promise<ClientBuild> {
  const config = await siteConfig(root, { command: 'build', mode: 'production' }, {
    base: '/',
    logLevel: 'warn',
    publicDir: false,
    plugins: [noCSS, weft({ islands })],
    // The runtime lives in this package; its `vue` must be the site's.
    resolve: { dedupe: ['vue'] },
    build: {
      outDir,
      assetsDir: 'assets',
      emptyOutDir: false,
      copyPublicDir: false,
      write: true,
      manifest: false,
      ssrManifest: false,
      sourcemap: false,
      modulePreload: { polyfill: false },
      // With noCSS every island's CSS is empty; unsplit, no chunk imports or preloads a stylesheet.
      cssCodeSplit: false,
    },
  })
  // Only the runtime entry: an input of the site's config (an SPA's index.html)
  // would merge with it. The site's other options (external, output) stay.
  // Vite takes either name and prefers rolldownOptions; merge both into one.
  const b = (config.build ??= {})
  const options = mergeConfig(b.rollupOptions ?? {}, b.rolldownOptions ?? {}, false)
  b.rolldownOptions = b.rollupOptions = { ...options, input: { client: CLIENT_ENTRY } }
  const result = (await build(config)) as Rolldown.RolldownOutput | Rolldown.RolldownOutput[]
  const output = (Array.isArray(result) ? result : [result]).flatMap((r) => r.output)
  const entry = output.find((o): o is Rolldown.OutputChunk => o.type === 'chunk' && o.isEntry)
  if (!entry) throw new Error('buildSite: the client build produced no entry chunk.')
  return {
    entry: '/' + entry.fileName,
    imports: entry.imports.map((f) => '/' + f),
    files: output.map((o) => o.fileName),
  }
}

const escapeHTML = (s: string) =>
  s.replace(/[&<>"']/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' })[c]!)

/** `<title>`, description and Open Graph tags for what the page set with useHead. */
function headTags({ title, description, image }: PageHead): string[] {
  const meta = (attr: string, key: string, value: string | undefined) =>
    value === undefined ? [] : [`<meta ${attr}="${key}" content="${escapeHTML(value)}">`]
  return [
    ...(title === undefined ? [] : [`<title>${escapeHTML(title)}</title>`]),
    ...meta('name', 'description', description),
    ...meta('property', 'og:title', title),
    ...meta('property', 'og:description', description),
    ...meta('property', 'og:image', image),
  ]
}

function pageHTML(p: {
  body: string
  head: PageHead
  css: string
  stylesheet: string | null
  scripts: ClientBuild | null
}): string {
  return [
    '<!doctype html>',
    '<html lang="en">',
    '<head>',
    '<meta charset="utf-8">',
    '<meta name="viewport" content="width=device-width, initial-scale=1">',
    ...headTags(p.head),
    p.stylesheet === null ? `<style>${p.css}</style>` : `<link rel="stylesheet" href="${p.stylesheet}">`,
    SPECULATION_RULES,
    ...(p.scripts
      ? [
          ...p.scripts.imports.map((m) => `<link rel="modulepreload" href="${m}">`),
          `<script type="module" src="${p.scripts.entry}"></script>`,
        ]
      : []),
    '</head>',
    `<body>${p.body}</body>`,
    '</html>',
    '',
  ].join('\n')
}
