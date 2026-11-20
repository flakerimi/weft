import { createRequire } from 'node:module'
import { join, relative, resolve } from 'node:path'
import { pathToFileURL } from 'node:url'
import { createServer, isCSSRequest, type EnvironmentModuleNode, type ViteDevServer } from 'vite'
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
  /** The CSS of every module the page imports (component `<style>` blocks, CSS imports), in load order. */
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
    const ctx: { islands?: string[]; head?: PageHead } = {}
    const html = await renderToString(createSSRApp(mod.default as never, opts.props), ctx)
    const islands = [...new Set(ctx.islands ?? [])]
    const modules = await pageModules(vite, file)
    return {
      html,
      islands,
      islandFiles: findIslandFiles(modules, root, file, islands),
      css: await collectCSS(vite, modules),
      head: ctx.head ?? {},
    }
  } finally {
    if (!opts.server) await vite.close()
  }
}

/** Every module the page imports, directly or not, in load order (a module after its imports). */
async function pageModules(vite: ViteDevServer, page: string): Promise<EnvironmentModuleNode[]> {
  const order: EnvironmentModuleNode[] = []
  const seen = new Set<EnvironmentModuleNode>()
  const visit = (mod: EnvironmentModuleNode) => {
    if (seen.has(mod)) return
    seen.add(mod)
    for (const dep of mod.importedModules) visit(dep)
    order.push(mod)
  }
  const start = await vite.environments.ssr.moduleGraph.getModuleByUrl(page)
  if (start) visit(start)
  return order
}

/**
 * The CSS of the page's style modules, each loaded with `?inline` so Vite
 * returns it processed (scoped, PostCSS) as a string.
 */
async function collectCSS(vite: ViteDevServer, modules: EnvironmentModuleNode[]): Promise<string> {
  const parts: string[] = []
  for (const mod of modules) {
    if (!isCSSRequest(mod.url) || /[?&](inline|raw|url)\b/.test(mod.url)) continue
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
