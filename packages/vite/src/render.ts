import { createRequire } from 'node:module'
import { join, relative, resolve } from 'node:path'
import { pathToFileURL } from 'node:url'
import vue from '@vitejs/plugin-vue'
import { createServer } from 'vite'

export interface RenderOptions {
  /** The site's root directory; `vue` is resolved from here. */
  root: string
  /** The page's `.vue` file, relative to `root` or absolute. */
  page: string
  /** Props for the page component. */
  props?: Record<string, unknown>
}

export interface RenderResult {
  html: string
  /** Ids of the islands on the page, each once, in the order they appear. */
  islands: string[]
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

  const vite = await createServer({
    configFile: false,
    root,
    plugins: [vue()],
    appType: 'custom',
    logLevel: 'error',
    server: { middlewareMode: true, ws: false },
    optimizeDeps: { noDiscovery: true, include: [] },
    ssr: { noExternal: ['@weft/ui'] },
  })
  try {
    const file = '/' + relative(root, resolve(root, opts.page)).split('\\').join('/')
    const mod = (await vite.ssrLoadModule(file)) as { default?: unknown }
    if (!mod.default) throw new Error(`renderPage: ${opts.page} has no default export.`)
    const ctx: { islands?: string[] } = {}
    const html = await renderToString(createSSRApp(mod.default as never, opts.props), ctx)
    return { html, islands: [...new Set(ctx.islands ?? [])] }
  } finally {
    await vite.close()
  }
}
