// The app's rule for what a form says when a call fails: a plain function
// with no Vue and no browser globals, so a test can run it under Node. Who is
// signed in is useSession, and where to go after signing in nextPath, both
// from @weft/ui.
import { ApiError } from './.weft-types/api.ts'

/** What the app says when the request never reached the server. */
export const unreachable = 'The server could not be reached; check the connection and try again.'

/** What the app says when its own code failed, which is logged for whoever looks at the console. */
export const pageFailed = 'Something went wrong on this page; reload it and try again.'

/**
 * What a form shows for a failed call: an ApiError's field messages by the
 * field's name, or its message when it has none. A request that never reached
 * the server (code network) gets one sentence of the app's, not fetch's own
 * words. Anything that is not an ApiError is a bug in the page's own code: it
 * is logged with console.error, and the form says the page failed.
 *
 * A 401 means the session ended while the page was open (signed out in
 * another window, or past its days), so `ended` is called, which a page
 * signed in gives as `sessionEnded` from @weft/ui: the session is asked
 * again, so `perm` buttons and Guards follow, and the visitor is sent to sign
 * in. The login form gives none, since its 401 is a wrong password.
 */
export function formProblem(e: unknown, ended?: () => unknown): { fields: Record<string, string>; message: string } {
  if (!(e instanceof ApiError)) {
    console.error('The page failed:', e)
    return { fields: {}, message: pageFailed }
  }
  if (e.code === 'network') return { fields: {}, message: unreachable }
  if (e.status === 401) ended?.()
  return Object.keys(e.fields).length > 0 ? { fields: e.fields, message: '' } : { fields: {}, message: e.message }
}
