// The signed-in user, shared by every island of a page. Pages are built with
// no session, so whatever depends on who is signed in asks in the browser:
// useSession() sends api.auth.me once per page load, however many islands
// call it, and shares the answer.
import { reactive, readonly } from 'vue'
import { api, ApiError, type User } from './.weft-types/api.ts'

export interface Session {
  /** The signed-in user, or null when there is none or it is not known yet. */
  user: User | null
  /** True until the answer to api.auth.me has come. */
  loading: boolean
}

const state = reactive<Session>({ user: null, loading: true })
let asked: Promise<User | null> | undefined

/** Asks api.auth.me: a 401 is no session; any other failure is said in the console and counts as none. */
async function ask(): Promise<User | null> {
  try {
    state.user = await api.auth.me()
  } catch (e) {
    if (!(e instanceof ApiError && e.status === 401)) console.error('shop: the session could not be read', e)
    state.user = null
  }
  state.loading = false
  return state.user
}

/**
 * The session: `session` is reactive `{ user, loading }`, `ready` resolves
 * with the user (or null) once known, and `signOut()` ends the session and
 * goes to /login. On the server, where pages are built, nothing is asked and
 * `ready` is null.
 */
export function useSession(): { session: Readonly<Session>; ready: Promise<User | null>; signOut: () => Promise<void> } {
  if (typeof window !== 'undefined') asked ??= ask()
  return { session: readonly(state) as Readonly<Session>, ready: asked ?? Promise.resolve(null), signOut }
}

async function signOut(): Promise<void> {
  try {
    await api.auth.logout()
  } catch (e) {
    // A 401 means the session had already ended, which is what was asked.
    if (!(e instanceof ApiError && e.status === 401)) throw e
  }
  state.user = null
  location.assign('/login')
}

/**
 * The user, or null after sending a visitor with no session to /login, with
 * this page's path as `next` so that signing in comes back here.
 */
export async function requireUser(): Promise<User | null> {
  const user = await useSession().ready
  if (!user) location.assign(`/login?next=${encodeURIComponent(location.pathname + location.search)}`)
  return user
}

/**
 * Where to go after signing in: `next` when it is a path on this site, such
 * as /products/4, and / for anything else (another site, //host, a scheme).
 */
export function nextPath(next: string | null): string {
  if (!next || !next.startsWith('/') || next.startsWith('//') || next.startsWith('/\\')) return '/'
  // Resolved against this page, it must stay on this origin.
  const url = new URL(next, location.origin)
  return url.origin === location.origin ? url.pathname + url.search + url.hash : '/'
}

/**
 * What a form shows for a failed call: an ApiError's field messages, by the
 * field's name, and its message when it has none; any other failure is one
 * sentence.
 */
export function formProblem(e: unknown): { fields: Record<string, string>; message: string } {
  if (e instanceof ApiError) {
    const fields = e.fields
    return Object.keys(fields).length > 0 ? { fields, message: '' } : { fields: {}, message: e.message }
  }
  console.error('shop: the request failed', e)
  return { fields: {}, message: 'The server could not be reached; check the connection and try again.' }
}
