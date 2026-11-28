// useSession and requireSession: who is signed in, asked once per page load
// from GET /api/auth/me, whose answer is the contract's Me JSON (a stub here).
// The session is module state, so each test loads a fresh copy of the module.
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { nextTick } from 'vue'

type SessionModule = typeof import('../../src/session/session.ts')

const member = {
  id: 7,
  created_at: '2026-11-28T09:00:00Z',
  updated_at: '2026-11-28T09:00:00Z',
  email: 'member@example.com',
  name: 'Mia',
  role: 'member',
  permissions: ['products:create', 'products:delete', 'products:edit', 'products:read'],
}
const owner = { ...member, id: 1, email: 'owner@example.com', name: 'Olu', role: 'owner', permissions: ['products:read'] }

const json = (status: number, body?: unknown) =>
  new Response(body === undefined ? null : JSON.stringify(body), {
    status,
    headers: body === undefined ? {} : { 'content-type': 'application/json' },
  })
const unauthorized = () => json(401, { error: { code: 'unauthorized', message: 'Sign in to do that.' } })

let fetchMock: ReturnType<typeof vi.fn>
let assign: ReturnType<typeof vi.spyOn>

async function load(): Promise<SessionModule> {
  vi.resetModules()
  return await import('../../src/session/session.ts')
}

// Each test loads a fresh copy of the module, which listens on document and
// window; the listeners each copy adds are taken off after its test.
let listening: [EventTarget, string, EventListenerOrEventListenerObject][] = []
beforeEach(() => {
  for (const target of [document, window]) {
    const add = target.addEventListener.bind(target)
    vi.spyOn(target, 'addEventListener').mockImplementation((type: string, listener: EventListenerOrEventListenerObject | null, options?: unknown) => {
      if (!listener) return
      listening.push([target, type, listener])
      add(type, listener, options as AddEventListenerOptions)
    })
  }
  fetchMock = vi.fn()
  vi.stubGlobal('fetch', fetchMock)
  assign = vi.spyOn(window.location, 'assign').mockImplementation(() => {})
  history.replaceState(null, '', '/products/3?tab=stock')
})
afterEach(() => {
  for (const [target, type, listener] of listening) target.removeEventListener(type, listener)
  listening = []
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
})

