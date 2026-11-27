/**
 * Where to go after signing in: `next` when it is a path on this site, such
 * as /notes/4?x=1#y, and / for anything else. The string returned is the
 * one that will be followed, so it is what is checked: it starts with one /
 * (not // or /\, which a browser reads as another host), and followed again
 * from `origin` it stays on `origin`. A value with a control character, a
 * space or a backslash is refused before parsing, since the parser would drop
 * or rewrite them; a dot segment (/.//host, /%2e//host) is caught after it,
 * where it has collapsed into //host.
 */
export function nextPath(next: string | null | undefined, origin: string): string {
  const home = '/'
  if (!next || /[\x00-\x20\x7f\\]/.test(next) || !next.startsWith('/') || next.startsWith('//')) return home
  let url: URL
  try {
    url = new URL(next, origin)
  } catch {
    return home
  }
  const path = url.pathname + url.search + url.hash
  if (url.origin !== origin || !path.startsWith('/') || path.startsWith('//') || path.startsWith('/\\')) return home
  return new URL(path, origin).origin === origin ? path : home
}
