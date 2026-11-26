// The shop's rules for what comes from the address and from the server: where
// to go after signing in, what a form says when a call fails, what the session
// is, and which product an address names. Plain functions with no Vue and no
// browser globals, so test/answers.test.ts runs them under Node.
import { ApiError, type User } from './.weft-types/api.ts'

/** What the shop says when the request never reached the server. */
export const unreachable = 'The server could not be reached; check the connection and try again.'

/** What the shop says when its own code failed, which is logged for whoever looks at the console. */
export const pageFailed = 'Something went wrong on this page; reload it and try again.'

/**
 * Where to go after signing in: `next` when it is a path on this site, such
 * as /products/4?x=1#y, and / for anything else. The string returned is the
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

/**
 * What a form shows for a failed call: an ApiError's field messages by the
 * field's name, or its message when it has none. A request that never reached
 * the server (code network) gets one sentence of the shop's, not fetch's own
 * words. Anything that is not an ApiError is a bug in the page's own code: it
 * is logged with console.error, and the form says the page failed.
 */
export function formProblem(e: unknown): { fields: Record<string, string>; message: string } {
  if (!(e instanceof ApiError)) {
    console.error('The shop failed:', e)
    return { fields: {}, message: pageFailed }
  }
  if (e.code === 'network') return { fields: {}, message: unreachable }
  return Object.keys(e.fields).length > 0 ? { fields: e.fields, message: '' } : { fields: {}, message: e.message }
}

/**
 * The session from `me`: the user; no user and no problem on a 401, which is
 * simply nobody signed in; or no user and what went wrong, so that a server
 * that cannot be reached is not mistaken for a visitor to send to /login.
 */
export async function sessionFrom(me: () => Promise<User>): Promise<{ user: User | null; problem: string }> {
  try {
    return { user: await me(), problem: '' }
  } catch (e) {
    if (e instanceof ApiError && e.status === 401) return { user: null, problem: '' }
    return { user: null, problem: formProblem(e).message }
  }
}

/**
 * The product id an address names, or null: only digits with no leading zero,
 * as the server writes an id, so /products/0x2a, /products/4.0 and
 * /products/007 do not stand for 42, 4 and 7.
 */
export function productId(raw: string | undefined): number | null {
  if (!raw || !/^[1-9][0-9]*$/.test(raw)) return null
  const id = Number(raw)
  return Number.isSafeInteger(id) ? id : null
}
