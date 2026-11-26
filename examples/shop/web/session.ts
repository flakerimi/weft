// The signed-in user, shared by every island of a page. Pages are built with
// no session, so whatever depends on who is signed in asks in the browser:
// useSession() sends api.auth.me once per page load, however many islands
// call it, and shares the answer.
import { reactive, readonly } from 'vue'
import { api, ApiError, type User } from './.weft-types/api.ts'
import { sessionFrom } from './answers.ts'

export interface Session {
  /** The signed-in user, or null when there is none or it is not known yet. */
  user: User | null
  /** True until the answer to api.auth.me has come. */
  loading: boolean
  /** Why the session could not be read, such as a server that cannot be reached; '' when it was. */
  problem: string
}

const state = reactive<Session>({ user: null, loading: true, problem: '' })
let asked: Promise<User | null> | undefined

async function ask(): Promise<User | null> {
  Object.assign(state, await sessionFrom(() => api.auth.me()), { loading: false })
  return state.user
}

/**
 * The session: `session` is reactive `{ user, loading, problem }`, `ready`
 * resolves with the user (or null) once known, and `signOut()` ends the
 * session and goes to /login. On the server, where pages are built, nothing
 * is asked and `ready` is null.
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
 * The user, or null. A visitor with no session is sent to /login, with this
 * page's path as `next` so that signing in comes back here; when the session
 * could not be read at all (`session.problem`), nobody is sent anywhere and
 * the island shows the problem.
 */
export async function requireUser(): Promise<User | null> {
  const user = await useSession().ready
  if (!user && !state.problem) location.assign(`/login?next=${encodeURIComponent(location.pathname + location.search)}`)
  return user
}
