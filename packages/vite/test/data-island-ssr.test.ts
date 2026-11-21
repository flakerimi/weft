// The server half of the data island: every feedback and data component
// renders under Node, with no window or document, to the markup
// data-island.test.ts hydrates. The table draws its first page as a table
// (the server cannot measure, so never as cards); the table that fetches
// draws busy skeleton rows, as the browser's first render does, and asks
// only once mounted.
import { fileURLToPath } from 'node:url'
import { expect, it } from 'vitest'
import { renderPage } from '../src/render.ts'
import { dataIsland } from './fixtures/data-island/markup.ts'

it('renders an island of every feedback and data component', async () => {
  expect(typeof window).toBe('undefined')
  const root = fileURLToPath(new URL('./fixtures/data-island', import.meta.url))
  const { html } = await renderPage({ root, page: 'pages/index.vue' })
  expect(html).toContain(dataIsland)
  expect(dataIsland).toContain('data-layout="table"')
  expect(dataIsland).not.toContain('data-layout="cards"')
  expect(dataIsland).toMatch(/<caption class="weft-table__caption">People<\/caption>/)
  // The remote table is busy, its skeleton rows hidden from assistive technology; fetch was not called.
  expect(dataIsland).toMatch(/aria-busy="true"[^>]*>\s*<caption class="weft-table__caption">Remote/)
  expect(dataIsland).toContain('class="weft-table__skeleton" aria-hidden="true"')
  expect(dataIsland).toContain('<output>|[]|0|false|left|true|0</output>')
  expect(dataIsland).toContain('role="alert"')
  expect(dataIsland).toContain('aria-current="step"')
  // The scroll area is a tab stop until it is measured.
  expect(dataIsland).toMatch(/class="weft-scroll-area"[^>]*tabindex="0"/)
})
