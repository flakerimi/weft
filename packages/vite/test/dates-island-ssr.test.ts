// The server half of the dates island: every date and file component renders
// under Node, with no window or document, to the markup dates-island.test.ts
// hydrates. No calendar that opens, no #weft-overlays root, and no today
// marker: the server and the browser may not agree on the day.
import { fileURLToPath } from 'node:url'
import { afterEach, expect, it, vi } from 'vitest'
import { renderPage } from '../src/render.ts'
import { datesIsland } from './fixtures/dates-island/markup.ts'

afterEach(() => vi.useRealTimers())
const root = fileURLToPath(new URL('./fixtures/dates-island', import.meta.url))
/** The page rendered as if built on `day`. */
async function renderOn(day: string) {
  vi.useFakeTimers({ toFake: ['Date'] })
  vi.setSystemTime(new Date(`${day}T12:00:00`))
  const { html } = await renderPage({ root, page: 'pages/index.vue' })
  vi.useRealTimers()
  return html
}

it('renders an island of every date and file component, in its locale', async () => {
  expect(typeof window).toBe('undefined')
  const html = await renderOn('2024-06-15')
  expect(html).toContain(datesIsland)
  expect(html).not.toContain('weft-overlays')
  expect(html).not.toContain('weft-popover')
  expect(html).not.toContain('aria-current')
  // en-GB, from the WeftProvider: the grid starts on Monday, the field writes day first.
  expect(datesIsland).toContain('abbr="Monday"')
  expect(datesIsland).toContain('value="01/03/2024"')
  expect(datesIsland).not.toContain('style=""')
})

it('renders a Calendar with no value the same whatever day it is built on: an empty, busy grid', async () => {
  const later = await renderOn('2023-01-01')
  expect(later).toContain(datesIsland)
  const loose = /<div class="weft-calendar loose"[\s\S]*?<\/table>/.exec(datesIsland)![0]
  expect(loose).toContain('aria-busy="true"')
  expect(loose).not.toContain('weft-calendar__day')
  // The month and year pickers the same: a busy grid of blank cells, no tab stop, no year.
  for (const cls of ['loose-month', 'loose-year']) {
    const picker = new RegExp(`<div class="weft-calendar ${cls}"[\\s\\S]*?</table>`).exec(datesIsland)![0]
    expect(picker, cls).toContain('aria-busy="true"')
    expect(picker, cls).not.toContain('weft-calendar__cell')
    expect(picker, cls).not.toContain('tabindex="0"')
    expect(picker, cls).not.toContain('2000')
  }
})
