// What Link and Button (with `to`) share about an address.

/** An address with a scheme and a host (`https://…`, `//…`): another site, as far as the markup can tell. */
export function isExternal(address: string | undefined): boolean {
  return /^([a-z][a-z\d+.-]*:)?\/\//i.test(address ?? '')
}

/** The rel for a link: the one given, with `noopener` added once for an external link; undefined when empty. */
export function relFor(rel: string | undefined, external: boolean): string | undefined {
  const words = (rel ?? '').split(/\s+/).filter(Boolean)
  if (external && !words.includes('noopener')) words.push('noopener')
  return words.length ? words.join(' ') : undefined
}

const trim = (path: string) => (path.length > 1 ? path.replace(/\/+$/, '') : path)

/**
 * Whether `address` is the page the browser is on: same origin and path, a
 * trailing slash aside. Browser only (it reads `location`); call it after
 * mounting, so the server's render and the first client render agree.
 */
export function isCurrentPage(address: string | undefined): boolean {
  if (!address || typeof location === 'undefined') return false
  const url = new URL(address, location.href)
  return url.origin === location.origin && trim(url.pathname) === trim(location.pathname)
}
