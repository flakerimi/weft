// The server half of ids.test.ts: Island counts ids per island, by position.
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'
import { renderPage } from '../src/render.ts'

describe('ids in islands, on the server', () => {
  it('are counted per island on the server', async () => {
    const root = fileURLToPath(new URL('./fixtures/ids', import.meta.url))
    const { html } = await renderPage({ root, page: 'pages/index.vue' })
    const ids = [...html.matchAll(/<button[^>]*\bid="([^"]+)"/g)].map((m) => m[1])
    expect(ids).toEqual(['field-i0-0', 'field-i1-0'])
    expect(html).toMatch(/<main id="page-v-\w+"/)
    // The markup ids.test.ts hydrates.
    for (const n of [0, 1]) {
      expect(html).toContain(
        `<weft-island data-id="Labelled" data-when="load"><script type="application/json">{}</script>` +
          `<div><button type="button" id="field-i${n}-0">label</button><!----></div></weft-island>`,
      )
    }
  })
})
