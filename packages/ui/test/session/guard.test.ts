// Guard: a page's islands shown only to a signed-in user with `perm`. Who is
// signed in comes from GET /api/auth/me (the contract's Me JSON, stubbed),
// asked once for this file: the session is module state, so these tests run
// in order, the first while it is still loading, and each later one asks
// again with refresh(). The server half is in packages/vite's session-island
// tests.
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { useSession } from '../../src/session/session.ts'
import GuardCases from '../fixtures/session/GuardCases.vue'
import { mount } from '../mount.ts'

const me = (permissions: string[], role = 'member') => ({
  id: 7,
  created_at: '2026-11-28T09:00:00Z',
  updated_at: '2026-11-28T09:00:00Z',
  email: 'mia@example.com',
  name: 'Mia',
  role,
  permissions,
})
const answer = (status: number, body: unknown) => new Response(JSON.stringify(body), { status, headers: { 'content-type': 'application/json' } })
const unauthorized = () => answer(401, { error: { code: 'unauthorized', message: 'Sign in to do that.' } })
const settle = () => new Promise((r) => setTimeout(r, 0))

const fetchMock = vi.fn()
let assign: ReturnType<typeof vi.spyOn>
let mounted: { unmount(): void }[] = []
beforeEach(() => {
  vi.stubGlobal('fetch', fetchMock)
  assign = vi.spyOn(window.location, 'assign').mockImplementation(() => {})
  history.replaceState(null, '', '/reports?year=2026')
})
afterEach(() => {
  mounted.forEach((m) => m.unmount())
  mounted = []
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
})
function guard(props: Record<string, unknown>) {
  const m = mount(GuardCases, props)
  mounted.push(m)
  return m.el
}
async function answerWith(res: () => Response) {
  fetchMock.mockImplementation(async () => res())
  await useSession().refresh()
  await settle()
}

