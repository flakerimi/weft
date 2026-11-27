// The server half of the session islands: under Node, with no window, a
// Button with perm and DataTable's perm actions render nothing, and a Guard
// renders its loading, since the server has no session, and nothing asks
// for one. session-island.test.ts
// hydrates this markup.
import { fileURLToPath } from 'node:url'
import { expect, it, vi } from 'vitest'
import { createRenderServer, renderPage } from '../src/render.ts'
import { lateIsland, sessionIsland } from './fixtures/session-island/markup.ts'

it('renders perm buttons and row actions as nothing, and Guards as loading, on the server, and asks nothing', async () => {
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
  // Every Guard is loading: a Spinner that says so, and nothing it holds.
  expect(sessionIsland.match(/class="weft-spinner" role="status"/g)).toHaveLength(2)
  expect(lateIsland.match(/class="weft-spinner" role="status"/g)).toHaveLength(1)
  for (const held of ['Stock', 'People', 'Signed in']) expect(sessionIsland + lateIsland).not.toContain(held)
  expect(asked).not.toHaveBeenCalled()
  asked.mockRestore()
})

// can.ts asks for the session only once a component has mounted: on the server a perm Button says no
// without calling import(). (Hydration's first render stays empty either way, as the import resolves after it.)
it('never loads the on-demand session on the server, where perm buttons only say no', async () => {
  const root = fileURLToPath(new URL('./fixtures/session-island', import.meta.url))
  const load = fileURLToPath(new URL('../../ui/src/session/load.ts', import.meta.url))
  const server = await createRenderServer(root)
  try {
    const { html } = await renderPage({ root, page: 'pages/index.vue', server })
    expect(html).toContain(sessionIsland)
    // Anything the render started would have come by now.
    for (let i = 0; i < 5; i++) await new Promise((r) => setTimeout(r, 20))
    // The page build transforms load.ts (it follows import() for CSS) but must never evaluate it.
    const evaluated = [...(server.moduleGraph.getModulesByFile(load) ?? [])].filter((m) => m.ssrModule)
    expect(evaluated).toHaveLength(0)
  } finally {
    await server.close()
  }
})
