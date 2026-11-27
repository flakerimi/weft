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

beforeEach(() => {
  fetchMock = vi.fn()
  vi.stubGlobal('fetch', fetchMock)
  assign = vi.spyOn(window.location, 'assign').mockImplementation(() => {})
  history.replaceState(null, '', '/products/3?tab=stock')
})
afterEach(() => {
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

describe('requireSession', () => {
  it('requireSession sends a stranger to /login with next', async () => {
    fetchMock.mockResolvedValue(unauthorized())
    const { requireSession } = await load()
    expect(await requireSession()).toBeNull()
    expect(assign).toHaveBeenCalledWith('/login?next=%2Fproducts%2F3%3Ftab%3Dstock')
  })

  it('sends to the sign-in page it is given', async () => {
    fetchMock.mockResolvedValue(unauthorized())
    const { requireSession, signInAgain } = await load()
    await requireSession({ signIn: '/account/sign-in' })
    expect(assign).toHaveBeenLastCalledWith('/account/sign-in?next=%2Fproducts%2F3%3Ftab%3Dstock')
    signInAgain('/enter?from=menu')
    expect(assign).toHaveBeenLastCalledWith('/enter?from=menu&next=%2Fproducts%2F3%3Ftab%3Dstock')
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
    const { useSession, requireSession } = await load()
    const { session, ready, can } = useSession()
    expect(await ready).toBeNull()
    expect(await requireSession({ perm: 'products:read' })).toBeNull()
    expect(session).toMatchObject({ user: null, loading: true, problem: '' })
    expect(can('products:read')).toBe(false)
    expect(fetchMock).not.toHaveBeenCalled()
    expect(assign).not.toHaveBeenCalled()
  })
})
