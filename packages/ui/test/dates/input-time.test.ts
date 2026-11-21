import { Time } from '@internationalized/date'
import { afterEach, describe, expect, it } from 'vitest'
import InputTimeCases from '../fixtures/dates/InputTimeCases.vue'
import { mount } from '../mount.ts'
import { loadStyles, ruleFor } from '../styles.ts'
import { px } from '../forms/helpers.ts'
import { $, $$, active, key, reset, settle } from '../overlays/helpers.ts'

let unmount: (() => void) | undefined
afterEach(() => {
  unmount?.()
  unmount = undefined
  reset()
})
type Exposed = { value: unknown; input: string; submitted: string | null }
async function cases(props: Record<string, unknown> = {}) {
  const m = mount(InputTimeCases, { initial: '13:05', ...props })
  unmount = m.unmount
  await settle()
  return m.exposed as unknown as Exposed
}
const group = () => $('.weft-input-time')
const segments = () => $$('[role="spinbutton"]')
const seg = (name: string) => segments().find((s) => s.getAttribute('aria-label') === name)!
const texts = () => segments().map((s) => s.textContent!.trim())
async function keys(...names: string[]) {
  for (const k of names) {
    key(k)
    await settle()
  }
}

describe('InputTime', () => {
  describe('on pointer', () => {
    it('is a group of spinbuttons in the locale’s order, labelled by its field', async () => {
      await cases({ locale: 'en-US' })
      const g = group()
      expect(g.getAttribute('role')).toBe('group')
      expect(g.getAttribute('aria-labelledby')).toBe($('label').id)
      expect($('label').hasAttribute('for')).toBe(false)
      expect(g.getAttribute('aria-describedby')).toBe($('.weft-form-field__description').id)
      expect(segments().map((s) => s.getAttribute('aria-label'))).toEqual(['Hour', 'Minute', 'AM/PM'])
      expect(texts()).toEqual(['1', '05', 'PM'])
      expect(g.textContent!.replace(/\s+/g, '')).toBe('1:05PM')
      const hour = seg('Hour')
      expect(hour.tabIndex).toBe(0)
      expect(hour.getAttribute('aria-valuenow')).toBe('1')
      expect(hour.getAttribute('aria-valuemin')).toBe('1')
      expect(hour.getAttribute('aria-valuemax')).toBe('12')
      expect(seg('Minute').getAttribute('aria-valuetext')).toBe('05')
      expect(seg('AM/PM').getAttribute('aria-valuetext')).toBe('PM')
      expect(segments().every((s) => s.tabIndex === 0 && s.getAttribute('inputmode') === 'numeric')).toBe(true)
      unmount!()
      await cases({ locale: 'en-GB' })
      expect(texts()).toEqual(['13', '05'])
      expect(seg('Hour').getAttribute('aria-valuemax')).toBe('23')
      unmount!()
      await cases({ locale: 'en-GB', hourCycle: 12 })
      expect(texts()).toEqual(['1', '05', 'pm'])
    })

    it('steps with the arrow keys, wrapping, and goes to the ends with Home and End', async () => {
      const e = await cases({ locale: 'en-GB', step: { minute: 15 } })
      seg('Hour').focus()
      await keys('ArrowUp')
      expect(e.value).toBe('14:05')
      await keys('End', 'ArrowUp')
      expect(e.value).toBe('00:05')
      expect(key('ArrowDown').defaultPrevented).toBe(true)
      await settle()
      expect(e.value).toBe('23:05')
      seg('Minute').focus()
      await keys('ArrowUp')
      expect(e.value).toBe('23:20')
      await keys('Home')
      expect(e.value).toBe('23:00')
    })

    it('takes typed digits, moving on when a segment is full', async () => {
      const e = await cases({ locale: 'en-GB', initial: null })
      expect(texts()).toEqual(['––', '––'])
      seg('Hour').focus()
      await keys('1', '4')
      expect(active()).toBe(seg('Minute'))
      expect(e.value).toBeNull()
      await keys('3', '0')
      expect(e.value).toBe('14:30')
      seg('Hour').focus()
      // 7 cannot start a two-digit hour: it is 07, and focus moves on.
      await keys('7')
      expect(e.value).toBe('07:30')
      expect(active()).toBe(seg('Minute'))
    })

    it('reads AM and PM from a key in 12-hour time', async () => {
      const e = await cases({ locale: 'en-US', initial: null })
      seg('Hour').focus()
      await keys('9', '4', '5')
      expect(active()).toBe(seg('AM/PM'))
      expect(e.value).toBeNull()
      await keys('p')
      expect(e.value).toBe('21:45')
      await keys('a')
      expect(e.value).toBe('09:45')
      await keys('ArrowUp')
      expect(e.value).toBe('21:45')
    })

    it('clears a segment with Backspace, unbinding the time', async () => {
      const e = await cases({ locale: 'en-GB' })
      seg('Minute').focus()
      await keys('Backspace')
      expect(e.value).toBeNull()
      expect(texts()).toEqual(['13', '––'])
    })

    it('moves between segments with Left and Right, mirrored right to left', async () => {
      await cases({ locale: 'en-GB' })
      seg('Hour').focus()
      await keys('ArrowRight')
      expect(active()).toBe(seg('Minute'))
      await keys('ArrowLeft')
      expect(active()).toBe(seg('Hour'))
      unmount!()
      await cases({ locale: 'en-GB', dir: 'rtl' })
      seg('Hour').focus()
      await keys('ArrowLeft')
      expect(active()).toBe(seg('Minute'))
    })

    it('has seconds with granularity second, and only hours with hour', async () => {
      const e = await cases({ locale: 'en-GB', granularity: 'second', initial: '08:15:30' })
      expect(texts()).toEqual(['08', '15', '30'])
      seg('Second').focus()
      await keys('ArrowUp')
      expect(e.value).toBe('08:15:31')
      unmount!()
      const h = await cases({ locale: 'en-GB', granularity: 'hour', initial: '08:15' })
      expect(texts()).toEqual(['08'])
      seg('Hour').focus()
      await keys('ArrowUp')
      expect(h.value).toBe('09:00')
    })

    it('marks a time outside min and max invalid', async () => {
      await cases({ locale: 'en-GB', min: '09:00', max: '17:00', initial: '18:30' })
      expect(group().dataset.invalid).toBe('true')
      expect(seg('Hour').getAttribute('aria-invalid')).toBe('true')
      seg('Hour').focus()
      await keys('1', '2')
      expect(group().dataset.invalid).toBeUndefined()
    })

    it('binds Time objects with value-format="object", and submits HH:mm under its name', async () => {
      const e = await cases({ locale: 'en-GB', valueFormat: 'object', initial: new Time(13, 5) })
      expect(texts()).toEqual(['13', '05'])
      seg('Hour').focus()
      await keys('ArrowDown')
      expect(e.value).toBeInstanceOf(Time)
      expect((e.value as Time).hour).toBe(12)
      $<HTMLButtonElement>('.submit').click()
      await settle()
      expect(e.submitted).toBe('12:05')
    })

    it('neither focuses nor changes while disabled, and only focuses while read-only', async () => {
      const e = await cases({ locale: 'en-GB', disabled: true })
      expect(segments().every((s) => s.tabIndex === -1 && s.getAttribute('aria-disabled') === 'true')).toBe(true)
      key('ArrowUp', {}, seg('Hour'))
      await settle()
      expect(e.value).toBe('13:05')
      unmount!()
      const r = await cases({ locale: 'en-GB', readonly: true })
      expect(seg('Hour').tabIndex).toBe(0)
      expect(seg('Hour').getAttribute('aria-readonly')).toBe('true')
      seg('Hour').focus()
      await keys('ArrowUp', '4')
      expect(r.value).toBe('13:05')
    })

    it('shows the focus ring on the focused segment', async () => {
      const unstyle = loadStyles()
      try {
        await cases()
        expect(ruleFor('.weft-input-time__segment:focus-visible')!.style.getPropertyValue('outline-color')).toBe('var(--weft-color-focus, Highlight)')
      } finally {
        unstyle()
      }
    })
  })

  describe('range', () => {
    it('is two times, start and end, each a named group of parts, bound as { start, end }', async () => {
      const e = await cases({ locale: 'en-GB', range: true, initial: { start: '09:00', end: '17:30' } })
      const groups = $$('.weft-input-time [role="group"]')
      expect(groups.map((g) => g.getAttribute('aria-label'))).toEqual(['Start time', 'End time'])
      expect(group().getAttribute('aria-labelledby')).toBe($('label').id)
      expect(texts()).toEqual(['09', '00', '17', '30'])
      $$('[role="spinbutton"][aria-label="Hour"]')[1]!.focus()
      await keys('ArrowUp')
      expect(e.value).toEqual({ start: '09:00', end: '18:30' })
      // From the start's last part, Right goes on to the end's first.
      $$('[role="spinbutton"][aria-label="Minute"]')[0]!.focus()
      await keys('ArrowRight')
      expect(active()).toBe($$('[role="spinbutton"][aria-label="Hour"]')[1])
      $<HTMLButtonElement>('.submit').click()
      await settle()
      expect(e.submitted).toBe('09:00–18:30')
    })

    it('marks an end before the start invalid, and is two platform fields on touch', async () => {
      await cases({ locale: 'en-GB', range: true, initial: { start: '18:00', end: '09:00' } })
      expect(group().dataset.invalid).toBe('true')
      unmount!()
      const t = await cases({ input: 'touch', range: true, initial: { start: '09:00', end: '17:30' } })
      const natives = $$<HTMLInputElement>('.weft-input-time input[type="time"]')
      expect(natives.map((n) => n.value)).toEqual(['09:00', '17:30'])
      expect(natives.map((n) => n.getAttribute('aria-label'))).toEqual(['Start time', 'End time'])
      natives[0]!.value = '08:15'
      natives[0]!.dispatchEvent(new Event('input', { bubbles: true }))
      await settle()
      expect(t.value).toEqual({ start: '08:15', end: '17:30' })
    })
  })

  describe('on touch', () => {
    const native = () => $<HTMLInputElement>('.weft-input-time input[type="time"]')

    it('is the platform’s time field, 44 px tall, with min, max and step, labelled by its field', async () => {
      const unstyle = loadStyles()
      try {
        const e = await cases({ input: 'touch', min: '09:00', max: '17:00', step: { minute: 15 } })
        expect(segments()).toHaveLength(0)
        const n = native()
        expect(n.value).toBe('13:05')
        expect(n.min).toBe('09:00')
        expect(n.max).toBe('17:00')
        expect(n.step).toBe('900')
        expect(n.getAttribute('aria-labelledby')).toBe($('label').id)
        expect(n.getAttribute('aria-describedby')).toBe($('.weft-form-field__description').id)
        expect(px(getComputedStyle(group()).blockSize)).toBe(44)
        n.value = '10:30'
        n.dispatchEvent(new Event('input', { bubbles: true }))
        await settle()
        expect(e.value).toBe('10:30')
        n.value = ''
        n.dispatchEvent(new Event('input', { bubbles: true }))
        await settle()
        expect(e.value).toBeNull()
      } finally {
        unstyle()
      }
    })

    it('keeps the time, and focus, when the input switches', async () => {
      const m = await cases({ locale: 'en-GB' })
      seg('Minute').focus()
      m.input = 'touch'
      await settle()
      expect(native().value).toBe('13:05')
      expect(active()).toBe(native())
      m.input = 'pointer'
      await settle()
      expect(texts()).toEqual(['13', '05'])
      expect(active()).toBe(seg('Hour'))
    })
  })
})