describe('useSession', () => {
  it('asks me once for many callers', async () => {
    fetchMock.mockResolvedValue(json(200, member))
    const { useSession } = await load()
    const a = useSession()
    const b = useSession()
    expect(a.session.loading).toBe(true)
    expect(await a.ready).toEqual(member)
    expect(await b.ready).toEqual(member)
    useSession()
    expect(fetchMock).toHaveBeenCalledTimes(1)
    const [url, init] = fetchMock.mock.calls[0]!
    expect(url).toBe('/api/auth/me')
    expect(init).toMatchObject({ credentials: 'same-origin', headers: { accept: 'application/json' } })
    expect(a.session).toMatchObject({ user: member, loading: false, problem: '' })
  })

  it('401 is signed out, not a problem', async () => {
    fetchMock.mockResolvedValue(unauthorized())
    const { useSession } = await load()
    const { session, ready } = useSession()
    expect(await ready).toBeNull()
    expect(session).toMatchObject({ user: null, loading: false, problem: '' })
  })

  it('network failure is a problem and sends no one away', async () => {
    fetchMock.mockRejectedValue(new TypeError('Failed to fetch'))
    const { useSession, requireSession } = await load()
    expect(await requireSession()).toBeNull()
    expect(useSession().session).toMatchObject({
      user: null,
      loading: false,
      problem: 'The server could not be reached; check the connection and try again.',
    })
    expect(assign).not.toHaveBeenCalled()
  })

  it("another status is the error shape's message, or a sentence of its own", async () => {
    fetchMock.mockResolvedValue(json(503, { error: { code: 'unavailable', message: 'The database is busy; try again in a moment.' } }))
    let s = await load()
    await s.useSession().ready
    expect(s.useSession().session.problem).toBe('The database is busy; try again in a moment.')

    fetchMock.mockResolvedValue(new Response('<html>Bad gateway</html>', { status: 502 }))
    s = await load()
    await s.useSession().ready
    expect(s.useSession().session.problem).toBe('The server could not answer right now; try again in a moment.')

    // A 200 that is not a user is not taken for one.
    fetchMock.mockResolvedValue(json(200, { hello: 'world' }))
    s = await load()
    expect(await s.useSession().ready).toBeNull()
    expect(s.useSession().session.problem).toBe('The server could not answer right now; try again in a moment.')
  })

  it('can follows permissions and owner', async () => {
    let resolve!: (r: Response) => void
    fetchMock.mockReturnValue(new Promise<Response>((r) => (resolve = r)))
    let s = await load()
    const { can, ready } = s.useSession()
    // Not known yet: nothing is allowed.
    expect(can('products:read')).toBe(false)
    resolve(json(200, member))
    await ready
    expect(can('products:edit')).toBe(true)
    expect(can('users:read')).toBe(false)

    fetchMock.mockResolvedValue(json(200, owner))
    s = await load()
    await s.useSession().ready
    // The owner may do anything, as the server's guard says, listed or not.
    expect(s.useSession().can('users:delete')).toBe(true)
    expect(s.useSession().can('reports:export')).toBe(true)

    fetchMock.mockResolvedValue(unauthorized())
    s = await load()
    await s.useSession().ready
    expect(s.useSession().can('products:read')).toBe(false)
  })

  it('can is reactive', async () => {
    fetchMock.mockResolvedValue(json(200, { ...member, permissions: [] }))
    const { useSession } = await load()
    const { can, ready, refresh } = useSession()
    const { computed } = await import('vue')
    const allowed = computed(() => can('products:edit'))
    await ready
    expect(allowed.value).toBe(false)
    fetchMock.mockResolvedValue(json(200, member))
    await refresh()
    await nextTick()
    expect(allowed.value).toBe(true)
  })

  it('signOut posts and leaves; a 401 counts as signed out', async () => {
    fetchMock.mockResolvedValueOnce(json(200, member)).mockResolvedValueOnce(json(204))
    let s = await load()
    await s.useSession().ready
    expect(await s.useSession().signOut()).toBe('')
    const [url, init] = fetchMock.mock.calls[1]!
    expect(url).toBe('/api/auth/logout')
    expect(init).toMatchObject({ method: 'POST', credentials: 'same-origin' })
    expect(s.useSession().session.user).toBeNull()
    expect(assign).toHaveBeenCalledWith('/login')

    // Already ended elsewhere: what was asked has happened.
    assign.mockClear()
    fetchMock.mockReset().mockResolvedValueOnce(json(200, member)).mockResolvedValueOnce(unauthorized())
    s = await load()
    await s.useSession().ready
    expect(await s.useSession().signOut()).toBe('')
    expect(assign).toHaveBeenCalledWith('/login')
  })

  it('signOut stays signed in and says why when the server cannot be asked', async () => {
    fetchMock.mockResolvedValueOnce(json(200, member)).mockRejectedValueOnce(new TypeError('Load failed'))
    let s = await load()
    await s.useSession().ready
    expect(await s.useSession().signOut()).toBe('The server could not be reached; check the connection and try again.')
    expect(s.useSession().session.user).toEqual(member)

    fetchMock.mockReset().mockResolvedValueOnce(json(200, member)).mockResolvedValueOnce(json(500, { error: { code: 'internal', message: 'Something went wrong on the server; its log has the details.' } }))
    s = await load()
    await s.useSession().ready
    expect(await s.useSession().signOut()).toBe('Something went wrong on the server; its log has the details.')
    expect(assign).not.toHaveBeenCalled()
  })

  it('refresh asks again', async () => {
    fetchMock.mockResolvedValueOnce(unauthorized()).mockResolvedValueOnce(json(200, member))
    const { useSession } = await load()
    const { session, ready, refresh } = useSession()
    expect(await ready).toBeNull()
    expect(await refresh()).toEqual(member)
    expect(fetchMock).toHaveBeenCalledTimes(2)
    expect(session.user).toEqual(member)
    // ready, asked after, is the newest answer.
    expect(await useSession().ready).toEqual(member)
  })

  it('refresh keeps only the newest answer', async () => {
    let first!: (r: Response) => void
    fetchMock.mockReturnValueOnce(new Promise<Response>((r) => (first = r))).mockResolvedValueOnce(json(200, member))
    const { useSession } = await load()
    const { session, refresh } = useSession()
    await refresh()
    first(unauthorized())
    await new Promise((r) => setTimeout(r, 0))
    expect(session.user).toEqual(member)
  })
})

