// The server half of the selection island: every selection component renders
// under Node, with no window or document, to the markup
// selection-island.test.ts hydrates. Fields and always-shown lists only: no
// list that opens, and no #weft-overlays root.
import { fileURLToPath } from 'node:url'
import { expect, it } from 'vitest'
import { renderPage } from '../src/render.ts'
import { selectionIsland } from './fixtures/selection-island/markup.ts'

it('renders an island of every selection component, the lists that open left out', async () => {
  expect(typeof window).toBe('undefined')
  const root = fileURLToPath(new URL('./fixtures/selection-island', import.meta.url))
  const { html } = await renderPage({ root, page: 'pages/index.vue' })
  expect(html).toContain(selectionIsland)
  expect(html).not.toContain('weft-overlays')
  expect(html).not.toContain('weft-popover')
  // The long Listbox draws only its first rows on the server, as the browser's first render does.
  const options = selectionIsland.match(/role="option"/g) ?? []
  expect(options.length).toBeGreaterThan(10)
  expect(options.length).toBeLessThan(30)
  expect(selectionIsland).toContain('aria-setsize="300"')
})
