/** The path decoded, as net/http decodes it into r.URL.Path, or as it is when it does not decode. */
function decode(path: string): string {
  try {
    return decodeURIComponent(path)
  } catch {
    return path
  }
}

/** The path's segments as Go's path.Clean leaves them: no empty or `.` segment, and a `..` taking the one before it back. */
function clean(path: string): string[] {
  const out: string[] = []
  for (const s of path.split('/')) {
    if (s === '..') out.pop()
    else if (s !== '' && s !== '.') out.push(s)
  }
  return out
}

/**
 * The params of the page's address, on a page whose paths are known only at
 * run time (one that exports `runtime = true`, such as `pages/products/[id].vue`).
 * The build writes the page's pattern on `<html>` as `data-weft-route`
 * (`/products/:id`); this reads `location.pathname` against it and gives the
 * params (`{ id: '42' }`). The path is split as warp.Serve splits it: decoded
 * first (or taken as it is when it does not decode), then cleaned as
 * path.Clean cleans it, so an encoded slash (`%2F`) divides segments and is
 * never inside one param. A catch-all (`[...rest]`) gets the rest of the path
 * without its leading slash, or `''` when there is none. It is `{}` on any other page, on an
 * address the pattern does not match, and on the server, which renders the
 * page once with every param empty. It is not reactive: the page loads once
 * per address.
 */
export function useParams(): Record<string, string> {
  if (typeof document === 'undefined') return {}
  const pattern = document.documentElement.dataset.weftRoute
  if (!pattern) return {}
  const want = pattern.split('/').filter(Boolean)
  const got = clean(decode(location.pathname))
  const params: Record<string, string> = {}
  for (let i = 0; i < want.length; i++) {
    const w = want[i]!
    if (w.startsWith(':...')) {
      params[w.slice(4)] = got.slice(i).join('/')
      return params
    }
    const g = got[i]
    if (g === undefined) return {}
    if (w.startsWith(':')) params[w.slice(1)] = g
    else if (g !== w) return {}
  }
  return got.length === want.length ? params : {}
}
