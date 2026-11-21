// A DatePicker opened for the first time before its popover chunk is in
// (Alt+ArrowDown in the field): nothing of the setup's preload, the chunk arrives a little later,
// as from the network, and the picker opens once it is in, focus on the date.
import { afterEach, expect, it, vi } from 'vitest'
import DatePickerCases from '../fixtures/dates/DatePickerCases.vue'
import { mount } from '../mount.ts'
import { type } from '../forms/helpers.ts'
import { $, key, overlays, reset, settle } from '../overlays/helpers.ts'
import { focusedIso } from './helpers.ts'

const chunk = vi.hoisted(() => ({ loads: 0 }))
vi.mock('../../src/dates/picker-panel.ts', async (importOriginal) => {
  chunk.loads++
  await new Promise((r) => setTimeout(r, 20))
  return importOriginal()
})

let unmount: (() => void) | undefined
afterEach(() => {
  unmount?.()
  unmount = undefined
  reset()
})

const until = async (ok: () => boolean) => {
  for (let i = 0; i < 60 && !ok(); i++) await new Promise((r) => setTimeout(r, 5))
}
void type
void key

it('opens with Alt+ArrowDown in the field as the first thing done, on the date typed', async () => {
  const m = mount(DatePickerCases, { locale: 'en-GB' })
  unmount = m.unmount
  await settle()
  const field = $<HTMLInputElement>('.weft-date-picker input.weft-input__field')
  field.focus()
  type(field, '2/3/2027')
  key('ArrowDown', { altKey: true })
  await until(() => focusedIso() === '2027-03-02')
  expect(m.exposed.open).toBe(true)
  expect(focusedIso()).toBe('2027-03-02')
})
