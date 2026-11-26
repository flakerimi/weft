// The server half of the choices island: radios and selects render under
// Node to the markup keep-typed.test.ts hydrates.
import { fileURLToPath } from 'node:url'
import { expect, it } from 'vitest'
import { renderPage } from '../src/render.ts'
import { choicesIsland } from './fixtures/choices-island/markup.ts'

it('renders the choices island', async () => {
  const root = fileURLToPath(new URL('./fixtures/choices-island', import.meta.url))
  const { html } = await renderPage({ root, page: 'pages/index.vue' })
  expect(html).toContain(choicesIsland)
})