describe('Guard', () => {
  it('loading first, on the server too', async () => {
    let resolve!: (r: Response) => void
    fetchMock.mockReturnValueOnce(new Promise<Response>((r) => (resolve = r)))
    const el = guard({ case: 'plain', perm: 'reports:read' })
    await settle()
    // A spinner that says what is happening, and nothing of the slot.
    const spinner = el.querySelector('.weft-spinner')!
    expect(spinner.getAttribute('role')).toBe('status')
    expect(spinner.textContent).toBe('Loading')
    expect(el.querySelector('.secret')).toBeNull()
    expect(fetchMock).toHaveBeenCalledTimes(1)
    resolve(answer(200, me(['reports:read'])))
    await settle()
    expect(el.querySelector('.weft-spinner')).toBeNull()
    expect(el.querySelector('.secret')!.textContent).toBe('For Mia')
  })

  it('with perm shows the slot', async () => {
    await answerWith(() => answer(200, me(['reports:read'])))
    const el = guard({ case: 'plain', perm: 'reports:read' })
    await settle()
    expect(el.querySelector('.secret')!.textContent).toBe('For Mia')
    expect(assign).not.toHaveBeenCalled()
    // Without perm, any signed-in user.
    const any = guard({ case: 'plain' })
    await settle()
    expect(any.querySelector('.secret')!.textContent).toBe('For Mia')
  })

  it('without perm shows denied', async () => {
    await answerWith(() => answer(200, me(['reports:read'])))
    const el = guard({ case: 'plain', perm: 'users:read' })
    await settle()
    expect(el.querySelector('.secret')).toBeNull()
    const alert = el.querySelector('.weft-alert')!
    expect(alert.textContent).toBe('Your account cannot see this; ask an administrator.')
    // It takes the place of a Spinner that said Loading, so it is announced as politely.
    expect(alert.getAttribute('role')).toBe('status')
    expect(assign).not.toHaveBeenCalled()
    // Its own words, and its own loading, by slot.
    const mine = guard({ case: 'slots', perm: 'users:read' })
    await settle()
    expect(mine.querySelector('.mine-denied')!.textContent).toBe('Not for you')
    expect(mine.querySelector('.secret')).toBeNull()
  })

  it('the owner sees everything, and a role change is followed', async () => {
    await answerWith(() => answer(200, me([], 'owner')))
    const el = guard({ case: 'plain', perm: 'users:delete' })
    await settle()
    expect(el.querySelector('.secret')).not.toBeNull()
    await answerWith(() => answer(200, me(['reports:read'], 'viewer')))
    expect(el.querySelector('.secret')).toBeNull()
    expect(el.querySelector('.weft-alert')!.textContent).toBe('Your account cannot see this; ask an administrator.')
  })

  it('signed out sends to sign-in with next', async () => {
    await answerWith(unauthorized)
    const el = guard({ case: 'plain', perm: 'reports:read' })
    await settle()
    expect(assign).toHaveBeenCalledTimes(1)
    expect(assign).toHaveBeenCalledWith('/login?next=%2Freports%3Fyear%3D2026')
    // While the browser leaves, it still says it is loading, and shows nothing of the slot.
    expect(el.querySelector('.weft-spinner')).not.toBeNull()
    expect(el.querySelector('.secret')).toBeNull()

    // Signed in again, then out: a Guard sends to the sign-in page it is given.
    mounted.pop()!.unmount()
    await answerWith(() => answer(200, me(['reports:read'])))
    await answerWith(unauthorized)
    assign.mockClear()
    guard({ case: 'plain', signIn: '/enter' })
    await settle()
    expect(assign).toHaveBeenCalledWith('/enter?next=%2Freports%3Fyear%3D2026')
  })

  it('several Guards send to sign in once', async () => {
    await answerWith(() => answer(200, me(['reports:read'])))
    for (const perm of ['reports:read', 'users:read', undefined]) guard({ case: 'plain', perm })
    await settle()
    await answerWith(unauthorized)
    expect(assign).toHaveBeenCalledTimes(1)
    expect(assign).toHaveBeenCalledWith('/login?next=%2Freports%3Fyear%3D2026')
  })

  it('signing out on a guarded page goes to /login alone', async () => {
    await answerWith(() => answer(200, me(['reports:read'])))
    const el = guard({ case: 'plain', perm: 'reports:read' })
    await settle()
    expect(el.querySelector('.secret')).not.toBeNull()
    fetchMock.mockResolvedValueOnce(new Response(null, { status: 204 }))
    expect(await useSession().signOut()).toBe('')
    await settle()
    expect(assign).toHaveBeenCalledTimes(1)
    expect(assign).toHaveBeenCalledWith('/login')
  })

  it('a session that ends mid-page sends to sign-in with next', async () => {
    await answerWith(() => answer(200, me(['reports:read'])))
    const el = guard({ case: 'plain', perm: 'reports:read' })
    await settle()
    expect(el.querySelector('.secret')!.textContent).toBe('For Mia')
    // Signed out elsewhere: the next answer is a 401.
    await answerWith(unauthorized)
    expect(el.querySelector('.secret')).toBeNull()
    expect(el.querySelector('.weft-spinner')).not.toBeNull()
    expect(assign).toHaveBeenCalledTimes(1)
    expect(assign).toHaveBeenCalledWith('/login?next=%2Freports%3Fyear%3D2026')
  })

  it('a problem shows it and sends no one away', async () => {
    await answerWith(() => answer(503, { error: { code: 'unavailable', message: 'The database is busy; try again in a moment.' } }))
    const el = guard({ case: 'plain', perm: 'reports:read' })
    await settle()
    const alert = el.querySelector('.weft-alert')!
    expect(alert.getAttribute('role')).toBe('alert')
    expect(alert.getAttribute('data-tone')).toBe('danger')
    expect(alert.textContent).toBe('The database is busy; try again in a moment.')
    expect(el.querySelector('.secret')).toBeNull()
    expect(assign).not.toHaveBeenCalled()

    fetchMock.mockRejectedValue(new TypeError('Failed to fetch'))
    await useSession().refresh()
    await settle()
    expect(el.querySelector('.weft-alert')!.textContent).toBe('The server could not be reached; check the connection and try again.')
    expect(assign).not.toHaveBeenCalled()
  })
})