describe('a session that changes while the page is open', () => {
  it('a refresh during the first load never sends a signed-in user away', async () => {
    let first!: (r: Response) => void
    let second!: (r: Response) => void
    fetchMock
      .mockReturnValueOnce(new Promise<Response>((r) => (first = r)))
      .mockReturnValueOnce(new Promise<Response>((r) => (second = r)))
    const { useSession, requireSession } = await load()
    const guarded = requireSession()
    const ready = useSession().ready
    void useSession().refresh()
    // The older answer comes first and is dropped; its waiters wait for the newest.
    first(unauthorized())
    await new Promise((r) => setTimeout(r, 0))
    expect(assign).not.toHaveBeenCalled()
    second(json(200, member))
    expect(await guarded).toEqual(member)
    expect(await ready).toEqual(member)
    expect(assign).not.toHaveBeenCalled()
    expect(useSession().session).toMatchObject({ user: member, loading: false, problem: '' })
  })

  it('after signOut, ready and requireSession give no one', async () => {
    fetchMock.mockResolvedValueOnce(json(200, member)).mockResolvedValueOnce(json(204))
    const { useSession, requireSession } = await load()
    await useSession().ready
    expect(await useSession().signOut()).toBe('')
    assign.mockClear()
    expect(await useSession().ready).toBeNull()
    expect(await requireSession()).toBeNull()
    expect(fetchMock).toHaveBeenCalledTimes(2)
  })

  it('asks again when the page is shown again, or restored from history', async () => {
    fetchMock.mockResolvedValueOnce(json(200, member))
    const { useSession } = await load()
    const { session } = useSession()
    await useSession().ready
    expect(fetchMock).toHaveBeenCalledTimes(1)

    // Another tab signed out: back on this one, it asks and the user is gone.
    fetchMock.mockResolvedValueOnce(unauthorized())
    const visibility = vi.spyOn(document, 'visibilityState', 'get')
    visibility.mockReturnValue('hidden')
    document.dispatchEvent(new Event('visibilitychange'))
    await new Promise((r) => setTimeout(r, 0))
    expect(fetchMock).toHaveBeenCalledTimes(1)
    visibility.mockReturnValue('visible')
    document.dispatchEvent(new Event('visibilitychange'))
    await new Promise((r) => setTimeout(r, 0))
    expect(fetchMock).toHaveBeenCalledTimes(2)
    expect(session.user).toBeNull()

    // A page from the back-forward cache asks; a fresh load's pageshow does not.
    // (happy-dom's PageTransitionEvent has no persisted, so the event is made by hand.)
    const pageshow = (persisted: boolean) => Object.defineProperty(new Event('pageshow'), 'persisted', { value: persisted })
    fetchMock.mockResolvedValueOnce(json(200, owner))
    window.dispatchEvent(pageshow(false))
    await new Promise((r) => setTimeout(r, 0))
    expect(fetchMock).toHaveBeenCalledTimes(2)
    window.dispatchEvent(pageshow(true))
    await new Promise((r) => setTimeout(r, 0))
    expect(fetchMock).toHaveBeenCalledTimes(3)
    expect(session.user).toEqual(owner)
  })

  it('listens once, however often it asks', async () => {
    fetchMock.mockImplementation(async () => json(200, member))
    const { useSession } = await load()
    await useSession().ready
    await useSession().refresh()
    await useSession().refresh()
    expect(listening.filter(([, type]) => type === 'visibilitychange')).toHaveLength(1)
    expect(listening.filter(([, type]) => type === 'pageshow')).toHaveLength(1)
  })
})

