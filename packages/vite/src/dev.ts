// The page dev server: each page is rendered when it is asked for, through one
// Vite server that listens, so an edit shows on the next request; a change
// to a file under the site reloads the browser. It writes the page as the
// build does (the theme, base and page CSS inline, the head tags, the icon
// tags and the pre-paint script), with Vite's client, and a module script
// that hydrates the page's islands from Vite's own modules, not a client
// build.
import { realpathSync, statSync } from 'node:fs'
import { stat } from 'node:fs/promises'
import { request as httpRequest, type IncomingMessage, type ServerResponse } from 'node:http'
import { request as httpsRequest } from 'node:https'
import { isAbsolute, join, relative, resolve, sep } from 'node:path'
import { fileURLToPath } from 'node:url'
import { createServer, isCSSRequest, normalizePath, type Plugin, type ViteDevServer } from 'vite'
import { contentType, CSS_MODULE, dropStaticComments } from './build.ts'
import { siteConfig } from './config.ts'
import { escapeHTML, iconTags, pageHTML, readStyles, THEMES, VIEW_TRANSITION, type Theme } from './document.ts'
import { renderPage } from './render.ts'
import { byPattern, cleanPath, decodePath, discoverRoutes, expandRoutes, matchPattern, refuseSameShape, type ExpandedRoute, type PageModule, type Route } from './routes.ts'

export interface DevOptions {
  /** The site's folder, which holds `pages/`. Its `package.json` may be in a folder above. */
  root: string
  /** The port to listen on; 0 takes a free one. Default 5173. */
  port?: number
  /** The host to listen on. Default `localhost`. */
  host?: string
  /** Where `/api` and what is under it is proxied, such as `http://127.0.0.1:8081`. */
  api?: string
  /** `neutral` (the default), `lisa` or `weft`. */
  theme?: Theme
}

export interface DevServer {
  /** Where it listens, such as `http://localhost:5173/`. */
  url: string
  /** Stop listening and close the Vite server. */
  close(): Promise<void>
}

const WHO = 'weft dev'
/** The module that hydrates islands, run in the browser from source. */
const RUNTIME = normalizePath(fileURLToPath(new URL('./island-runtime.ts', import.meta.url)))
/** The module script of a page with islands: `?i=` names them, as base64url JSON of name → file. */
const ISLANDS = 'virtual:weft/dev-islands'
const RESOLVED_ISLANDS = '\0' + ISLANDS

/**
 * Start the dev server for the site at `root`. It renders a page whose path
 * is requested then (a dynamic page's `getPaths()` is called each time), a
 * path a runtime page's pattern matches as that page's shell, as the build
 * writes it, an unknown path as `pages/404.vue` with status 404 or a plain 404, and a page
 * that fails as a 500 that names its file; serves the files the pages it has
 * rendered wrote with `useFile`; proxies `/api` to `api`; and
 * sends a full reload to the browser when a file under `root` changes.
 * Refuses, in one sentence, a port in use.
 */
