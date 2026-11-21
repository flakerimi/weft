// A DatePicker opened for the first time before its popover chunk is in
// (its touch sheet): nothing of the setup's preload, the chunk arrives a little later,
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

it('opens the touch sheet from its button, pressed at once', async () => {
  const m = mount(DatePickerCases, { input: 'touch', initial: '2026-11-21' })
  unmount = m.unmount
  await settle()
  $<HTMLButtonElement>('.weft-date-picker__trigger').click()
  await until(() => focusedIso() === '2026-11-21')
  const sheet = $('.weft-popover', overlays()!)
  expect(sheet.dataset.form).toBe('sheet')
  expect(focusedIso()).toBe('2026-11-21')
  expect(m.exposed.open).toBe(true)
})
