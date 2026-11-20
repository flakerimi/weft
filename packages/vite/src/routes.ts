import { readdir } from 'node:fs/promises'
import { join } from 'node:path'

export type Route = { path: string; file: string; dynamic: boolean }

/** The shape of `routes.json`, written by the build. */
export type RoutesJson = {
  pages: { path: string; html: string; islands: string[] }[]
  notFound: string | null
  app: { prefix: string; index: string } | null
}

export type PageModule = { getPaths?: () => Promise<Record<string, string>[]> }

const collision = (a: string, b: string, path: string) =>
  new Error(`pages "${a}" and "${b}" both make ${path}`)

async function walk(dir: string, prefix: string, out: string[]): Promise<void> {
  const entries = await readdir(dir, { withFileTypes: true })
  for (const e of entries) {
    if (e.name.startsWith('_') || e.name.startsWith('.')) continue
    if (e.isDirectory()) await walk(join(dir, e.name), `${prefix}${e.name}/`, out)
    else if (e.isFile() && e.name.endsWith('.vue')) out.push(`${prefix}${e.name}`)
  }
}

function toPath(file: string): string {
  const parts = file.slice(0, -'.vue'.length).split('/')
  if (parts[parts.length - 1] === 'index') parts.pop()
  const path = '/' + parts.map((p) => p.replace(/^\[(.+)\]$/, ':$1')).join('/')
  return path
}

const byOrder = (a: Route, b: Route) =>
  Number(a.dynamic) - Number(b.dynamic) || (a.path < b.path ? -1 : a.path > b.path ? 1 : 0)

/** Finds the pages under `pagesDir`; dynamic routes are marked, not expanded. */
export async function discoverRoutes(pagesDir: string): Promise<Route[]> {
  const files: string[] = []
  await walk(pagesDir, '', files)
  files.sort()
  const routes: Route[] = files.map((file) => {
    const path = toPath(file)
    return { path, file, dynamic: path.includes(':') }
  })
  const seen = new Map<string, string>()
  for (const r of routes) {
    const other = seen.get(r.path)
    if (other !== undefined) throw collision(other, r.file, r.path)
    seen.set(r.path, r.file)
  }
  return routes.sort(byOrder)
}

/** A concrete page: its path, its file, and the params that made the path (the page's props). */
export type ExpandedRoute = { path: string; file: string; params: Record<string, string> }

/** Turns routes into concrete paths, calling each dynamic page's `getPaths()`. */
export async function expandRoutes(
  routes: Route[],
  loadPage: (file: string) => Promise<PageModule>,
): Promise<ExpandedRoute[]> {
  const out: ExpandedRoute[] = []
  const owner = new Map<string, string>()
  const add = (path: string, file: string, params: Record<string, string>) => {
    const other = owner.get(path)
    if (other !== undefined) throw collision(other, file, path)
    owner.set(path, file)
    out.push({ path, file, params })
  }
  for (const r of routes) if (!r.dynamic) add(r.path, r.file, {})
  for (const r of routes) {
    if (!r.dynamic) continue
    const { getPaths } = await loadPage(r.file)
    if (typeof getPaths !== 'function') {
      throw new Error(`page "${r.file}" is dynamic but does not export getPaths()`)
    }
    for (const params of await getPaths()) {
      const path = r.path.replace(/:([^/]+)/g, (_, name: string) => {
        const v = params[name]
        if (v === undefined) throw new Error(`getPaths() of "${r.file}" returned no "${name}"`)
        return v
      })
      add(path, r.file, params)
    }
  }
  return out
}
