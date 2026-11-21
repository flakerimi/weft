// A DatePicker whose popover chunk cannot be fetched (offline, a deploy
// that removed it) closes again rather than stay "open" with nothing drawn,
// so its button says it is closed and the next press tries the fetch again.
// Here the chunk fails once, then loads.
import { afterEach, expect, it, vi } from 'vitest'
import DatePickerCases from '../fixtures/dates/DatePickerCases.vue'
import { mount } from '../mount.ts'
import { $, overlays, reset, settle } from '../overlays/helpers.ts'
import { focusedIso } from './helpers.ts'

const chunk = vi.hoisted(() => ({ tries: 0 }))
vi.mock('../../src/dates/picker-panel.ts', async (importOriginal) => {
  chunk.tries++
  if (chunk.tries === 1) throw new Error('offline')
  return importOriginal()
})

let unmount: (() => void) | undefined
afterEach(() => {
  unmount?.()
  unmount = undefined
  reset()
})

const button = () => $<HTMLButtonElement>('.weft-date-picker__button')
const until = async (ok: () => boolean) => {
  for (let i = 0; i < 50 && !ok(); i++) await settle()
}

it('closes when its popover cannot be fetched, and the next press fetches it again and opens', async () => {
  const m = mount(DatePickerCases, { initial: '2026-11-21' })
  unmount = m.unmount
  await settle()
  button().click()
  await until(() => chunk.tries === 1 && m.exposed.open === false)
  expect(m.exposed.open).toBe(false)
  expect(button().getAttribute('aria-expanded')).toBe('false')
  expect(overlays()?.querySelector('.weft-popover') ?? null).toBeNull()

  button().focus()
  button().click()
  await until(() => !!overlays()?.querySelector('.weft-popover'))
  expect(chunk.tries).toBe(2)
  expect(m.exposed.open).toBe(true)
  expect(button().getAttribute('aria-expanded')).toBe('true')
  await until(() => focusedIso() === '2026-11-21')
  expect(focusedIso()).toBe('2026-11-21')
})
