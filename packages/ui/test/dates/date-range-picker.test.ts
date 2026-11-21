import { afterEach, describe, expect, it } from 'vitest'
import DateRangePickerCases from '../fixtures/dates/DateRangePickerCases.vue'
import { mount } from '../mount.ts'
import { loadStyles } from '../styles.ts'
import { px, type } from '../forms/helpers.ts'
import { press as pressKey } from '../keyboard.ts'
import { $, $$, active, key, locked, overlays, reset, settle } from '../overlays/helpers.ts'
import { day, focusedIso, heading, wrapFunctions } from './helpers.ts'

let unmount: (() => void) | undefined
afterEach(() => {
  unmount?.()
  unmount = undefined
  reset()
})
type Exposed = { value: unknown; open: boolean; input: string; changes: unknown[]; submitted: string[] }
async function cases(props: Record<string, unknown> = {}) {
  const m = mount(DateRangePickerCases, { initial: { start: '2026-11-21', end: '2026-11-25' }, ...wrapFunctions(props) })
  unmount = m.unmount
  await settle()
  return m.exposed as unknown as Exposed
}
const fields = () => $$<HTMLInputElement>('.weft-date-picker input.weft-input__field')
const button = () => $<HTMLButtonElement>('.weft-date-picker__button')
const panel = () => $('.weft-popover', overlays() ?? document)
const nameOf = (el: Element) =>
  el.getAttribute('aria-labelledby')!.split(' ').map((id) => document.getElementById(id)!.textContent!.replace('*', '').trim()).join(' ')
async function typeIn(el: HTMLInputElement, text: string) {
  el.focus()
  type(el, text)
  el.dispatchEvent(new Event('change', { bubbles: true }))
  await settle()
}
async function openWithButton() {
  button().focus()
  button().click()
  await settle()
}

describe('DateRangePicker', () => {
  describe('on pointer', () => {
    it('is two text fields, start and end, in the locale, each named, and a button for the calendar', async () => {
      await cases({ locale: 'en-GB' })
      const [start, end] = fields()
      expect(start!.value).toBe('21/11/2026')
      expect(end!.value).toBe('25/11/2026')
      expect($('label').getAttribute('for')).toBe(start!.id)
      expect(nameOf(start!)).toBe('Stay Start date')
      expect(nameOf(end!)).toBe('Stay End date')
      expect(start!.getAttribute('aria-describedby')).toBe($('.weft-form-field__description').id)
      expect(end!.getAttribute('aria-describedby')).toBe($('.weft-form-field__description').id)
      expect(button().getAttribute('aria-haspopup')).toBe('dialog')
      // (ICU puts thin spaces round the dash, or none, by version.)
      expect(button().getAttribute('aria-label')!.replace(/\s*–\s*/g, '–')).toBe('Choose dates, 21–25 November 2026')
    })

    it('reads typed dates in the locale, refusing an end before the start', async () => {
      const e = await cases({ locale: 'en-GB', initial: null })
      const [start, end] = fields()
      await typeIn(start!, '1/12/2026')
      expect(e.value).toEqual({ start: '2026-12-01', end: null })
      await typeIn(end!, '5 Dec 2026')
      expect(e.value).toEqual({ start: '2026-12-01', end: '2026-12-05' })
      expect(end!.value).toBe('05/12/2026')
      await typeIn(end!, '30/11/2026')
      expect(end!.getAttribute('aria-invalid')).toBe('true')
      expect(e.value).toEqual({ start: '2026-12-01', end: null })
      await typeIn(start!, '')
      await typeIn(end!, '')
      expect(e.value).toBeNull()
    })

    it('opens two months side by side on the start, and chooses a range in two presses', async () => {
      const e = await cases()
      await openWithButton()
      expect($$('[role="grid"]', panel())).toHaveLength(2)
      expect(heading()).toBe('November – December 2026')
      expect(focusedIso()).toBe('2026-11-21')
      expect(day('2026-11-23').dataset.range).toBe('middle')
      day('2026-11-28').click()
      await settle()
      expect(e.open).toBe(true)
      day('2026-12-03').click()
      await settle()
      expect(e.value).toEqual({ start: '2026-11-28', end: '2026-12-03' })
      expect(e.open).toBe(false)
      expect(active()).toBe(button())
      expect(fields().map((f) => f.value)).toEqual(['11/28/2026', '12/03/2026'])
      expect(e.changes).toHaveLength(1)
    })

    it('submits both ends under name[start] and name[end]', async () => {
      const e = await cases()
      $<HTMLButtonElement>('.submit').click()
      await settle()
      expect(e.submitted).toEqual(['2026-11-21', '2026-11-25'])
    })
  })

  describe('on touch', () => {
    it('is one button showing the range, opening a sheet of one month with 44 px dates', async () => {
      const unstyle = loadStyles()
      try {
        const e = await cases({ input: 'touch', locale: 'en-GB' })
        const t = $<HTMLButtonElement>('.weft-date-picker__trigger')
        expect(fields()).toHaveLength(0)
        expect(nameOf(t)).toBe('Stay 21/11/2026 – 25/11/2026')
        t.focus()
        t.click()
        await settle()
        expect(panel().dataset.form).toBe('sheet')
        expect(locked()).toBe(true)
        expect($$('[role="grid"]', panel())).toHaveLength(1)
        expect(px(getComputedStyle(day('2026-11-21')).minBlockSize)).toBe(44)
        day('2026-11-02').click()
        await settle()
        day('2026-11-04').click()
        await settle()
        expect(e.value).toEqual({ start: '2026-11-02', end: '2026-11-04' })
        expect(active()).toBe(t)
      } finally {
        unstyle()
      }
    })
  })

  it('keeps a half-chosen range and the focused date when the input switches while open (Review Focus 2)', async () => {
    const m = await cases()
    await openWithButton()
    day('2026-11-25').focus()
    day('2026-11-25').click()
    await settle()
    key('ArrowDown')
    await settle()
    key('ArrowDown')
    await settle()
    expect(focusedIso()).toBe('2026-12-09')
    m.input = 'touch'
    await settle()
    await settle()
    expect(panel().dataset.form).toBe('sheet')
    expect($$('[role="grid"]', panel())).toHaveLength(1)
    expect(heading()).toBe('December 2026')
    expect(focusedIso()).toBe('2026-12-09')
    expect(day('2026-12-01').dataset.range).toBe('middle')
    pressKey(active() as HTMLElement, 'Enter')
    await settle()
    expect(m.value).toEqual({ start: '2026-11-25', end: '2026-12-09' })
    expect(m.open).toBe(false)
  })
})
