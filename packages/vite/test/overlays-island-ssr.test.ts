// The server half of the overlays island: every overlay component renders
// under Node, with no window or document, to the markup
// overlays-island.test.ts hydrates. Only triggers: no overlay content, not
// even for a popover that starts open, and no #weft-overlays root.
import { fileURLToPath } from 'node:url'
import { expect, it } from 'vitest'
import { renderPage } from '../src/render.ts'
import { overlaysIsland } from './fixtures/overlays-island/markup.ts'

it('renders an island of every overlay component, triggers only', async () => {
  expect(typeof window).toBe('undefined')
  const root = fileURLToPath(new URL('./fixtures/overlays-island', import.meta.url))
  const { html } = await renderPage({ root, page: 'pages/index.vue' })
  expect(html).toContain(overlaysIsland)
  for (const content of ['role="dialog"', 'role="menu"', 'role="tooltip"', 'weft-overlays', 'weft-toaster', 'class="field"']) {
    expect(html, content).not.toContain(content)
  }
})
