// The shop's rules for what comes from the address and from the server: what
// a form says when a call fails, and which product an address names. Plain
// functions with no Vue and no browser globals, so test/answers.test.ts runs
// them under Node. Who is signed in is useSession, and where to go after
// signing in nextPath, both from @weft/ui.
import { ApiError } from './.weft-types/api.ts'

/** What the shop says when the request never reached the server. */
export const unreachable = 'The server could not be reached; check the connection and try again.'

/** What the shop says when its own code failed, which is logged for whoever looks at the console. */
export const pageFailed = 'Something went wrong on this page; reload it and try again.'

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
 * The product id an address names, or null: only digits with no leading zero,
 * as the server writes an id, so /products/0x2a, /products/4.0 and
 * /products/007 do not stand for 42, 4 and 7.
 */
export function productId(raw: string | undefined): number | null {
  if (!raw || !/^[1-9][0-9]*$/.test(raw)) return null
  const id = Number(raw)
  return Number.isSafeInteger(id) ? id : null
}
