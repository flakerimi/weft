// The server half of the session islands: under Node, with no window, a
// Button with perm and DataTable's perm actions render nothing, since the
// server has no session, and nothing asks for one. session-island.test.ts
// hydrates this markup.
import { fileURLToPath } from 'node:url'
import { expect, it, vi } from 'vitest'
import { renderPage } from '../src/render.ts'
import { lateIsland, sessionIsland } from './fixtures/session-island/markup.ts'

it('renders perm buttons and row actions as nothing on the server, and asks nothing', async () => {
  expect(typeof window).toBe('undefined')
  const asked = vi.spyOn(globalThis, 'fetch')
  const root = fileURLToPath(new URL('./fixtures/session-island', import.meta.url))
  const { html } = await renderPage({ root, page: 'pages/index.vue' })
  expect(html).toContain(sessionIsland)
  expect(html).toContain(lateIsland)
  // Button with perm renders nothing on the server: only the one without perm is drawn.
  expect(sessionIsland.match(/<button/g)).toHaveLength(1)
  expect(sessionIsland).toContain('Always')
  expect(sessionIsland).not.toContain('Edit')
  expect(sessionIsland).not.toContain('Users')
  // The table: no Export, no New, no Actions column, no row action, no dialog.
  expect(lateIsland).not.toContain('<button')
  expect(lateIsland).not.toContain('Actions')
  expect(asked).not.toHaveBeenCalled()
  asked.mockRestore()
})
