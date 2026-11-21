// The server half of the dates island: every date and file component renders
// under Node, with no window or document, to the markup dates-island.test.ts
// hydrates. No calendar that opens, no #weft-overlays root, and no today
// marker: the server and the browser may not agree on the day.
import { fileURLToPath } from 'node:url'
import { expect, it } from 'vitest'
import { renderPage } from '../src/render.ts'
import { datesIsland } from './fixtures/dates-island/markup.ts'

it('renders an island of every date and file component, in its locale', async () => {
  expect(typeof window).toBe('undefined')
  const root = fileURLToPath(new URL('./fixtures/dates-island', import.meta.url))
  const { html } = await renderPage({ root, page: 'pages/index.vue' })
  expect(html).toContain(datesIsland)
  expect(html).not.toContain('weft-overlays')
  expect(html).not.toContain('weft-popover')
  expect(html).not.toContain('aria-current')
  // en-GB, from the WeftProvider: the grid starts on Monday, the field writes day first.
  expect(datesIsland).toContain('abbr="Monday"')
  expect(datesIsland).toContain('value="01/03/2024"')
  expect(datesIsland).not.toContain('style=""')
})
