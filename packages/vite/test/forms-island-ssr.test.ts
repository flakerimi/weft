// The server half of the forms island: every form component renders under
// Node, with no window or document, to the markup forms-island.test.ts
// hydrates. Ids come from the island's own scope (field-i0-0, …).
import { fileURLToPath } from 'node:url'
import { expect, it } from 'vitest'
import { renderPage } from '../src/render.ts'
import { formsIsland } from './fixtures/forms-island/markup.ts'

it('renders an island of every form component', async () => {
  expect(typeof window).toBe('undefined')
  const root = fileURLToPath(new URL('./fixtures/forms-island', import.meta.url))
  const { html } = await renderPage({ root, page: 'pages/index.vue' })
  expect(html).toContain(formsIsland)
  // A control given no style renders no empty style attribute.
  expect(formsIsland).not.toContain('style=""')
})
