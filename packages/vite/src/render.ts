import { createRequire } from 'node:module'
import { join, relative, resolve } from 'node:path'
import { pathToFileURL } from 'node:url'
import { createServer, isCSSRequest, normalizePath, type EnvironmentModuleNode, type ViteDevServer } from 'vite'
import { siteConfig } from './config.ts'
import { sfcName } from './plugin.ts'

export interface RenderOptions {
  /** The site's root directory; `vue` is resolved from here. */
  root: string
  /** The page's `.vue` file, relative to `root` or absolute. */
  page: string
  /** Props for the page component. */
  props?: Record<string, unknown>
  /** A server from `createRenderServer(root)` to render through; without one, renderPage starts and closes its own. */
  server?: ViteDevServer
  /** Absolute paths of stylesheets the caller already includes; left out of `css`. */
  providedCSS?: string[]
  /**
   * Absolute paths of stylesheets that bundle every Weft component's CSS (weft.css and
   * its group sheets). A page that imports one is refused in a sentence: its CSS would
   * repeat the rules each component brings.
   */
  bundleCSS?: string[]
}

/** What the page set with `useHead` from @weft/ui. */
export interface PageHead {
  title?: string
  description?: string
  image?: string
}

export interface RenderResult {
  html: string
  /** Ids of the islands on the page, each once, in the order they appear. */
  islands: string[]
  /** Each island's id → the absolute path of its `.vue` file. */
  islandFiles: Record<string, string>
  /**
   * The CSS of what the page renders, in load order: the stylesheets and
   * `<style>` blocks of the page, of each component the server render used, of
   * the components those import, and of the modules (not components) they
   * import. A component the page only imports, through an index such as
   * @weft/ui's, and never renders adds nothing. An island brings the CSS of
   * everything its file imports, rendered or not, since the browser may draw it.
   */
  css: string
  head: PageHead
}

/**
 * A Vite dev server in middleware mode for rendering the site's pages: the
 * site's own `vite.config` if it has one, plus what server rendering needs.
 * `mode` is what the pages see as `import.meta.env.MODE`. Close it when done.
 */
export async function createRenderServer(root: string, mode = 'development'): Promise<ViteDevServer> {
  return createServer(
    await siteConfig(resolve(root), { command: 'serve', mode }, {
      appType: 'custom',
      logLevel: 'error',
      server: { middlewareMode: true, ws: false },
      optimizeDeps: { noDiscovery: true, include: [] },
      ssr: { noExternal: ['@weft/ui'] },
    }),
  )
}

/**
 * Render one page to HTML. App and Weft source load through a Vite dev server in
 * middleware mode; `vue` and `vue/server-renderer` load with Node's own `import`,
 * resolved from the site, because Vite's SSR loader cannot evaluate Vue's
 * CommonJS entry ("module is not defined").
 */
export async function renderPage(opts: RenderOptions): Promise<RenderResult> {
  const root = resolve(opts.root)
  const require = createRequire(join(root, 'package.json'))
  const { createSSRApp } = (await import(pathToFileURL(require.resolve('vue')).href)) as typeof import('vue')
  const { renderToString } = (await import(
    pathToFileURL(require.resolve('vue/server-renderer')).href
  )) as typeof import('vue/server-renderer')

  const vite = opts.server ?? (await createRenderServer(root))
  try {
    const file = '/' + relative(root, resolve(root, opts.page)).split('\\').join('/')
    const mod = (await vite.ssrLoadModule(file)) as { default?: unknown }
    if (!mod.default) throw new Error(`renderPage: ${opts.page} has no default export.`)
    // `modules` is filled by @vitejs/plugin-vue: every SFC whose setup ran in this render, relative to root.
    const ctx: { islands?: string[]; head?: PageHead; modules?: Set<string> } = {}
    const html = await renderToString(createSSRApp(mod.default as never, opts.props), ctx)
    const islands = [...new Set(ctx.islands ?? [])]
    const modules = await pageModules(vite, file)
    const islandFiles = findIslandFiles(modules, root, file, islands)
    // Vite's module files are posix paths, on Windows too.
    const rendered = new Set([...(ctx.modules ?? [])].map((m) => normalizePath(resolve(root, m))))
    const used = await pageModules(vite, file, { rendered, islands: new Set(Object.values(islandFiles).map(normalizePath)) })
    refuseBundles(used, opts.bundleCSS ?? [], root, opts.page)
    return {
      html,
      islands,
      islandFiles,
      css: await collectCSS(vite, used, new Set(opts.providedCSS)),
      head: ctx.head ?? {},
    }
  } finally {
    if (!opts.server) await vite.close()
  }
}

