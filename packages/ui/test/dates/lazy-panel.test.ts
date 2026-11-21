// A DatePicker draws its Popover and Calendar only while open, and loads
// them (src/dates/picker-panel.ts) on demand. Here the module is held back
// until the test lets it through, as a slow network would: nothing of the
// popover is drawn while it is on its way, it starts loading as soon as the
// pointer or focus comes to the picker, and once it is in the popover opens
// as ever, focus on the chosen date. (test/setup.ts loads it up front for
// every other test.)
import { afterEach, describe, expect, it, vi } from 'vitest'
import DatePickerCases from '../fixtures/dates/DatePickerCases.vue'
import { mount } from '../mount.ts'
import { $, overlays, reset, settle } from '../overlays/helpers.ts'
import { focusedIso } from './helpers.ts'

const panel = vi.hoisted(() => {
  let release = () => {}
  const gate = new Promise<void>((r) => (release = r))
  return { gate, release: () => release(), loads: 0 }
})
vi.mock('../../src/dates/picker-panel.ts', async (importOriginal) => {
  panel.loads++
  await panel.gate
  return importOriginal()
})

let unmount: (() => void) | undefined
afterEach(() => {
  unmount?.()
  unmount = undefined
  reset()
})

const button = () => $<HTMLButtonElement>('.weft-date-picker__button')

describe('DatePicker, its popover still loading', () => {
  it('loads it when the pointer comes, opens once it is in, and moves focus to the chosen date', async () => {
    const m = mount(DatePickerCases, { initial: '2026-11-21' })
    unmount = m.unmount
    await settle()
    expect(panel.loads).toBe(0)
    $('.weft-date-picker').dispatchEvent(new PointerEvent('pointerenter'))
    await settle()
    expect(panel.loads).toBe(1)

    button().focus()
    button().click()
    await settle()
    // Open, as the button says, but nothing is drawn yet, and focus stays.
    expect(m.exposed.open).toBe(true)
    expect(button().getAttribute('aria-expanded')).toBe('true')
    expect(overlays()?.querySelector('.weft-popover') ?? null).toBeNull()
    expect(document.activeElement).toBe(button())

    panel.release()
    await settle()
    await settle()
    const el = $('.weft-popover', overlays()!)
    expect(el.getAttribute('role')).toBe('dialog')
    expect(button().getAttribute('aria-controls')).toBe(el.id)
    expect(focusedIso()).toBe('2026-11-21')
    expect(panel.loads).toBe(1)
  })
})