export async function devSite(opts: DevOptions): Promise<DevServer> {
  const root = resolve(opts.root)
  const theme = opts.theme ?? 'neutral'
  if (!THEMES.includes(theme)) throw new Error(`${WHO}: unknown theme "${theme}"; use ${THEMES.slice(0, -1).join(', ')} or ${THEMES.at(-1)}.`)
  const port = opts.port ?? 5173
  if (!Number.isInteger(port) || port < 0 || port > 65535) throw new Error(`${WHO}: port ${port} is not a port number; give one from 0 to 65535.`)
  if (opts.api !== undefined && !/^https?:\/\/[^/]/.test(opts.api)) {
    throw new Error(`${WHO}: --api "${opts.api}" is not an http:// or https:// address; give one like http://127.0.0.1:8081.`)
  }
  const pagesDir = join(root, 'pages')
  const pagesStat = await stat(pagesDir).catch(() => null)
  if (!pagesStat?.isDirectory()) {
    throw new Error(`${WHO}: ${pagesDir} ${pagesStat ? 'is not a folder' : 'does not exist'}; a site's pages are .vue files in it, like pages/index.vue.`)
  }
  // Read now, so a site without @weft/ui is refused before anything listens; again for each page.
  await readStyles(root, theme, WHO)

  // The watcher may name a file by its real path, under a root reached through a symlink.
  const roots = [...new Set([root, realpathSync(root)])]
  const known = new Set<string>()
  let server: ViteDevServer
  const plugin: Plugin = {
    name: 'weft:dev',
    enforce: 'pre',
    resolveId(id) {
      if (id.startsWith(ISLANDS + '?')) return '\0' + id
      if (id.startsWith(RESOLVED_ISLANDS + '?')) return id
    },
    load(id) {
      if (id.startsWith(RESOLVED_ISLANDS + '?')) return islandsModule(id, known)
      // As in the build, the browser gets no stylesheet: the page's <style>
      // holds the CSS of all it renders, its islands' included. A CSS module
      // keeps its class names, the server's, and adds rules the page has.
      if (this.environment.name === 'client' && isCSSRequest(id) && !CSS_MODULE.test(id)) return { code: '' }
    },
    // Last, after @vitejs/plugin-vue's: every page is rendered on request, so
    // a reload is always right, and no module is replaced in place.
    hotUpdate: {
      order: 'post',
      handler({ file }) {
        // A declaration file, such as the one weftAutoImport() writes when it starts, changes no page.
        if (this.environment.name !== 'client' || file.endsWith('.d.ts')) return []
        const used = (name: 'client' | 'ssr') => (server.environments[name].moduleGraph.getModulesByFile(file)?.size ?? 0) > 0
        if (roots.some((r) => inside(r, file)) || used('client') || used('ssr')) this.environment.hot.send({ type: 'full-reload', path: '*' })
        return []
      },
    },
    configureServer(vite) {
      server = vite
      // Before Vite's own middleware, its proxy included: what warp.Serve
      // would send to the API, as sent, and the escape net/http refuses.
      vite.middlewares.use((req, res, next) => (asServed(req, res, opts.api) ? undefined : next()))
      // After Vite's own middleware (its modules, public/ and the site's files): what is left is a page.
      return () => vite.middlewares.use((req, res, next) => void pages(req, res, next))
    },
  }

  const config = await siteConfig(root, { command: 'serve', mode: 'development' }, {
    appType: 'custom',
    logLevel: 'warn',
    clearScreen: false,
    plugins: [plugin],
    // The runtime lives in this package; its `vue` must be the site's, one copy for every island.
    resolve: { dedupe: ['vue'] },
    server: {
      port,
      host: opts.host ?? 'localhost',
      strictPort: true,
      ...(opts.api ? { proxy: { '^/api(?:/|\\?|$)': { target: opts.api, ws: true } } } : {}),
    },
    ssr: { noExternal: ['@weft/ui'] },
  })
  server = await createServer(config)

  // One page at a time, as the build renders them.
  let queue: Promise<unknown> = Promise.resolve()
  const mtimes = new Map<string, number>()
  /** The files the pages rendered so far wrote with useFile, by path; a page's latest render wins. */
  const written = new Map<string, string>()

  /**
   * Drop what changed on disk since it was loaded, before the watcher may have
   * said so: the next request after a save renders the saved file.
   */
  async function freshen() {
    const changed: string[] = []
    await Promise.all(
      [...mtimes].map(async ([file, was]) => {
        const now = await stat(file).then((s) => s.mtimeMs, () => -1)
        if (now !== was) {
          changed.push(file)
          mtimes.delete(file)
        }
      }),
    )
    for (const file of changed) for (const env of Object.values(server.environments)) env.moduleGraph.onFileChange(file)
  }
  function track() {
    for (const mod of server.environments.ssr.moduleGraph.idToModuleMap.values()) {
      const file = mod.file
      // Files only: not a virtual module, nor a package, which is not edited.
      if (!file || !isAbsolute(file) || file.includes('\0') || mtimes.has(file) || file.includes('/node_modules/')) continue
      mtimes.set(file, statSync(file, { throwIfNoEntry: false })?.mtimeMs ?? -1)
    }
  }

  const load = (file: string) => server.ssrLoadModule(`/pages/${file}`) as Promise<PageModule>

  /**
   * The page at `path`: a static one, else the first dynamic page whose
   * getPaths() gives it. Without one, the runtime pages too, in the order
   * warp.Serve tries them, for the caller to try after the files pages wrote.
   */
  async function find(routes: Route[], path: string, failing: { file: string }): Promise<{ route: ExpandedRoute | null; runtime: ExpandedRoute[] }> {
    const pages = routes.filter((r) => r.file !== '404.vue')
    const page = pages.find((r) => !r.dynamic && r.path === path)
    if (page) return { route: { path, file: page.file, params: {} }, runtime: [] }
    const runtime: ExpandedRoute[] = []
    for (const r of pages.filter((r) => r.dynamic)) {
      failing.file = r.file
      const expanded = await expandRoutes([r], load)
      const found = expanded.find((e) => !e.runtime && e.path === path)
      if (found) return { route: found, runtime: [] }
      runtime.push(...expanded.filter((e) => e.runtime))
    }
    // Refused as the build refuses them: two runtime pages of one shape.
    failing.file = ''
    refuseSameShape(runtime)
    return { route: null, runtime: runtime.sort(byPattern) }
  }

  async function render(route: ExpandedRoute): Promise<string> {
    const { css: baseCSS, files: baseFiles, bundles } = await readStyles(root, theme, WHO)
    const page = await renderPage({ root, page: join('pages', route.file), props: route.params, server, providedCSS: baseFiles, bundleCSS: bundles })
    for (const file of Object.values(page.islandFiles)) known.add(normalizePath(file))
    for (const { path, body } of page.files) written.set(path, body)
    const islands = Buffer.from(JSON.stringify(page.islandFiles)).toString('base64url')
    return pageHTML({
      body: dropStaticComments(page.html),
      head: page.head,
      prePaint: page.prePaint,
      // As the build has them; Vite serves public/ itself.
      icons: iconTags(root),
      fonts: [],
      css: [baseCSS, page.css, VIEW_TRANSITION].filter(Boolean).join('\n') + '\n',
      stylesheet: null,
      route: route.runtime ? route.path : undefined,
      tags: [
        '<script type="module" src="/@vite/client"></script>',
        ...(page.islands.length ? [`<script type="module" src="/@id/__x00__${ISLANDS}?i=${islands}"></script>`] : []),
      ],
      who: WHO,
    })
  }

  /** The answer for the page at `path`, or the file a page wrote at `exact` (the path as asked for). */
  async function answer(path: string, exact: string): Promise<{ status: number; type: string; body: string }> {
    const failing = { file: '' }
    try {
      await freshen()
      const routes = await discoverRoutes(pagesDir)
      const found = await find(routes, path, failing)
      let route = found.route
      const file = written.get(exact)
      if (!route && file !== undefined) return { status: 200, type: contentType(exact), body: file }
      // A runtime page's shell, rendered as the build renders it: every param empty.
      route ??= found.runtime.find((r) => matchPattern(r.path, path) !== null) ?? null
      const status = route ? 200 : 404
      if (!route && routes.some((r) => r.file === '404.vue')) route = { path, file: '404.vue', params: {} }
      if (!route) return { status: 404, type: 'text/plain; charset=utf-8', body: '404 page not found\n' }
      failing.file = route.file
      return { status, type: 'text/html; charset=utf-8', body: await render(route) }
    } catch (e) {
      const error = e instanceof Error ? e : new Error(String(e))
      server.ssrFixStacktrace(error)
      const page = failing.file ? `pages/${failing.file}` : 'pages/'
      server.config.logger.error(`${WHO}: ${page} did not render: ${error.message.split('\n')[0]}`)
      return { status: 500, type: 'text/html; charset=utf-8', body: errorPage(error, page, roots) }
    } finally {
      track()
    }
  }

  async function pages(req: IncomingMessage, res: ServerResponse, next: (e?: unknown) => void) {
    if (req.method !== 'GET' && req.method !== 'HEAD') return next()
    // The path as asked for, decoded as net/http decodes it: not read as a
    // URL, where //api/x would be the host api, and with an escape that is
    // not UTF-8 (%FF) kept as it is, so it is a param like any other; then
    // cleaned of dot segments as warp.Serve cleans it (%2e%2e too).
    const exact = cleanPath(decodePath((req.url ?? '/').replace(/[?#][\s\S]*$/, '')))
    // A page at its path, with or without a trailing slash, as warp.Serve has it.
    const path = exact.length > 1 && exact.endsWith('/') ? exact.slice(0, -1) : exact
    const done = queue.then(() => answer(path, exact))
    queue = done.catch(() => {})
    let status, type, body
    try {
      ;({ status, type, body } = await done)
    } catch (e) {
      return next(e)
    }
    res.statusCode = status
    res.setHeader('Content-Type', type)
    res.setHeader('Cache-Control', 'no-cache')
    res.end(req.method === 'HEAD' ? undefined : body)
  }

  try {
    await server.listen()
  } catch (e) {
    await server.close()
    if (e instanceof Error && /is already in use/.test(e.message)) {
      throw new Error(`${WHO}: port ${port} is in use; stop what is using it or pass --port.`)
    }
    throw e
  }
  const url = server.resolvedUrls?.local[0] ?? `http://${opts.host ?? 'localhost'}:${server.config.server.port}/`
  return { url, close: () => server.close() }
}

/**
 * Answers a request as warp.Serve's side of net/http would before any page,
 * and says whether it did: a path with a % that starts no escape is 400, as
 * net/http refuses it; a path whose decoded form, before any cleaning, is
 * /api or under it goes to `api` as it was sent (`/api/../login`,
 * `/%61pi/x`), where Vite's proxy would clean it or miss it. A WebSocket's
 * upgrade still goes through Vite's proxy.
 */
function asServed(req: IncomingMessage, res: ServerResponse, api: string | undefined): boolean {
  const raw = (req.url ?? '/').replace(/[?#][\s\S]*$/, '')
  if (/%(?![0-9a-fA-F]{2})/.test(raw)) {
    res.statusCode = 400
    res.setHeader('Content-Type', 'text/plain; charset=utf-8')
    res.end('400 Bad Request')
    return true
  }
  const path = decodePath(raw)
  if (!api || (path !== '/api' && !path.startsWith('/api/'))) return false
  const target = new URL(api)
  const send = target.protocol === 'https:' ? httpsRequest : httpRequest
  const up = send(
    { protocol: target.protocol, hostname: target.hostname, port: target.port, method: req.method, path: target.pathname.replace(/\/$/, '') + (req.url ?? '/'), headers: req.headers },
    (answer) => {
      res.writeHead(answer.statusCode ?? 502, answer.headers)
      answer.pipe(res)
    },
  )
  up.on('error', () => {
    if (!res.headersSent) res.statusCode = 502
    res.end()
  })
  req.pipe(up)
  return true
}

const inside = (dir: string, file: string) => resolve(file).startsWith(dir.endsWith(sep) ? dir : dir + sep)

/** The module script of a page with islands: it starts them from their own `.vue` files. */
function islandsModule(id: string, known: Set<string>): string {
  const files = JSON.parse(Buffer.from(new URLSearchParams(id.slice(id.indexOf('?'))).get('i') ?? '', 'base64url').toString() || '{}') as Record<string, string>
  const entries = Object.entries(files).map(([name, file]) => {
    // Only what a render found: the URL cannot name any file to be served.
    if (typeof file !== 'string' || !known.has(normalizePath(file))) throw new Error(`${WHO}: ${name} is not an island a page rendered; reload the page.`)
    return `  ${JSON.stringify(name)}: () => import(${JSON.stringify(normalizePath(file))}).then((m) => m.default),`
  })
  return `import { startIslands } from ${JSON.stringify(RUNTIME)}\nstartIslands({\n${entries.join('\n')}\n})\n`
}

/** A page that failed: what went wrong, where, and Vite's client, so saving a fix reloads it. */
function errorPage(error: Error & { id?: string; loc?: { file?: string; line?: number; column?: number }; frame?: string }, page: string, roots: string[]): string {
  /** A file from the site's folder, as the page is named, or its whole path outside it. */
  const name = (file: string) => {
    const path = file.split('?')[0]!
    const root = roots.find((r) => inside(r, path))
    return root ? relative(root, path).split(sep).join('/') : path
  }
  const where = error.loc?.file ?? error.id
  const at = where && name(where) !== page ? ` in ${name(where)}${error.loc?.line ? `:${error.loc.line}:${error.loc.column ?? 0}` : ''}` : ''
  const details = [error.message, error.frame, error.stack?.split('\n').slice(1).join('\n')].filter(Boolean).join('\n\n')
  return [
    '<!doctype html>',
    '<html lang="en">',
    '<head>',
    '<meta charset="utf-8">',
    `<title>${escapeHTML(page)} did not render</title>`,
    '<script type="module" src="/@vite/client"></script>',
    '<style>body { font: 15px/1.5 system-ui, sans-serif; margin: 2rem; } pre { white-space: pre-wrap; background: #f4f4f5; padding: 1rem; border-radius: 6px; }</style>',
    '</head>',
    `<body><h1>${escapeHTML(page)} did not render${escapeHTML(at)}</h1><pre>${escapeHTML(details)}</pre></body>`,
    '</html>',
    '',
  ].join('\n')
}
