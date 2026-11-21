// The server half of the in-place island: menus with :portal="false" that
// start open are drawn open where they stand, the pointer one floating and
// unpositioned (hanging from its trigger's place), the touch one a sheet
// over its backdrop; nothing goes to #weft-overlays.
import { fileURLToPath } from 'node:url'
import { expect, it } from 'vitest'
import { renderPage } from '../src/render.ts'
import { inPlaceIsland } from './fixtures/in-place-island/markup.ts'

it('draws both menus open in place, in the island', async () => {
  expect(typeof window).toBe('undefined')
  const root = fileURLToPath(new URL('./fixtures/in-place-island', import.meta.url))
  const { html } = await renderPage({ root, page: 'pages/index.vue' })
  expect(html).toContain(inPlaceIsland)
  expect(html).toContain('data-in-place="bottom-start"')
  expect(html).toContain('data-form="sheet"')
  expect(html).toContain('weft-overlay-backdrop')
  expect(html).not.toContain('weft-overlays')
})
