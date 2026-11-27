// The server half of the label-slot island: FormFields whose label slot holds
// components that take ids (a Tooltip, a Checkbox). The server draws a
// field's label after its control, yet the ids it writes are the ones the
// browser counts in place (label-slot-island.test.ts hydrates this markup),
// each once on the page; and a control in a label is no control of the
// field: it takes neither the field's id nor its name.
import { fileURLToPath } from 'node:url'
import { Window } from 'happy-dom'
import { expect, it } from 'vitest'
import { renderPage } from '../src/render.ts'
import { labelSlotIsland } from './fixtures/label-slot-island/markup.ts'

it('renders the label slots of an island with ids of their own', async () => {
  expect(typeof window).toBe('undefined')
  const root = fileURLToPath(new URL('./fixtures/label-slot-island', import.meta.url))
  const { html } = await renderPage({ root, page: 'pages/index.vue' })
  expect(html).toContain(labelSlotIsland)
  const win = new Window()
  win.document.body.innerHTML = labelSlotIsland
  const doc = win.document as unknown as Document
  const ids = [...doc.querySelectorAll('[id]')].map((e) => e.id)
  expect(ids.filter((id, i) => ids.indexOf(id) !== i), 'ids used twice').toEqual([])
  // The island counts its ids as if the labels held nothing (the slider,
  // the radio group and the input take one each); what a label holds counts
  // in the label's own scope.
  expect([...doc.querySelectorAll('.weft-form-field__label')].map((l) => l.id)).toEqual([0, 2, 4, 6].map((i) => `field-i0-${i}-label`))
  const box = doc.querySelector<HTMLInputElement>('.weft-form-field__label .weft-checkbox__input')!
  expect(box.id).toBe('checkbox-field-i0-2-label-0')
  // The Checkbox in the Plan label is its own: the radio group keeps the field's id and name.
  expect(box.id).not.toBe(doc.querySelector('fieldset')!.id)
  expect(box.getAttribute('name')).toBeNull()
})