describe('requireSession', () => {
  it('requireSession sends a stranger to /login with next', async () => {
    fetchMock.mockResolvedValue(unauthorized())
    const { requireSession } = await load()
    expect(await requireSession()).toBeNull()
    expect(assign).toHaveBeenCalledWith('/login?next=%2Fproducts%2F3%3Ftab%3Dstock')
  })

  it('sends to the sign-in page it is given', async () => {
    fetchMock.mockResolvedValue(unauthorized())
    const { requireSession } = await load()
    await requireSession({ signIn: '/account/sign-in' })
    expect(assign).toHaveBeenLastCalledWith('/account/sign-in?next=%2Fproducts%2F3%3Ftab%3Dstock')
    const { signInAgain } = await load()
    signInAgain('/enter?from=menu')
    expect(assign).toHaveBeenLastCalledWith('/enter?from=menu&next=%2Fproducts%2F3%3Ftab%3Dstock')
  })

  it('a page leaves once: later redirects fold into the first', async () => {
    fetchMock.mockResolvedValue(unauthorized())
    const { requireSession, signInAgain } = await load()
    await Promise.all([requireSession(), requireSession({ perm: 'users:read' }), requireSession({ signIn: '/enter' })])
    signInAgain()
    expect(assign).toHaveBeenCalledTimes(1)
    expect(assign).toHaveBeenCalledWith('/login?next=%2Fproducts%2F3%3Ftab%3Dstock')
  })

  it('after signOut, requireSession sends no one to sign in with next', async () => {
    fetchMock.mockResolvedValueOnce(json(200, member)).mockResolvedValueOnce(json(204)).mockResolvedValue(unauthorized())
    const { useSession, requireSession } = await load()
    await useSession().ready
    expect(await useSession().signOut()).toBe('')
    await requireSession()
    await useSession().refresh()
    await requireSession()
    expect(assign).toHaveBeenCalledTimes(1)
    expect(assign).toHaveBeenCalledWith('/login')
  })

  it('a page back from history, or signed in again, may leave again', async () => {
    fetchMock.mockResolvedValue(unauthorized())
    const { requireSession, useSession } = await load()
    await requireSession()
    expect(assign).toHaveBeenCalledTimes(1)
    window.dispatchEvent(Object.defineProperty(new Event('pageshow'), 'persisted', { value: true }))
    await new Promise((r) => setTimeout(r, 0))
    await requireSession()
    expect(assign).toHaveBeenCalledTimes(2)

    fetchMock.mockResolvedValueOnce(json(200, member))
    await useSession().refresh()
    await useSession().refresh()
    await requireSession()
    expect(assign).toHaveBeenCalledTimes(3)
  })
})

describe('sessionEnded', () => {
  it('asks again, and sends to sign in when no one is signed in now', async () => {
    fetchMock.mockResolvedValueOnce(json(200, member)).mockResolvedValueOnce(unauthorized())
    const { useSession, sessionEnded } = await load()
    const { session } = useSession()
    await useSession().ready
    await sessionEnded()
    expect(fetchMock).toHaveBeenCalledTimes(2)
    expect(session.user).toBeNull()
    expect(assign).toHaveBeenCalledWith('/login?next=%2Fproducts%2F3%3Ftab%3Dstock')

    // To the sign-in page it is given.
    fetchMock.mockReset().mockResolvedValueOnce(json(200, member)).mockResolvedValueOnce(unauthorized())
    const again = await load()
    await again.useSession().ready
    await again.sessionEnded('/enter')
    expect(assign).toHaveBeenLastCalledWith('/enter?next=%2Fproducts%2F3%3Ftab%3Dstock')
  })

  it('stays when someone is still signed in, or the session cannot be read', async () => {
    fetchMock.mockResolvedValueOnce(json(200, member)).mockResolvedValueOnce(json(200, owner))
    let s = await load()
    await s.useSession().ready
    await s.sessionEnded()
    expect(s.useSession().session.user).toEqual(owner)

    fetchMock.mockReset().mockResolvedValueOnce(json(200, member)).mockRejectedValueOnce(new TypeError('Failed to fetch'))
    s = await load()
    await s.useSession().ready
    await s.sessionEnded()
    expect(s.useSession().session.problem).toBe('The server could not be reached; check the connection and try again.')
    expect(assign).not.toHaveBeenCalled()
  })

  it('requireSession without the perm stays', async () => {
    fetchMock.mockResolvedValue(json(200, member))
    const { requireSession } = await load()
    expect(await requireSession({ perm: 'users:read' })).toBeNull()
    expect(assign).not.toHaveBeenCalled()
    expect(await requireSession({ perm: 'products:read' })).toEqual(member)
    expect(await requireSession()).toEqual(member)
    expect(fetchMock).toHaveBeenCalledTimes(1)
  })
})

describe('on the server', () => {
  it('on the server nothing is asked', async () => {
    vi.stubGlobal('window', undefined)
    const { useSession, requireSession, sessionEnded } = await load()
    await sessionEnded()
    const { session, ready, can } = useSession()
    expect(await ready).toBeNull()
    expect(await requireSession({ perm: 'products:read' })).toBeNull()
    expect(session).toMatchObject({ user: null, loading: true, problem: '' })
    expect(can('products:read')).toBe(false)
    expect(fetchMock).not.toHaveBeenCalled()
    expect(assign).not.toHaveBeenCalled()
  })
})
