// Following an island's named imports through an index is an optimisation: when
// the index's exports cannot be read (the SSR transform writes them in a form
// imports.ts does not know), the page takes the whole index, and every
// stylesheet under it, rather than none: never an unstyled island.
import { fileURLToPath } from 'node:url'
import { expect, it, vi } from 'vitest'

vi.mock('../src/imports.ts', async (load) => {
  const real = await load<typeof import('../src/imports.ts')>()
  // Reads @weft/ui's index (and every module) as one exporting nothing it can name.
  return { ...real, exportsOf: () => ({ names: new Map(), all: [] }) }
})

const { renderPage } = await import('../src/render.ts')
const root = fileURLToPath(new URL('./fixtures/css-site', import.meta.url))

it("styles an island whose index's exports cannot be read, taking the whole index", async () => {
  const { css } = await renderPage({ root, page: 'pages/barrel-button-island.vue' })
  // All of it: Button's sheet (drawn on the server too) and every other component's.
  expect(css).toMatch(/\.weft-slider\s*\{/)
})
