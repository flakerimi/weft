import { readdir, stat } from 'node:fs/promises'
import { join } from 'node:path'

export type Route = { path: string; file: string; dynamic: boolean }

/**
 * The shape of `routes.json`, written by the build. A runtime page's entry has
 * its pattern as its path (`/products/:id`) and `runtime: true`; the others
 * have no `runtime`. Runtime pages come after every other page, in the order
 * `byPattern` gives, which is the order the server tries them in.
 */
export type RoutesJson = {
  pages: { path: string; html: string; islands: string[]; runtime?: true }[]
  notFound: string | null
  app: { prefix: string; index: string } | null
}

/**
 * What a page's plain `<script>` may export: `getPaths()`, or `runtime = true`
 * for a dynamic page built once as a shell whose paths are known only at run
 * time. Any other value of `runtime` is the same as none.
 */
export type PageModule = { getPaths?: () => Promise<Record<string, string>[]>; runtime?: unknown }

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

/**
 * A concrete page: its path, its file, and the params that made the path (the
 * page's props). A runtime page is one shell: its path is its pattern, and
 * every param is the empty string.
 */
export type ExpandedRoute = { path: string; file: string; params: Record<string, string>; runtime?: true }

/** The pattern's segments: `/products/:id` → ['products', ':id']. */
const segmentsOf = (pattern: string) => pattern.slice(1).split('/').filter(Boolean)
const isParam = (segment: string) => segment.startsWith(':')
const isRest = (segment: string) => segment.startsWith(':...')

/**
 * The order the server tries runtime pages in, so that a catch-all never
 * hides a more specific page: more static segments first; then a pattern
 * without a catch-all before one with (`/docs/:a/:b` before `/docs/:...rest`);
 * then fewer params, except that of two catch-alls the one with more comes
 * first (`/docs/:a/:...r` before `/docs/:...rest`, which matches everything
 * it does); then by path. Two patterns without a catch-all and with as many
 * static segments overlap only when their lengths are equal, so for them the
 * count orders nothing that matters.
 */
export function byPattern(a: { path: string }, b: { path: string }): number {
  const [x, y] = [segmentsOf(a.path), segmentsOf(b.path)]
  const statics = (s: string[]) => s.filter((p) => !isParam(p)).length
  const params = (s: string[]) => s.filter(isParam).length
  const rest = (s: string[]) => Number(s.some(isRest))
  return (
    statics(y) - statics(x) ||
    rest(x) - rest(y) ||
    (rest(x) ? params(y) - params(x) : params(x) - params(y)) ||
    (a.path < b.path ? -1 : a.path > b.path ? 1 : 0)
  )
}

/**
 * A path's segments as Go's path.Clean leaves them: no empty or `.` segment,
 * and a `..` taking the one before it back.
 */
function cleanSegments(path: string): string[] {
  const out: string[] = []
  for (const s of path.split('/')) {
    if (s === '..') out.pop()
    else if (s !== '' && s !== '.') out.push(s)
  }
  return out
}

/**
 * The params of `path` against a runtime page's `pattern`, or null when it
 * does not match, as warp.Serve matches: `:name` takes one segment, `:...name`
 * the rest of the path (joined by `/`) or none of it (`''`), and a static
 * segment is compared exactly. `path` is decoded already, and cleaned here as
 * the server cleans it (cleanSegments).
 */
export function matchPattern(pattern: string, path: string): Record<string, string> | null {
  const want = segmentsOf(pattern)
  const got = cleanSegments(path)
  const params: Record<string, string> = {}
  for (let i = 0; i < want.length; i++) {
    const w = want[i]!
    if (isRest(w)) {
      params[w.slice(4)] = got.slice(i).join('/')
      return params
    }
    const g = got[i]
    if (g === undefined) return null
    if (isParam(w)) params[w.slice(1)] = g
    else if (g !== w) return null
  }
  return got.length === want.length ? params : null
}

/** A runtime page's shape, the same for patterns that match the same paths: `/a/:x` and `/a/:y` are `/a/:`. */
const shapeOf = (pattern: string) => pattern.replace(/\/:(\.\.\.)?[^/]+/g, (_, rest: string | undefined) => (rest ? '/:...' : '/:'))

/** Refuses two runtime pages of one shape, which would match the same paths, in the order they are given. */
export function refuseSameShape(pages: { path: string; file: string }[]): void {
  const shapes = new Map<string, string>()
  for (const r of pages) {
    const shape = shapeOf(r.path)
    const other = shapes.get(shape)
    if (other !== undefined) {
      throw new Error(`pages "${other}" and "${r.file}" both match paths like ${shape.replace(/:(\.\.\.)?/g, '1')}; keep one.`)
    }
    shapes.set(shape, r.file)
  }
}

/**
 * Turns routes into concrete paths, calling each dynamic page's `getPaths()`;
 * a dynamic page that exports `runtime = true` is one shell instead. Two
 * runtime pages of the same shape are refused; an exact page a runtime page's
 * pattern also matches is allowed, and wins at run time.
 */
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
    const { getPaths, runtime } = await loadPage(r.file)
    if (runtime === true) {
      if (getPaths !== undefined) throw new Error(`page "${r.file}" exports both runtime and getPaths; keep one.`)
      const params = Object.fromEntries(segmentsOf(r.path).filter(isParam).map((p) => [p.replace(/^:(\.\.\.)?/, ''), '']))
      out.push({ path: r.path, file: r.file, params, runtime: true })
      continue
    }
    if (typeof getPaths !== 'function') {
      throw new Error(`page "${r.file}" is dynamic but does not export getPaths()`)
    }
    for (const params of await getPaths()) add(fill(r, params), r.file, params)
  }
  refuseSameShape(out.filter((r) => r.runtime))
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
