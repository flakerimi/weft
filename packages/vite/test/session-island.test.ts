// @vitest-environment happy-dom
// The session islands hydrate the server's markup in place, with no mismatch
// warning, and only then draw what the session allows: the first island
// hydrates while GET /api/auth/me is on its way; the second wakes after the
// answer has come, and its first render must still be the server's. The
// server half is session-island-ssr.test.ts. The answer is the contract's
// Me JSON, stubbed.
import { afterEach, expect, it, vi } from 'vitest'
import type { Component } from 'vue'
import { startIslands } from '../src/island-runtime.ts'
import Late from './fixtures/session-island/components/Late.vue'
import Session from './fixtures/session-island/components/Session.vue'
import { lateIsland, sessionIsland } from './fixtures/session-island/markup.ts'

const settle = () => new Promise((r) => setTimeout(r, 0))

afterEach(() => {
  document.body.innerHTML = ''
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
})

const member = {
  id: 7,
  created_at: '2026-11-28T09:00:00Z',
  updated_at: '2026-11-28T09:00:00Z',
  email: 'mia@example.com',
  name: 'Mia',
  role: 'member',
  permissions: ['products:create', 'products:edit', 'products:read'],
}

it('hydrates perm buttons and row actions with no mismatch, then draws what the session allows', async () => {
  const warn = vi.spyOn(console, 'warn').mockImplementation(() => {})
  const error = vi.spyOn(console, 'error').mockImplementation(() => {})
  let answer!: () => void
  const fetchMock = vi.fn(
    (url: string) =>
      new Promise<Response>((resolve) => {
        if (url !== '/api/auth/me') throw new Error(`unexpected ${url}`)
        answer = () => resolve(new Response(JSON.stringify(member), { status: 200, headers: { 'content-type': 'application/json' } }))
      }),
  )
  vi.stubGlobal('fetch', fetchMock)
  document.body.innerHTML = `<main>${sessionIsland}${lateIsland}</main>`
  const [first, second] = document.querySelectorAll('weft-island')
  const before = (island: Element) =>
    Array.from(island.querySelectorAll('*')).filter((e) => e.tagName !== 'SCRIPT' && !e.closest('tbody') && !(e instanceof SVGElement && !(e instanceof SVGSVGElement)))
  const kept = [...before(first!), ...before(second!)]
  let wake!: () => void
  const late = new Promise<Component>((resolve) => (wake = () => resolve(Late as Component)))
  startIslands({ Session: () => Promise.resolve(Session as Component), Late: () => late })
  await settle()

  // The first island hydrated while the session was loading: nothing more is drawn.
  expect(first!.querySelector('script')).toBeNull()
  expect([...first!.querySelectorAll('button')].map((b) => b.textContent)).toEqual(['Always'])
  expect(fetchMock).toHaveBeenCalledTimes(1)

  // The session comes; the first island draws Edit, not Users.
  answer()
  await settle()
  expect([...first!.querySelectorAll('button')].map((b) => b.textContent)).toEqual(['Always', 'Edit'])

  // The second island wakes now, with the session known, and still hydrates the server's markup.
  wake()
  await settle()
  await settle()
  expect(second!.querySelector('script')).toBeNull()
  for (const el of kept) expect(el.isConnected, el.outerHTML).toBe(true)
  expect(warn).not.toHaveBeenCalled()
  expect(error).not.toHaveBeenCalled()

  // Then it draws what the member may do: Export, New and Edit; no Delete.
  const labels = [...second!.querySelectorAll('button')].map((b) => b.textContent!.trim())
  expect(labels).toEqual(expect.arrayContaining(['Export', 'New product', 'Edit']))
  expect(second!.querySelector('[data-weft-row-action="delete"]')).toBeNull()
  expect(second!.querySelector('[data-weft-row-action="edit"]')).not.toBeNull()
  expect([...second!.querySelectorAll('thead th')].map((th) => th.textContent!.trim())).toEqual(['Name', 'Actions'])
  // One question for the whole page.
  expect(fetchMock).toHaveBeenCalledTimes(1)
  expect(warn).not.toHaveBeenCalled()
  expect(error).not.toHaveBeenCalled()
})