/** A component's own module: a `.vue` file, not one of its blocks (`?vue&type=style…`). */
const isSFC = (mod: EnvironmentModuleNode) => !!mod.id && mod.id.endsWith('.vue')

/**
 * Every module the page imports, directly or not, in load order (a module
 * after its imports). With `only`, just what the page's CSS needs:
 * - a component (a `.vue` file) is followed when the server render used it
 *   (`rendered`, absolute posix paths), or when a followed component imports it
 *   directly (it may draw it later, as a Button draws its Spinner once loading);
 * - one reached only through other modules, such as an index re-exporting
 *   every component, is left out with what it imports;
 * - under an island's file (`islands`), everything is followed, since the
 *   browser may draw any of it.
 */
async function pageModules(
  vite: ViteDevServer,
  page: string,
  only?: { rendered: Set<string>; islands: Set<string> },
): Promise<EnvironmentModuleNode[]> {
  const order: EnvironmentModuleNode[] = []
  const placed = new Set<EnvironmentModuleNode>()
  // Visited at all, and visited with everything under it followed (a module
  // first reached outside an island is visited again from inside one).
  const seen = new Set<EnvironmentModuleNode>()
  const seenWhole = new Set<EnvironmentModuleNode>()
  const visit = (mod: EnvironmentModuleNode, fromSFC: boolean, whole: boolean) => {
    const file = mod.file ? normalizePath(mod.file) : ''
    whole ||= !only || (isSFC(mod) && only.islands.has(file))
    if (whole) {
      if (seenWhole.has(mod)) return
      seenWhole.add(mod)
    } else {
      if (seen.has(mod)) return
      if (isSFC(mod) && !fromSFC && !only!.rendered.has(file)) return
    }
    seen.add(mod)
    for (const dep of mod.importedModules) visit(dep, isSFC(mod), whole)
    if (!placed.has(mod)) {
      placed.add(mod)
      order.push(mod)
    }
  }
  const start = await vite.environments.ssr.moduleGraph.getModuleByUrl(page)
  if (start) visit(start, true, false)
  return order
}

/** Refuse a page whose CSS would take in one of the stylesheets that bundle every component's CSS. */
function refuseBundles(modules: EnvironmentModuleNode[], bundles: string[], root: string, page: string) {
  const refused = new Set(bundles.map(normalizePath))
  for (const mod of modules) {
    if (!mod.file || !refused.has(normalizePath(mod.file)) || !isCSSRequest(mod.url)) continue
    const name = relative(root, resolve(root, page)).split('\\').join('/')
    throw new Error(
      `renderPage: ${name} imports ${relative(root, mod.file).split('\\').join('/')}, which holds every Weft component's CSS; a page gets the CSS of the components it renders, so remove the import.`,
    )
  }
}

/**
 * The CSS of the page's style modules, each loaded with `?inline` so Vite
 * returns it processed (scoped, PostCSS) as a string.
 */
async function collectCSS(vite: ViteDevServer, modules: EnvironmentModuleNode[], provided: Set<string>): Promise<string> {
  const parts: string[] = []
  for (const mod of modules) {
    if (!isCSSRequest(mod.url) || /[?&](inline|raw|url)\b/.test(mod.url)) continue
    if (mod.file && provided.has(mod.file)) continue
    const url = mod.url.includes('?') ? mod.url.replace('?', '?inline&') : mod.url + '?inline'
    const css = ((await vite.ssrLoadModule(url)) as { default?: string }).default
    if (css?.trim()) parts.push(css.trim())
  }
  return parts.join('\n')
}

/**
 * Each island's `.vue` file, found among the modules the page imports by its
 * SFC name, the island's id.
 */
function findIslandFiles(modules: EnvironmentModuleNode[], root: string, page: string, islands: string[]) {
  const out: Record<string, string> = {}
  const wanted = new Set(islands)
  const found = new Map<string, Set<string>>()
  for (const mod of modules) {
    if (!mod.file?.endsWith('.vue') || !wanted.has(sfcName(mod.file))) continue
    const name = sfcName(mod.file)
    if (!found.has(name)) found.set(name, new Set())
    found.get(name)!.add(mod.file)
  }
  for (const name of islands) {
    const [file, other] = [...(found.get(name) ?? [])].sort()
    if (!file) throw new Error(`renderPage: island ${name} on ${page.slice(1)} is not a .vue file the page imports.`)
    if (other) {
      throw new Error(
        `Islands ${relative(root, file)} and ${relative(root, other)} are both named ${name}; rename one of them.`,
      )
    }
    out[name] = file
  }
  return out
}
