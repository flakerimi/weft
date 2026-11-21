import { readdir, stat } from 'node:fs/promises'
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
    if (e.isSymbolicLink()) {
      // A linked page or folder would be built from somewhere else, or twice; say so rather than skip it.
      const isDir = await stat(join(dir, e.name)).then((s) => s.isDirectory(), () => false)
      if (e.name.endsWith('.vue') || isDir) {
        throw new Error(
          `pages/${prefix}${e.name} is a symlink, which Weft does not follow; put the ${isDir ? 'folder' : 'page'} itself in pages/.`,
        )
      }
    } else if (e.isDirectory()) await walk(join(dir, e.name), `${prefix}${e.name}/`, out)
    else if (e.isFile() && e.name.endsWith('.vue')) out.push(`${prefix}${e.name}`)
  }
}

/**
 * A path as the server and the file systems compare it: the build writes each
 * page to a folder named after its path, and a file system that ignores case or
 * Unicode form would take two such folders for one, so paths that differ only
 * there clash on every OS.
 */
const pathKey = (path: string) => path.normalize('NFC').toLowerCase()

/**
 * A page file's path: `[name]` is a param (`:name`) that fills one segment,
 * `[...name]` a catch-all (`:...name`) that fills the rest of the path, or none of it.
 */
function toPath(file: string): string {
  const parts = file.slice(0, -'.vue'.length).split('/')
  if (parts[parts.length - 1] === 'index') parts.pop()
  const rest = parts.findIndex((p) => /^\[\.\.\..+\]$/.test(p))
  if (rest !== -1 && rest !== parts.length - 1) {
    throw new Error(
      `page "${file}" has the catch-all ${parts[rest]} before its last segment; a catch-all takes the rest of the path, so it must come last.`,
    )
  }
  return '/' + parts.map((p) => p.replace(/^\[(.+)\]$/, ':$1')).join('/')
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
    const other = seen.get(pathKey(r.path))
    if (other !== undefined) throw collision(other, r.file, r.path)
    seen.set(pathKey(r.path), r.file)
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
    const other = owner.get(pathKey(path))
    if (other !== undefined) throw collision(other, file, path)
    owner.set(pathKey(path), file)
    out.push({ path, file, params })
  }
  for (const r of routes) if (!r.dynamic) add(r.path, r.file, {})
  for (const r of routes) {
    if (!r.dynamic) continue
    const { getPaths } = await loadPage(r.file)
    if (typeof getPaths !== 'function') {
      throw new Error(`page "${r.file}" is dynamic but does not export getPaths()`)
    }
    for (const params of await getPaths()) add(fill(r, params), r.file, params)
  }
  return out
}

/** A dynamic route's path with `params` in its params' places. */
function fill(r: Route, params: Record<string, string>): string {
  const segments = r.path.slice(1).split('/').flatMap((segment) => {
    const param = /^:(\.\.\.)?(.+)$/.exec(segment)
    if (!param) return [segment]
    const rest = param[1] !== undefined
    const name = param[2]!
    const v = params[name]
    if (v === undefined) throw new Error(`getPaths() of "${r.file}" returned no "${name}"`)
    const bad = (s: string) => s === '' || s === '.' || s === '..'
    if (!rest) {
      if (bad(v) || v.includes('/')) {
        throw new Error(
          `getPaths() of "${r.file}" returned ${JSON.stringify(v)} for ${name}, which is not one path segment; a page named [...${name}].vue takes several.`,
        )
      }
      return [v]
    }
    // A catch-all of "" is the folder itself: /docs for docs/[...slug].vue.
    if (v === '') return []
    if (v.split('/').some(bad)) {
      throw new Error(
        `getPaths() of "${r.file}" returned ${JSON.stringify(v)} for ${name}, which has an empty, . or .. segment; give segments joined by /, or "" for the folder itself.`,
      )
    }
    return v.split('/')
  })
  return '/' + segments.join('/')
}
