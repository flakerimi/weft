// A DatePicker opened for the first time before its popover chunk is in
// (reached by focus): nothing of the setup's preload, the chunk arrives a little later,
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

it('starts loading on focus, and opens from the button once it is in', async () => {
  const m = mount(DatePickerCases, { initial: '2026-11-21' })
  unmount = m.unmount
  await settle()
  $<HTMLInputElement>('.weft-date-picker input.weft-input__field').focus()
  await settle()
  expect(chunk.loads).toBe(1)
  const button = $<HTMLButtonElement>('.weft-date-picker__button')
  button.focus()
  button.click()
  await until(() => focusedIso() === '2026-11-21')
  expect($('.weft-popover', overlays()!).getAttribute('role')).toBe('dialog')
  expect(focusedIso()).toBe('2026-11-21')
  expect(chunk.loads).toBe(1)
})
