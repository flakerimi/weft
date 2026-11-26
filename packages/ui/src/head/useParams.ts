/** A path segment decoded, or as it is when it does not decode. */
function decode(segment: string): string {
  try {
    return decodeURIComponent(segment)
  } catch {
    return segment
  }
}

/**
 * The params of the page's address, on a page whose paths are known only at
 * run time (one that exports `runtime = true`, such as `pages/products/[id].vue`).
 * The build writes the page's pattern on `<html>` as `data-weft-route`
 * (`/products/:id`); this reads `location.pathname` against it and gives each
 * param decoded (`{ id: '42' }`), or as it is when it does not decode. A
 * catch-all (`[...rest]`) gets the rest of the path without its leading
 * slash, or `''` when there is none. It is `{}` on any other page, on an
 * address the pattern does not match, and on the server, which renders the
 * page once with every param empty. It is not reactive: the page loads once
 * per address.
 */
export function useParams(): Record<string, string> {
  if (typeof document === 'undefined') return {}
  const pattern = document.documentElement.dataset.weftRoute
  if (!pattern) return {}
  // As the server cleans the path: no empty segment, no trailing slash.
  const want = pattern.split('/').filter(Boolean)
  const got = location.pathname.split('/').filter(Boolean).map(decode)
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
