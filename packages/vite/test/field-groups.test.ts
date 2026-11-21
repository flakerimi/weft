// A FormField's label over a group (radios, checkboxes, a slider), as the
// server draws it: the label is drawn before the group says it is named by
// the label, so it keeps its `for`; that must name an element on the page
// (the group's fieldset, the slider's thumb), never an id that is not there.
import { fileURLToPath } from 'node:url'
import { expect, it } from 'vitest'
import { renderPage } from '../src/render.ts'

it("points every label's for at an element on the page", async () => {
  const root = fileURLToPath(new URL('./fixtures/field-groups', import.meta.url))
  const { html } = await renderPage({ root, page: 'pages/index.vue' })
  const targets = [...html.matchAll(/<label\b[^>]*\sfor="([^"]+)"/g)].map((m) => m[1]!)
  expect(targets.length).toBeGreaterThan(0)
  for (const id of targets) expect(html, `label for="${id}"`).toContain(`id="${id}"`)
})
