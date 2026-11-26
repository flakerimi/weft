/**
 * The path decoded, as net/http decodes it into r.URL.Path. When it is not
 * valid UTF-8 as a whole, each escape that decodes (a character's escapes
 * together) still does, and one that does not (`%FF`, a cut-off `%C3`) stays
 * as it is, so an encoded slash splits the path where the server splits it.
 */
function decode(path: string): string {
  try {
    return decodeURIComponent(path)
  } catch {
    return path.replace(/(?:%[0-9a-fA-F]{2})+/g, decodeRun)
  }
}

/** A run of escapes, one character at a time: the escapes its first byte says it takes, or that byte alone, kept. */
function decodeRun(run: string): string {
  const bytes = run.match(/%../g)!
  let out = ''
  for (let i = 0; i < bytes.length; ) {
    const lead = parseInt(bytes[i]!.slice(1), 16)
    const n = lead >= 0xf0 ? 4 : lead >= 0xe0 ? 3 : lead >= 0xc0 ? 2 : 1
    try {
      out += decodeURIComponent(bytes.slice(i, i + n).join(''))
      i += n
    } catch {
      out += bytes[i]
      i++
    }
  }
  return out
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
 * params (`{ id: '42' }`). The path is split as warp.Serve splits it:
 * decoded first (an escape that is not UTF-8, such as `%FF`, kept as it
 * is), then cleaned as path.Clean cleans it, so an encoded slash (`%2F`)
 * divides segments and is never inside one param. A catch-all
 * (`[...rest]`) gets the rest of the path without its leading slash, or
 * `''` when there is none. It is `{}` on any other page, on an address the
 * pattern does not match, and on the server, which renders the page once
 * with every param empty. It is not reactive: the page loads once
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
