// Who is signed in, shared by every island of a page. Pages are built with no
// session, so whatever depends on who is signed in asks in the browser:
// useSession() sends GET /api/auth/me once per page load, however many islands
// and buttons call it, and shares the answer. The state is this module's, and
// the module imports only vue, so every island shares it through one small
// chunk. Button and DataTable load it through can.ts, only once one with
// `perm` has mounted, so a page with no `perm` never fetches it.
import { reactive, readonly } from 'vue'

/** The signed-in user, as GET /api/auth/me answers it: Warp's `auth.Me`, whose generated type is assignable to this one. */
export interface SessionUser {
  id: number
  email: string
  name: string
  role: string
  /** Every permission the role has, such as `products:edit`, sorted. */
  permissions: string[]
  created_at: string
  updated_at: string
}

export interface Session {
  /** The signed-in user, or null when there is none or it is not known yet. */
  user: SessionUser | null
  /** True until the answer to GET /api/auth/me has come; always true on the server, which never asks. */
  loading: boolean
  /** Why the session could not be read, such as a server that cannot be reached; '' when it was. */
  problem: string
}

const unreachable = 'The server could not be reached; check the connection and try again.'
const unanswered = 'The server could not answer right now; try again in a moment.'

const state = reactive<Session>({ user: null, loading: true, problem: '' })
const shown = readonly(state) as Readonly<Session>
/** The newest question asked, and its number, so an older answer that comes late is dropped. */
let asked: Promise<SessionUser | null> | undefined
let asking = 0
/**
 * Set once the page is on its way to another (signOut, signInAgain), so a
 * later redirect, such as a Guard's after a sign-out or several Guards' at
 * once, folds into the first; cleared when an answer brings a user again or
 * the page comes back from the back-forward cache.
 */
let leaving = false

const inBrowser = () => typeof window !== 'undefined'

/** The error shape's message from an answer that is not a success, or a sentence of ours when it has none. */
async function problemOf(res: Response): Promise<string> {
  try {
    const message = ((await res.json()) as { error?: { message?: unknown } } | null)?.error?.message
    if (typeof message === 'string' && message !== '') return message
  } catch {
    // Not JSON: a proxy's page, say.
  }
  return unanswered
}

/** A user as the contract shapes one, or null. */
function userOf(body: unknown): SessionUser | null {
  if (typeof body !== 'object' || body === null) return null
  const u = body as Partial<SessionUser>
  return typeof u.id === 'number' && typeof u.role === 'string' && Array.isArray(u.permissions) ? (u as SessionUser) : null
}

async function answer(): Promise<{ user: SessionUser | null; problem: string }> {
  let res: Response
  try {
    res = await fetch('/api/auth/me', { credentials: 'same-origin', headers: { accept: 'application/json' } })
  } catch {
    return { user: null, problem: unreachable }
  }
  if (res.status === 401) return { user: null, problem: '' }
  if (!res.ok) return { user: null, problem: await problemOf(res) }
  const user = userOf(await res.json().catch(() => null))
  return user ? { user, problem: '' } : { user: null, problem: unanswered }
}

let listening = false

/**
 * Asks who is signed in. Only the newest question's answer is kept: an older
 * one that comes late is dropped, and whoever waits on it gets the newest
 * instead, never the state of a question already overtaken. The first time,
 * it starts listening for the page coming back, shown again or restored from
 * the back-forward cache, and asks again then, so a tab left open after a
 * sign-out elsewhere does not keep showing the old user.
 */
function ask(): Promise<SessionUser | null> {
  const n = ++asking
  const mine: Promise<SessionUser | null> = answer().then((got) => {
    if (n !== asking) return asked!
    Object.assign(state, got, { loading: false })
    if (state.user) leaving = false
    return state.user
  })
  asked = mine
  if (!listening) {
    listening = true
    document.addEventListener('visibilitychange', () => void (document.visibilityState === 'visible' && ask()))
    addEventListener('pageshow', (e) => void ((e as PageTransitionEvent).persisted && ((leaving = false), ask())))
  }
  return mine
}

/** True when the signed-in user has `permission`, always for the role owner; false while loading or signed out. Reactive. */
function can(permission: string): boolean {
  const user = state.user
  return !!user && (user.role === 'owner' || user.permissions.includes(permission))
}

/** Asks again, after a sign-in or a role change; resolves with the user, or null. Nothing on the server. */
function refresh(): Promise<SessionUser | null> {
  return inBrowser() ? ask() : Promise.resolve(null)
}

async function signOut(): Promise<string> {
  let res: Response
  try {
    res = await fetch('/api/auth/logout', { method: 'POST', credentials: 'same-origin', headers: { accept: 'application/json' } })
  } catch {
    return unreachable
  }
  // A 401 means the session had already ended, which is what was asked.
  if (!res.ok && res.status !== 401) return await problemOf(res)
  asking++
  asked = Promise.resolve(null)
  Object.assign(state, { user: null, problem: '', loading: false })
  leaving = true
  location.assign('/login')
  return ''
}

/**
 * The session: `session` is reactive `{ user, loading, problem }`; `ready`
 * resolves with the user (or null) once known; `can(permission)` says whether
 * the user may; `signOut()` ends the session and goes to /login, or, when the
 * server could not be asked, stays signed in and resolves with what to show;
 * `refresh()` asks again. On the server, where pages are built, nothing is
 * asked, `loading` stays true and `ready` is null.
 */
export function useSession(): {
  session: Readonly<Session>
  ready: Promise<SessionUser | null>
  can: (permission: string) => boolean
  signOut: () => Promise<string>
  refresh: () => Promise<SessionUser | null>
} {
  if (inBrowser() && !asked) ask()
  return { session: shown, ready: asked ?? Promise.resolve(null), can, signOut, refresh }
}

/**
 * Sends the visitor to `signIn` (default /login), with this page's path as
 * `next`, so that signing in comes back here; nothing when the page is
 * already leaving, after signOut or an earlier call.
 */
export function signInAgain(signIn = '/login'): void {
  if (leaving) return
  leaving = true
  const next = encodeURIComponent(location.pathname + location.search)
  location.assign(`${signIn}${signIn.includes('?') ? '&' : '?'}next=${next}`)
}

/**
 * The signed-in user, or null. A visitor with no session is sent to `signIn`
 * with `next`; a user without `perm` resolves null and stays where they are,
 * for the page to say why; when the session could not be read at all
 * (`session.problem`), nobody is sent anywhere. On the server, null.
 */
export async function requireSession(options: { perm?: string; signIn?: string } = {}): Promise<SessionUser | null> {
  if (!inBrowser()) return null
  await useSession().ready
  // What the session is now, not what the awaited answer was: it may have ended meanwhile.
  const user = state.user
  if (!user) {
    if (!state.problem) signInAgain(options.signIn)
    return null
  }
  return options.perm && !can(options.perm) ? null : user
}

/**
 * For a call answered 401: the session ended while the page was open, signed
 * out in another window, ended from another device, or past its days. Asks
 * who is signed in again, so `perm` buttons and Guards follow at once, and
 * sends the visitor to `signIn` (default /login) with `next` when no one is;
 * a session that could not be read sends no one. Nothing on the server.
 */
export async function sessionEnded(signIn?: string): Promise<void> {
  await refresh()
  await requireSession({ signIn })
}
