import { createHash } from 'node:crypto'
import { constants as fs, existsSync, realpathSync, type Dirent } from 'node:fs'
import { chmod, copyFile, lstat, mkdir, mkdtemp, readFile, readdir, realpath, rename, rm, rmdir, stat, unlink, writeFile } from 'node:fs/promises'
import { createRequire } from 'node:module'
import { basename, dirname, extname, join, relative, resolve, sep } from 'node:path'
import { fileURLToPath } from 'node:url'
import { brotliCompressSync, constants as zlib, gzipSync } from 'node:zlib'
import { build, isCSSRequest, mergeConfig, type Plugin, type ResolvedConfig, type Rolldown } from 'vite'
import { siteConfig } from './config.ts'
import { weft } from './plugin.ts'
import { createRenderServer, renderPage, type PageHead, type RenderResult } from './render.ts'
import { discoverRoutes, expandRoutes, type ExpandedRoute, type PageModule, type RoutesJson } from './routes.ts'

export type Theme = 'neutral' | 'lisa'

export interface BuildOptions {
  /** The site's root directory; its pages are `<root>/pages/**.vue`. */
  root: string
  /**
   * Where the build goes (the `dist` the Go binary embeds), e.g. `<root>/dist`.
   * The build is made in a staging folder beside it and then takes its place,
   * which it does only when outDir is missing, empty, or exactly what an earlier
   * build left there (its `.weft-build` lists every file with its size and
   * SHA-256, and nothing has changed); anything else is refused and left as it
   * is. It may not be the site root, contain it, or be in one of its source
   * folders.
   */
  outDir: string
  /**
   * The client-rendered app's path prefix, e.g. `/app`; its build is copied from
   * `appDir` into the folder the prefix names (`/admin` into `admin/`), so its
   * files are served at their own URLs.
   */
  appPrefix?: string
  /** Where the app's own build is, with its `index.html`; read, never changed. Default `<root>/app-dist`. */
  appDir?: string
  /** The theme stylesheet the pages use. Default `neutral`. */
  theme?: Theme
  /**
   * A page's CSS up to this many bytes, brotli-compressed as it is sent, is
   * inlined; more is linked. Default 14 KB, about what the first round trip
   * carries.
   */
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

/**
 * Prefetch same-site links as the pointer settles on them, except the API's and
 * the app's: an API call may change something, and the app is not a page.
 */
function speculationRules(app: RoutesJson['app']): string {
  const skip = ['/api', ...(app ? [app.prefix] : [])].flatMap((p) => [p, `${p}/*`])
  const rules = { prefetch: [{ where: { and: [{ href_matches: '/*' }, { not: { href_matches: skip } }] }, eagerness: 'moderate' }] }
  return `<script type="speculationrules">${JSON.stringify(rules)}</script>`
}
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

/**
 * Build a site's pages: render every page to HTML with its CSS inlined (or
 * linked when large), build the island runtime and island chunks when some page
 * has an island, pre-compress every text file with brotli and gzip, and write
 * `routes.json` and `manifest.json`, which the Go server serves from.
 */
export async function buildSite(opts: BuildOptions): Promise<Manifest> {
  const root = resolve(opts.root)
  const outDir = resolve(opts.outDir)
  const theme = opts.theme ?? 'neutral'
  if (!THEMES.includes(theme)) throw new Error(`buildSite: unknown theme "${theme}"; use ${THEMES.join(' or ')}.`)
  const app = opts.appPrefix === undefined ? null : appRoute(opts.appPrefix)
  const appDir = app ? resolve(opts.appDir ?? join(root, 'app-dist')) : null
  checkOutDir(outDir, root, appDir)
  const pagesDir = join(root, 'pages')
  const pagesStat = await stat(pagesDir).catch(() => null)
  if (!pagesStat?.isDirectory()) {
    throw new Error(
      `buildSite: ${pagesDir} ${pagesStat ? 'is not a folder' : 'does not exist'}; a site's pages are .vue files in it, like pages/index.vue.`,
    )
  }
  if (app && appDir && !existsSync(join(appDir, 'index.html'))) {
    throw new Error(
      `buildSite: appPrefix is ${app.prefix} but ${join(appDir, 'index.html')} is missing; build the app into ${appDir} first, or set appDir.`,
    )
  }
  const { css: baseCSS, files: baseFiles } = await readStyles(root, theme)
  const routes = await discoverRoutes(pagesDir)

  // A production build, as `vite build` makes one: the first Vite server sets
  // NODE_ENV for the whole process, so it must not be the dev server's default.
  process.env.NODE_ENV ??= 'production'

  // One server renders every page, and loads the dynamic pages for getPaths().
  const pages: (ExpandedRoute & RenderResult)[] = []
  const server = await createRenderServer(root, 'production')
  const vueFiles: string[] = []
  try {
    const load = (file: string) => server.ssrLoadModule(`/pages/${file}`) as Promise<PageModule>
    for (const route of await expandRoutes(routes, load)) {
      const page = join('pages', route.file)
      const rendered = await renderPage({ root, page, props: route.params, server, providedCSS: baseFiles })
      pages.push({ ...route, ...rendered })
    }
    for (const mod of server.environments.ssr.moduleGraph.idToModuleMap.values()) {
      if (mod.file?.endsWith('.vue')) vueFiles.push(mod.file)
    }
  } finally {
    await server.close()
  }
  // Nor may outDir hold, or be among, the .vue files the pages were built from.
  checkOutDir(outDir, root, appDir, vueFiles)
  // Refused before the client build and the writing; checked again at the swap.
  await replaceable(outDir)

  // Everything is written into a fresh folder beside outDir, which then takes its place.
  await mkdir(dirname(outDir), { recursive: true })
  const staging = await mkdtemp(join(dirname(outDir), `.${basename(outDir)}.weft-staging-`))
  // It becomes outDir: a folder anyone may read, not mkdtemp's owner-only one.
  await chmod(staging, 0o755)
  try {
    const manifest = await writeBuild(staging, { root, app, appDir, pages, baseCSS, cssInlineLimit: opts.cssInlineLimit })
    await swap(staging, outDir)
    return manifest
  } catch (e) {
    // Only the staging folder: it is this build's own. After a swap it is gone.
    await rm(staging, { recursive: true, force: true })
    throw e
  }
}

/** What goes into the build, besides where it goes. */
interface BuildInput {
  root: string
  app: RoutesJson['app']
  appDir: string | null
  pages: (ExpandedRoute & RenderResult)[]
  baseCSS: string
  cssInlineLimit: number | undefined
}

/** The marker of a folder Weft built: every other file in it, with its size and SHA-256. */
const MARKER = '.weft-build'

/** One file in the marker. */
interface MarkedFile {
  path: string
  size: number
  sha256: string
}

/** A name as the file system compares names: in NFC, and in one case where it ignores case. */
const fsKey = (p: string) => (foldCase ? p.normalize('NFC').toLowerCase() : p.normalize('NFC'))

/** The files of a build by `fsKey`: each one's name, and where it came from ('' for the build itself). */
type Taken = Map<string, { path: string; source: string }>
const take = (taken: Taken, path: string, source = '') => taken.set(fsKey(path), { path, source })

/**
 * Bytes of brotli at quality 11, as the build compresses what it serves. Pages
 * share CSS, so within one build each set is measured once.
 */
function sentSizes(): (css: string) => number {
  const sizes = new Map<string, number>()
  return (css) => {
    let size = sizes.get(css)
    if (size === undefined) {
      size = brotliCompressSync(css, { params: { [zlib.BROTLI_PARAM_QUALITY]: 11 } }).length
      sizes.set(css, size)
    }
    return size
  }
}

/** Write the whole build into `dir`, a fresh folder of its own, and mark it. */
async function writeBuild(dir: string, { root, app, appDir, pages, baseCSS, cssInlineLimit }: BuildInput): Promise<Manifest> {
  // Every file in the build; none is ever written twice, nor under two names the file system takes for one.
  const taken: Taken = new Map()
  take(taken, 'manifest.json')
  take(taken, MARKER)
  const immutable = new Set<string>()
  const write = async (file: string, data: string | Buffer) => {
    await mkdir(dirname(join(dir, file)), { recursive: true })
    await writeFile(join(dir, file), data, { flag: 'wx' })
    take(taken, file)
  }

  const islandFiles = [...new Set(pages.flatMap((p) => Object.values(p.islandFiles)))]
  const client = islandFiles.length ? await buildClient(root, dir, islandFiles) : null
  for (const file of client?.files ?? []) {
    take(taken, file)
    immutable.add(file)
  }

  const hints: Manifest['hints'] = {}
  const routesJson: RoutesJson = { pages: [], notFound: null, app }
  const rules = speculationRules(app)
  const sentSize = sentSizes()
  for (const page of pages) {
    // The theme, weft.css, the page's component CSS, then view transitions.
    const css = [baseCSS, page.css, VIEW_TRANSITION].filter(Boolean).join('\n') + '\n'
    let stylesheet: string | null = null
    if (sentSize(css) > (cssInlineLimit ?? 14 * 1024)) {
      const file = `assets/weft-${sha256(css).slice(0, 8)}.css`
      if (!taken.has(fsKey(file))) await write(file, css)
      immutable.add(file)
      stylesheet = '/' + file
    }
    const scripts = page.islands.length > 0 ? client : null
    const file = htmlFile(page.path)
    await write(file, pageHTML({ body: page.html, head: page.head, css, stylesheet, scripts, rules }))

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
  // The app's own .br/.gz would be stale beside the ones made here; public/ is copied as it is.
  if (app && appDir) {
    await copyTree(appDir, appDir, dir, app.prefix.slice(1), taken, { variants: false })
    // Vite names the app's built assets <name>-<hash>.<ext>: those are cached forever, as an SPA's are.
    const assets = `${app.prefix.slice(1)}/assets/`
    for (const { path } of taken.values()) {
      if (path.startsWith(assets) && hashedName(path.slice(assets.length))) immutable.add(path)
    }
  }
  await copyTree(join(root, 'public'), 'public', dir, '', taken, { variants: true })

  // The .br and .gz the build makes may not be names already taken.
  const listed = [...taken.values()].map((t) => t.path).filter((f) => f !== 'manifest.json' && f !== MARKER).sort()
  for (const file of listed) {
    if (!TEXT.has(extname(file).toLowerCase())) continue
    for (const variant of [file + '.br', file + '.gz']) {
      const holder = taken.get(fsKey(variant))
      if (holder) throw new Error(`buildSite: ${holder.source || holder.path} would overwrite the generated ${variant}; rename or move it.`)
    }
  }

  const files: Manifest['files'] = {}
  for (const file of listed) {
    const bytes = await readFile(join(dir, file))
    const text = TEXT.has(extname(file).toLowerCase())
    if (text) {
      const br = brotliCompressSync(bytes, {
        params: {
          [zlib.BROTLI_PARAM_QUALITY]: 11,
          [zlib.BROTLI_PARAM_MODE]: zlib.BROTLI_MODE_TEXT,
          [zlib.BROTLI_PARAM_SIZE_HINT]: bytes.length,
        },
      })
      await writeFile(join(dir, file + '.br'), br, { flag: 'wx' })
      await writeFile(join(dir, file + '.gz'), gzipSync(bytes, { level: 9 }), { flag: 'wx' })
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
  await writeFile(join(dir, 'manifest.json'), JSON.stringify(manifest, null, 2) + '\n', { flag: 'wx' })
  // The marker lists what is really there, whoever wrote it (the client build included).
  const marked: MarkedFile[] = []
  for (const e of await walk(dir)) {
    if (e.kind !== 'file' || e.path === MARKER) continue
    const bytes = await readFile(join(dir, e.path))
    marked.push({ path: e.path, size: bytes.length, sha256: sha256(bytes) })
  }
  marked.sort((a, b) => (a.path < b.path ? -1 : 1))
  await writeFile(join(dir, MARKER), JSON.stringify({ weft: 1, files: marked }, null, 2) + '\n', { flag: 'wx' })
  return manifest
}

/**
 * Copy the files under `src` into `<dir>/<prefix>`, recording each in `taken`
 * under `label`. A symlink is copied as the file it leads to only when that is
 * a file inside `src`; anything else that is not a file or folder is refused.
 * Unless `variants`, a `.br`/`.gz` beside a text file is left out: the build
 * makes its own.
 */
async function copyTree(src: string, label: string, dir: string, prefix: string, taken: Taken, { variants = true }) {
  if (!existsSync(src)) return
  const base = canonical(src)
  const entries = await walk(src)
  const paths = new Set(entries.map((e) => e.path))
  for (const entry of entries) {
    if (entry.kind === 'dir') continue
    const shown = `${label}/${entry.path}`
    const path = join(src, entry.path)
    let source = path
    if (entry.kind === 'link') {
      const target = await realpath(path).catch(() => null)
      if (!target || !contains(base, canonical(target)) || !(await stat(target)).isFile()) {
        throw new Error(`buildSite: ${shown} is a symlink that does not lead to a file inside ${label}; put the file itself there.`)
      }
      source = target
    } else if (entry.kind === 'other') {
      throw new Error(`buildSite: ${shown} is not a file or a folder; move it out of ${label}.`)
    }
    const variantOf = /\.(br|gz)$/.test(entry.path) ? entry.path.slice(0, -3) : null
    if (!variants && variantOf && paths.has(variantOf) && TEXT.has(extname(variantOf).toLowerCase())) continue
    const to = prefix ? `${prefix}/${entry.path}` : entry.path
    const holder = taken.get(fsKey(to))
    if (holder) {
      const what = holder.source ? `${holder.path} from ${holder.source}` : `the generated ${holder.path}`
      throw new Error(`buildSite: ${shown} would overwrite ${what}; rename or move it.`)
    }
    await mkdir(dirname(join(dir, to)), { recursive: true })
    await copyFile(source, join(dir, to), fs.COPYFILE_EXCL)
    take(taken, to, shown)
  }
}

interface Entry {
  /** Posix path relative to the walked folder. */
  path: string
  kind: 'file' | 'dir' | 'link' | 'other'
}

const kindOf = (e: Dirent): Entry['kind'] =>
  e.isSymbolicLink() ? 'link' : e.isDirectory() ? 'dir' : e.isFile() ? 'file' : 'other'

/** Everything under `dir`, sorted, without following a symlink. */
async function walk(dir: string, prefix = ''): Promise<Entry[]> {
  const out: Entry[] = []
  const entries = (await readdir(join(dir, prefix), { withFileTypes: true })).sort((a, b) => (a.name < b.name ? -1 : 1))
  for (const e of entries) {
    const path = prefix ? `${prefix}/${e.name}` : e.name
    const kind = kindOf(e)
    out.push({ path, kind })
    if (kind === 'dir') out.push(...(await walk(dir, path)))
  }
  return out
}

/**
 * Whether a file name carries a content hash as Vite writes one: `name-<8 of
 * A-Z a-z 0-9 _ ->.ext`, the eight with a capital or a digit, so that a word
 * such as `app-settings.js` is not taken for one. A miss is only revalidated.
 */
const hashedName = (name: string) => {
  const hash = /^[^/]+-([A-Za-z0-9_-]{8})\.[A-Za-z0-9]+$/.exec(name)?.[1]
  return hash !== undefined && /[A-Z0-9]/.test(hash)
}

/** A relative path in a marker: segments, none empty, `.` or `..`, no backslash or NUL. */
const safePath = (p: unknown) =>
  typeof p === 'string' && !/[\\\0]/.test(p) && p.split('/').every((s) => s !== '' && s !== '.' && s !== '..')

const markedFile = (f: unknown): f is MarkedFile => {
  const m = f as Partial<MarkedFile> | null
  return (
    typeof m === 'object' &&
    m !== null &&
    safePath(m.path) &&
    Number.isSafeInteger(m.size) &&
    m.size! >= 0 &&
    typeof m.sha256 === 'string' &&
    /^[0-9a-f]{64}$/.test(m.sha256)
  )
}

/** Whether the build may take a folder's place, and for a built one, its files as named on disk. */
interface Replaceable {
  state: 'missing' | 'empty' | 'built'
  files: MarkedFile[]
}

/**
 * Whether the build may take `dir`'s place: `missing`, `empty`, or `built`
 * (its marker lists exactly the files in it, each with the size and SHA-256
 * it has, with no symlink and no other folder). Anything else is refused in
 * one sentence naming `shown`.
 */
async function replaceable(dir: string, shown = dir): Promise<Replaceable> {
  const refuse = (why: string, then = 'Weft replaces only an empty folder or one it built, so move that out or build into a new folder'): never => {
    throw new Error(`buildSite: outDir ${shown} ${why}; ${then}.`)
  }
  const foreign = (path: string): never =>
    basename(path) === '.DS_Store'
      ? refuse(`holds ${path} (made by Finder)`, 'remove it or build into a new folder')
      : refuse(`holds ${path}, which Weft did not build`)
  const self = await lstat(dir).catch((e: NodeJS.ErrnoException) => {
    if (e.code === 'ENOENT') return null
    throw e
  })
  if (!self) return { state: 'missing', files: [] }
  if (self.isSymbolicLink()) refuse('is a symlink')
  if (!self.isDirectory()) refuse('is not a folder')
  try {
    const top = (await readdir(dir)).sort()
    if (top.length === 0) return { state: 'empty', files: [] }
    // Without a marker, the first entry is the foreign one; the rest is not walked.
    const marker = await lstat(join(dir, MARKER)).catch(() => null)
    if (marker && !marker.isFile()) refuse(`has a ${MARKER} that is not a regular file`)
    if (!marker) foreign(top[0]!)
    let listed: MarkedFile[] = []
    try {
      const m = JSON.parse(await readFile(join(dir, MARKER), 'utf8')) as { weft?: unknown; files?: unknown } | null
      if (m?.weft !== 1 || !Array.isArray(m.files) || !m.files.every(markedFile)) throw new Error()
      listed = m.files
    } catch {
      refuse(`has a ${MARKER} that Weft cannot read`)
    }
    // Names are compared in one Unicode form, as file systems that ignore the form see them;
    // two entries whose names differ only in form are refused where the file system keeps both.
    const nfc = (p: string) => p.normalize('NFC')
    const want = new Map(listed.map((f) => [nfc(f.path), f]))
    const dirs = new Set(listed.flatMap((f) => f.path.split('/').slice(0, -1).map((_, n, s) => nfc(s.slice(0, n + 1).join('/')))))
    const found = new Map<string, string>()
    const seen = new Set<string>()
    for (const e of await walk(dir)) {
      const path = nfc(e.path)
      if (seen.has(path)) refuse(`holds ${e.path}, whose name another entry has in another Unicode form`)
      seen.add(path)
      if (e.kind === 'link') refuse(`holds the symlink ${e.path}`)
      if (path === MARKER) continue
      if (e.kind === 'dir' ? !dirs.has(path) : e.kind !== 'file' || !want.has(path)) foreign(e.path)
      if (e.kind === 'file') found.set(path, e.path)
    }
    const files: MarkedFile[] = []
    for (const f of listed) {
      const name = found.get(nfc(f.path))
      if (name === undefined) return refuse(`no longer holds ${f.path}, which Weft built`)
      const bytes = await readFile(join(dir, name))
      if (bytes.length !== f.size || sha256(bytes) !== f.sha256) refuse(`holds ${name}, changed since Weft built it`)
      files.push({ ...f, path: name })
    }
    return { state: 'built', files }
  } catch (e) {
    const { code, path } = e as NodeJS.ErrnoException
    if (code !== 'EACCES' && code !== 'EPERM') throw e
    const inside = path ? relative(dir, path).split(sep).join('/') : ''
    return refuse(inside ? `holds ${inside}, which Weft cannot read` : 'cannot be read')
  }
}

/**
 * Put the finished build in `staging` in place of `outDir`, if outDir is still
 * missing, empty or the earlier build. That earlier build is moved aside,
 * checked once more, and only then removed.
 */
async function swap(staging: string, outDir: string) {
  const { state } = await replaceable(outDir)
  if (state === 'empty') await rmdir(outDir)
  if (state !== 'built') return rename(staging, outDir)
  const old = await mkdtemp(join(dirname(outDir), `.${basename(outDir)}.weft-old-`))
  await rmdir(old)
  await rename(outDir, old)
  let files: MarkedFile[]
  try {
    ;({ files } = await replaceable(old, outDir))
    await rename(staging, outDir)
  } catch (e) {
    await rename(old, outDir)
    throw e
  }
  // The build is done by now, so a folder that will not go is reported, not fatal.
  if (!(await removeBuilt(old, files))) {
    console.warn(`buildSite: the build is in ${outDir}, but the earlier build moved to ${old} could not be removed; remove that folder by hand.`)
  }
}

/**
 * Delete the build in `dir` by its marker's list, `files` (paths as named on
 * disk): each file only if it still has its listed size and SHA-256, then the
 * marker, then the folders the list names, deepest first, and `dir` itself,
 * each only if it is empty. Anything else (a file added or changed since the
 * build was checked, one that will not go) stays. Never throws; reports
 * whether `dir` is gone.
 */
export async function removeBuilt(dir: string, files: MarkedFile[]): Promise<boolean> {
  let kept = false
  for (const f of files) {
    const path = join(dir, f.path)
    try {
      const bytes = await readFile(path)
      if (bytes.length === f.size && sha256(bytes) === f.sha256) await unlink(path)
      else kept = true
    } catch {
      kept = true
    }
  }
  if (!kept) await unlink(join(dir, MARKER)).catch(() => {})
  const dirs = [...new Set(files.flatMap((f) => f.path.split('/').slice(0, -1).map((_, n, s) => s.slice(0, n + 1).join('/'))))]
  dirs.sort((a, b) => b.split('/').length - a.split('/').length)
  for (const d of [...dirs, '']) await rmdir(join(dir, d)).catch(() => {})
  return !existsSync(dir)
}

const foldCase = process.platform === 'darwin' || process.platform === 'win32'

/**
 * `p` as the file system sees it, for comparing: symlinks resolved (through
 * its nearest existing ancestor if it does not exist), in one Unicode form, and
 * in one case where the file system ignores case.
 */
function canonical(p: string): string {
  const rest: string[] = []
  let real = resolve(p)
  for (let dir = real; ; dir = dirname(dir)) {
    try {
      real = join(realpathSync.native(dir), ...rest)
      break
    } catch {
      if (dirname(dir) === dir) break
      rest.unshift(basename(dir))
    }
  }
  const nfc = real.normalize('NFC')
  return foldCase ? nfc.toLowerCase() : nfc
}

/** Whether `inner` is `outer` or inside it, by whole segments of canonical paths. */
function contains(outer: string, inner: string): boolean {
  const [o, i] = [outer.split(sep).filter(Boolean), inner.split(sep).filter(Boolean)]
  return o.length <= i.length && o.every((s, n) => s === i[n])
}

/** The site's source folders; an outDir in one of them is refused. */
const SOURCE_DIRS = ['pages', 'public', 'components', 'src', 'node_modules']

/**
 * Refuse an outDir that is the site root or contains it, is inside one of its
 * source folders (a symlinked one by where it leads), holds a `.vue` file the
 * build used, or overlaps the app build. A folder of
 * its own inside the root, such as `dist`, is fine.
 */
function checkOutDir(outDir: string, root: string, appDir: string | null, vueFiles: string[] = []) {
  const out = canonical(outDir)
  const site = canonical(root)
  const sources = SOURCE_DIRS.map((d) => canonical(join(root, d)))
  const vue = vueFiles.map(canonical)
  if (
    contains(out, site) ||
    sources.some((d) => contains(d, out)) ||
    vue.some((f) => contains(out, f))
  ) {
    throw new Error(
      `buildSite: outDir ${outDir} is the site root, contains it, or is one of its source folders; build into a folder of its own, like dist.`,
    )
  }
  if (appDir && (contains(out, canonical(appDir)) || contains(canonical(appDir), out))) {
    throw new Error(`buildSite: outDir ${outDir} and the app build ${appDir} overlap; keep the app build in a folder of its own.`)
  }
}

/** First path segments the app may not have, and why. */
const RESERVED_PREFIXES = new Map([
  ['pages', 'holds the built pages'],
  ['api', 'the Go server sends to the API'],
])

/** The app's route; its build goes into the folder its prefix names. */
function appRoute(prefix: string): NonNullable<RoutesJson['app']> {
  if (!/^\/[^/]/.test(prefix) || prefix.endsWith('/')) {
    throw new Error(`buildSite: appPrefix "${prefix}" must start with "/" and not end with one, like "/app".`)
  }
  const segments = prefix.slice(1).split('/')
  if (segments.some((s) => s === '' || s === '.' || s === '..' || /[\\\0]/.test(s))) {
    throw new Error(`buildSite: appPrefix "${prefix}" has an empty, ".", ".." or backslashed segment.`)
  }
  // Plain URL characters only: the prefix is a folder name, a URL path and a speculation rules pattern.
  if (segments.some((s) => !/^[A-Za-z0-9._~-]+$/.test(s))) {
    throw new Error(
      `buildSite: appPrefix "${prefix}" has a character other than A-Z, a-z, 0-9, ".", "_", "~" and "-"; use a plain path, like "/app".`,
    )
  }
  // The Go server sends /api/ to the API and never serves the page files under /pages/ as they are.
  const first = segments[0]!
  const taken = RESERVED_PREFIXES.get(first.toLowerCase())
  if (taken) throw new Error(`buildSite: appPrefix "${prefix}" is under /${first}, which ${taken}; use another prefix, like "/app".`)
  return { prefix, index: `${segments.join('/')}/index.html` }
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

/** A CSS module request: `*.module.css` and the like, or a `<style module>` block. */
const CSS_MODULE = /\.module\.(css|less|sass|scss|styl|stylus|pcss|postcss|sss)(?:$|\?)/

/**
 * Island CSS is inlined into the pages that use the island, so the client build
 * ships none: plain stylesheets load empty, and the stylesheet the CSS modules
 * leave is dropped from the output. A CSS module keeps its class mapping, the
 * same one the server rendered with.
 */
const noCSS: Plugin = {
  name: 'weft:no-client-css',
  enforce: 'pre',
  load(id) {
    if (isCSSRequest(id) && !CSS_MODULE.test(id)) return { code: '' }
  },
  generateBundle: {
    order: 'post',
    handler(_, bundle) {
      for (const [name, file] of Object.entries(bundle)) if (file.type === 'asset' && name.endsWith('.css')) delete bundle[name]
    },
  },
}

type OutputOptions = Rolldown.OutputOptions

/** The client build's lock: its plugin, and the refusal it made, if any. */
interface OutputLock {
  plugin: Plugin
  /** The sentence the lock refused the build with, when it did. */
  refused: () => string | null
}

/**
 * Hold the client build to `dir`: after every other plugin has had its say,
 * the client environment, the one `build()` builds, must still build into
 * `dir` once, without emptying it and with no output folder or file of its
 * own; otherwise the build stops, before anything is written, with one
 * sentence naming the setting. Then the settings that say where files go are
 * frozen on every environment, so a plugin that changes one later, in
 * `buildStart` say, is refused the same way.
 */
function lockOutput(dir: string): OutputLock {
  let refused: string | null = null
  const refuse = (setting: string, then = 'the page build alone decides where that goes'): never => {
    refused = `buildSite: the site's Vite config or a plugin sets ${setting} for the client build; ${then}, so remove it.`
    throw new Error(refused)
  }
  const outputs = (o: OutputOptions | OutputOptions[] | undefined) => (Array.isArray(o) ? o : o ? [o] : [])
  const check = (root: string, b: ResolvedConfig['build']) => {
    for (const name of ['rollupOptions', 'rolldownOptions'] as const) {
      for (const o of outputs((b[name] as { output?: OutputOptions | OutputOptions[] } | undefined)?.output)) {
        if (o.dir !== undefined) refuse(`build.${name}.output.dir`)
        if (o.file !== undefined) refuse(`build.${name}.output.file`)
      }
    }
    if (resolve(root, b.outDir) !== dir) refuse('build.outDir')
    if (b.emptyOutDir !== false) refuse('build.emptyOutDir')
    if (b.watch != null) refuse('build.watch', 'the page build builds once')
  }
  /** Make `obj[key]` keep the value it has: reading it gives that, setting it is refused, redefining it fails. */
  const freeze = (obj: object, key: string, setting: string) => {
    if (Object.getOwnPropertyDescriptor(obj, key)?.configurable === false) return
    const value = (obj as Record<string, unknown>)[key]
    Object.defineProperty(obj, key, {
      get: () => value,
      set: () => refuse(setting),
      enumerable: true,
      configurable: false,
    })
  }
  const plugin: Plugin = {
    name: 'weft:lock-output',
    enforce: 'post',
    configResolved(config) {
      check(config.root, config.environments.client!.build)
      // The top-level options and every environment's. An environment reads
      // publicDir from the top level unless it has its own, and setting it on
      // the environment's config gives it one: each gets its own, frozen.
      for (const target of [config, ...Object.values(config.environments)]) {
        for (const key of ['outDir', 'emptyOutDir', 'copyPublicDir']) freeze(target.build, key, `build.${key}`)
        if (!('publicDir' in target)) Object.assign(target, { publicDir: config.publicDir })
        freeze(target, 'publicDir', 'publicDir')
      }
    },
    outputOptions: {
      order: 'post',
      handler(o) {
        if (o.file !== undefined || (o.dir !== undefined && resolve(o.dir) !== dir)) refuse(o.file !== undefined ? 'output.file' : 'output.dir')
      },
    },
  }
  return { plugin, refused: () => refused }
}

/** The site's output options without those that say where files go or what they are called. */
const placeless = ({ dir, file, entryFileNames, chunkFileNames, assetFileNames, ...rest }: OutputOptions): OutputOptions => rest

/** Build the island runtime and one chunk per island into `<dir>/assets`. */
async function buildClient(root: string, dir: string, islands: string[]): Promise<ClientBuild> {
  const lock = lockOutput(dir)
  const config = await siteConfig(root, { command: 'build', mode: 'production' }, {
    base: '/',
    logLevel: 'warn',
    publicDir: false,
    plugins: [noCSS, weft({ islands }), lock.plugin],
    // The runtime lives in this package; its `vue` must be the site's.
    resolve: { dedupe: ['vue'] },
    build: {
      outDir: dir,
      assetsDir: 'assets',
      emptyOutDir: false,
      copyPublicDir: false,
      write: true,
      manifest: false,
      ssrManifest: false,
      sourcemap: false,
      modulePreload: { polyfill: false },
      // Unsplit, no chunk imports or preloads a stylesheet.
      cssCodeSplit: false,
    },
  })
  // Only the runtime entry: an input of the site's config (an SPA's index.html)
  // would merge with it. The site's other options (external, output format)
  // stay, but not where it puts files: they go into `dir` and nowhere else.
  // Vite takes either name and prefers rolldownOptions; merge both into one.
  const b = (config.build ??= {})
  const options = mergeConfig(b.rollupOptions ?? {}, b.rolldownOptions ?? {}, false) as Rolldown.InputOptions & {
    output?: OutputOptions | OutputOptions[]
  }
  const output = Array.isArray(options.output) ? options.output.map(placeless) : placeless(options.output ?? {})
  b.rolldownOptions = b.rollupOptions = { ...options, input: { client: CLIENT_ENTRY }, output }
  let result: Rolldown.RolldownOutput | Rolldown.RolldownOutput[]
  try {
    result = (await build(config)) as typeof result
  } catch (e) {
    // Vite and Rolldown wrap a hook's error in their own words; the lock's sentence is the one to give.
    const refused = lock.refused()
    throw refused ? new Error(refused) : e
  }
  const chunks = (Array.isArray(result) ? result : [result]).flatMap((r) => r.output)
  const entry = chunks.find((o): o is Rolldown.OutputChunk => o.type === 'chunk' && o.isEntry)
  if (!entry) throw new Error('buildSite: the client build produced no entry chunk.')
  return {
    entry: '/' + entry.fileName,
    imports: entry.imports.map((f) => '/' + f),
    files: chunks.map((o) => o.fileName),
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
  /** The speculation rules script. */
  rules: string
}): string {
  return [
    '<!doctype html>',
    '<html lang="en">',
    '<head>',
    '<meta charset="utf-8">',
    '<meta name="viewport" content="width=device-width, initial-scale=1">',
    ...headTags(p.head),
    p.stylesheet === null ? `<style>${p.css}</style>` : `<link rel="stylesheet" href="${p.stylesheet}">`,
    p.rules,
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
