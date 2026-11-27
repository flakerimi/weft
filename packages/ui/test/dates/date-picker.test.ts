import { CalendarDate } from '@internationalized/date'
import { afterEach, describe, expect, it } from 'vitest'
import DatePickerCases from '../fixtures/dates/DatePickerCases.vue'
import { mount } from '../mount.ts'
import { loadStyles, ruleFor } from '../styles.ts'
import { px, type } from '../forms/helpers.ts'
import { press as pressKey } from '../keyboard.ts'
import { $, active, key, locked, moveFocus, overlays, reset, settle } from '../overlays/helpers.ts'
import { day, focusedIso, heading, swipe, wrapFunctions } from './helpers.ts'

let unmount: (() => void) | undefined
afterEach(() => {
  unmount?.()
  unmount = undefined
  reset()
})
type Exposed = { value: unknown; open: boolean; input: string; changes: unknown[]; submitted: string | null; blurs: number }
async function cases(props: Record<string, unknown> = {}) {
  const m = mount(DatePickerCases, { initial: '2026-11-21', ...wrapFunctions(props) })
  unmount = m.unmount
  await settle()
  return m.exposed as unknown as Exposed
}
const field = () => $<HTMLInputElement>('.weft-date-picker input.weft-input__field')
const button = () => $<HTMLButtonElement>('.weft-date-picker__button')
const panel = () => $('.weft-popover', overlays() ?? document)
/** Type into the field and commit, as leaving it does. */
async function typeIn(text: string) {
  const el = field()
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

describe('DatePicker', () => {
  describe('on pointer', () => {
    it('is a text field showing the date in the locale, with a button that opens the calendar', async () => {
      await cases({ locale: 'en-GB' })
      const f = field()
      expect(f.type).toBe('text')
      expect(f.value).toBe('21/11/2026')
      expect($('label').getAttribute('for')).toBe(f.id)
      expect(f.getAttribute('aria-describedby')).toBe($('.weft-form-field__description').id)
      expect(f.getAttribute('placeholder')).toBe('dd/mm/yyyy')
      const b = button()
      expect(b.type).toBe('button')
      expect(b.getAttribute('aria-haspopup')).toBe('dialog')
      expect(b.getAttribute('aria-expanded')).toBe('false')
      expect(b.getAttribute('aria-label')).toBe('Choose date, 21 November 2026')
      expect(overlays()).toBeNull()
      unmount!()
      await cases({ locale: 'en-US', initial: null, placeholder: 'When?' })
      expect(field().value).toBe('')
      expect(field().getAttribute('placeholder')).toBe('When?')
      expect(button().getAttribute('aria-label')).toBe('Choose date')
    })

    it('gives its input an id when neither a FormField nor a name is set', async () => {
      const DatePicker = (await import('../../src/dates/DatePicker.vue')).default
      const m = mount(DatePicker)
      unmount = m.unmount
      await settle()
      expect(field().id).toMatch(/^date-picker-/)
    })

    it('reads a typed date in the locale on change, writing it back as the locale writes it', async () => {
      const e = await cases({ locale: 'en-GB' })
      await typeIn('3/12/2026')
      expect(e.value).toBe('2026-12-03')
      expect(field().value).toBe('03/12/2026')
      await typeIn('1 Jan 27')
      expect(e.value).toBe('2027-01-01')
      expect(field().value).toBe('01/01/2027')
      expect(field().hasAttribute('aria-invalid')).toBe(false)
      expect(e.changes).toEqual(['2026-12-03', '2027-01-01'])
      unmount!()
      const us = await cases({ locale: 'en-US' })
      await typeIn('3/12/2026')
      expect(us.value).toBe('2026-03-12')
      unmount!()
      const de = await cases({ locale: 'de-DE' })
      await typeIn('24.12.2026')
      expect(de.value).toBe('2026-12-24')
      expect(field().value).toBe('24.12.2026')
    })

    it('clears invalid text when its form resets', async () => {
      await cases({ locale: 'en-GB', initial: null })
      await typeIn('soon')
      expect(field().getAttribute('aria-invalid')).toBe('true')
      field().form!.reset()
      await settle()
      expect(field().value).toBe('')
      expect(field().hasAttribute('aria-invalid')).toBe(false)
      expect($('.weft-date-picker').dataset.invalid).toBeUndefined()
    })

    it('keeps invalid text when the form’s reset is cancelled', async () => {
      await cases({ locale: 'en-GB', initial: null })
      const form = field().form!
      form.addEventListener('reset', (event) => event.preventDefault())
      await typeIn('soon')
      form.reset()
      await settle()
      expect(field().value).toBe('soon')
      expect(field().getAttribute('aria-invalid')).toBe('true')
    })

    it('commits on Enter, before the form hears it', async () => {
      const e = await cases({ locale: 'en-GB' })
      field().focus()
      type(field(), '5/1/2027')
      key('Enter')
      await settle()
      expect(e.value).toBe('2027-01-05')
    })

    it('clears on empty text, and marks text that is no date it allows invalid, binding null', async () => {
      const e = await cases({ locale: 'en-GB', min: '2026-01-01', max: '2026-12-31', isDateDisabled: (d: CalendarDate) => d.day === 13 })
      // A date bound, then text that is none: null bound, the text kept to correct.
      await typeIn('30/2/2026')
      expect(e.value).toBeNull()
      expect(field().value).toBe('30/2/2026')
      expect(field().getAttribute('aria-invalid')).toBe('true')
      await typeIn('')
      expect(e.value).toBeNull()
      expect(field().hasAttribute('aria-invalid')).toBe(false)
      for (const bad of ['31/02/2026', 'soon', '01/01/2030', '13/11/2026']) {
        await typeIn(bad)
        expect(e.value, bad).toBeNull()
        expect(field().value, bad).toBe(bad)
        expect(field().getAttribute('aria-invalid'), bad).toBe('true')
        expect($('.weft-date-picker').dataset.invalid, bad).toBe('true')
      }
      await typeIn('12/11/2026')
      expect(e.value).toBe('2026-11-12')
      expect(field().hasAttribute('aria-invalid')).toBe(false)
    })

    it('opens the calendar from its button: floating, a dialog, focus on the chosen date', async () => {
      const e = await cases()
      await openWithButton()
      expect(e.open).toBe(true)
      expect(button().getAttribute('aria-expanded')).toBe('true')
      expect(button().getAttribute('aria-controls')).toBe(panel().id)
      expect(panel().dataset.form).toBe('floating')
      expect(panel().getAttribute('role')).toBe('dialog')
      expect(panel().getAttribute('aria-label')).toBe('Start date')
      expect(locked()).toBe(false)
      expect(focusedIso()).toBe('2026-11-21')
      expect(heading()).toBe('November 2026')
    })

    it('opens on today with nothing typed or bound, focus on it', async () => {
      await cases({ initial: null })
      await openWithButton()
      const now = new Date()
      const iso = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}-${String(now.getDate()).padStart(2, '0')}`
      expect(focusedIso()).toBe(iso)
      expect($('[aria-busy="true"]', panel())).toBeNull()
    })

    it('opens with Alt+ArrowDown in the field, on the date typed', async () => {
      await cases({ locale: 'en-GB' })
      field().focus()
      type(field(), '2/3/2027')
      key('ArrowDown', { altKey: true })
      await settle()
      expect(panel()).not.toBeNull()
      expect(focusedIso()).toBe('2027-03-02')
    })

    it('chooses with the keyboard: the date is bound, the calendar closes and focus returns to the button', async () => {
      const e = await cases({ locale: 'en-GB' })
      await openWithButton()
      key('ArrowRight')
      await settle()
      key('PageDown')
      await settle()
      pressKey(active() as HTMLElement, 'Enter')
      await settle()
      expect(e.value).toBe('2026-12-22')
      expect(e.open).toBe(false)
      expect(overlays()?.querySelector('.weft-popover') ?? null).toBeNull()
      expect(active()).toBe(button())
      expect(field().value).toBe('22/12/2026')
      expect(e.changes).toEqual(['2026-12-22'])
    })

    it('closes on Escape, focus back on the button, and when focus leaves for the field', async () => {
      const e = await cases()
      await openWithButton()
      key('Escape')
      await settle()
      expect(e.open).toBe(false)
      expect(active()).toBe(button())
      await openWithButton()
      moveFocus(field())
      await settle()
      expect(e.open).toBe(false)
      expect(active()).toBe(field())
      expect(e.value).toBe('2026-11-21')
    })

    it('keeps Tab going round the calendar’s buttons, as the APG dialog does', async () => {
      await cases()
      await openWithButton()
      const date = active()
      const first = $<HTMLButtonElement>('[aria-label="Previous year"]', panel())
      expect(key('Tab').defaultPrevented).toBe(true)
      expect(active()).toBe(first)
      expect(key('Tab', { shiftKey: true }).defaultPrevented).toBe(true)
      expect(active()).toBe(date)
      // Within, Tab is the browser's.
      first.focus()
      expect(key('Tab').defaultPrevented).toBe(false)
    })

    it('closes on a second press of its button, the press keeping focus in the calendar', async () => {
      const e = await cases()
      await openWithButton()
      const down = new MouseEvent('mousedown', { bubbles: true, cancelable: true })
      button().dispatchEvent(down)
      expect(down.defaultPrevented).toBe(true)
      button().click()
      await settle()
      expect(e.open).toBe(false)
    })

    it('reports a blur only when focus leaves the field, its button and its calendar', async () => {
      const e = await cases()
      field().focus()
      button().focus()
      await settle()
      expect(e.blurs, 'to the button').toBe(0)
      await openWithButton()
      expect(e.blurs, 'into the calendar').toBe(0)
      key('Escape')
      await settle()
      expect(active()).toBe(button())
      expect(e.blurs, 'back to the button').toBe(0)
      // Chrome sends a focusout to nowhere when the focused date goes with the
      // calendar, before focus comes back to the button: no blur.
      await openWithButton()
      active()!.dispatchEvent(new FocusEvent('focusout', { bubbles: true, relatedTarget: null }))
      key('Escape')
      await settle()
      expect(active()).toBe(button())
      expect(e.blurs, 'through nowhere').toBe(0)
      // (A real focus move: moveFocus would send a second focusout of its own.)
      $<HTMLButtonElement>('.after').focus()
      await settle()
      expect(e.blurs).toBe(1)
    })

    it('submits the ISO date under the field name', async () => {
      const e = await cases({ locale: 'en-GB' })
      $<HTMLButtonElement>('.submit').click()
      await settle()
      expect(e.submitted).toBe('2026-11-21')
      expect(field().name).toBe('')
    })

    it('binds CalendarDate objects with value-format="object"', async () => {
      const e = await cases({ valueFormat: 'object', initial: new CalendarDate(2026, 11, 21) })
      expect(field().value).toBe('11/21/2026')
      await typeIn('12/25/2026')
      expect(e.value).toBeInstanceOf(CalendarDate)
      expect(String(e.value)).toBe('2026-12-25')
    })

    it('neither opens, edits nor submits while disabled, and only shows while read-only', async () => {
      const d = await cases({ disabled: true })
      $<HTMLButtonElement>('.submit').click()
      await settle()
      expect(d.submitted).toBeNull()
      expect(field().disabled).toBe(true)
      expect(button().disabled).toBe(true)
      button().click()
      key('ArrowDown', { altKey: true }, field())
      await settle()
      expect(overlays()).toBeNull()
      unmount!()
      await cases({ readonly: true })
      expect(field().readOnly).toBe(true)
      expect(button().disabled).toBe(true)
    })

    it('shows the focus ring on its button and field', async () => {
      const unstyle = loadStyles()
      try {
        await cases()
        expect(ruleFor('.weft-date-picker__button:focus-visible')!.style.getPropertyValue('outline')).toBe('var(--weft-focus-width, 2px) solid var(--weft-color-focus, Highlight)')
        expect(ruleFor('.weft-input:focus-within')).toBeDefined()
      } finally {
        unstyle()
      }
    })

    it('opens in the layer of a right-to-left page', async () => {
      await cases({ dir: 'rtl' })
      await openWithButton()
      expect(panel().closest('.weft-overlay')!.getAttribute('dir')).toBe('rtl')
    })
  })

  describe('on touch', () => {
    const trigger = () => $<HTMLButtonElement>('.weft-date-picker__trigger')

    it('is one button showing the date, named by its label and the date', async () => {
      const unstyle = loadStyles()
      try {
        await cases({ input: 'touch', locale: 'en-GB' })
        expect($('.weft-date-picker input.weft-input__field')).toBeNull()
        const t = trigger()
        expect(t.id).toBe($('label').getAttribute('for'))
        expect(t.getAttribute('aria-haspopup')).toBe('dialog')
        const names = t.getAttribute('aria-labelledby')!.split(' ').map((id) => document.getElementById(id)!.textContent!.replace('*', '').trim())
        expect(names).toEqual(['Start date', '21/11/2026'])
        expect(t.getAttribute('aria-describedby')).toBe($('.weft-form-field__description').id)
        expect(px(getComputedStyle($('.weft-date-picker')).blockSize)).toBe(44)
      } finally {
        unstyle()
      }
    })

    it('opens the sheet on today with nothing bound, focus on it', async () => {
      await cases({ input: 'touch', initial: null })
      $<HTMLButtonElement>('.weft-date-picker__trigger').click()
      await settle()
      const now = new Date()
      const iso = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}-${String(now.getDate()).padStart(2, '0')}`
      expect(focusedIso()).toBe(iso)
    })

    it('opens a modal bottom sheet titled by the label, with 44 px dates that swipe between months', async () => {
      const unstyle = loadStyles()
      try {
        const e = await cases({ input: 'touch' })
        trigger().focus()
        trigger().click()
        await settle()
        const sheet = panel()
        expect(sheet.dataset.form).toBe('sheet')
        expect(sheet.getAttribute('aria-modal')).toBe('true')
        expect(sheet.querySelector('h2.weft-popover__title')!.textContent).toBe('Start date')
        expect(locked()).toBe(true)
        expect(focusedIso()).toBe('2026-11-21')
        expect(px(getComputedStyle(day('2026-11-21')).minBlockSize)).toBe(44)
        expect(px(getComputedStyle(day('2026-11-21')).minInlineSize)).toBe(44)
        await swipe($('.weft-calendar__body'), -120)
        expect(heading()).toBe('December 2026')
        day('2026-12-24').click()
        await settle()
        expect(e.value).toBe('2026-12-24')
        expect(e.open).toBe(false)
        expect(active()).toBe(trigger())
        expect(locked()).toBe(false)
      } finally {
        unstyle()
      }
    })
  })

  describe('switching input while open (Review Focus 2)', () => {
    it('keeps the chosen and the focused date from pointer to touch and back, focus on the date', async () => {
      const m = await cases()
      await openWithButton()
      key('ArrowRight')
      await settle()
      key('ArrowDown')
      await settle()
      const calendar = $('.weft-calendar')
      const focusedEl = active()
      expect(focusedIso()).toBe('2026-11-29')
      m.input = 'touch'
      await settle()
      expect(panel().dataset.form).toBe('sheet')
      expect(m.open).toBe(true)
      expect($('.weft-calendar')).toBe(calendar)
      expect(active()).toBe(focusedEl)
      expect(focusedIso()).toBe('2026-11-29')
      expect(day('2026-11-21').closest('td')!.getAttribute('aria-selected')).toBe('true')
      expect(locked()).toBe(true)
      m.input = 'pointer'
      await settle()
      expect(panel().dataset.form).toBe('floating')
      expect(active()).toBe(focusedEl)
      expect(locked()).toBe(false)
      key('Escape')
      await settle()
      expect(m.open).toBe(false)
      expect(active()).toBe(button())
      expect(m.value).toBe('2026-11-21')
    })

    it('moves focus from the text field to the button when the field goes with a switch to touch', async () => {
      const m = await cases()
      field().focus()
      m.input = 'touch'
      await settle()
      expect(active()).toBe($('.weft-date-picker__trigger'))
    })

    it('opened on touch, returns focus to its button after switching to pointer', async () => {
      const m = await cases({ input: 'touch' })
      const t = $<HTMLButtonElement>('.weft-date-picker__trigger')
      t.focus()
      t.click()
      await settle()
      key('ArrowLeft')
      await settle()
      m.input = 'pointer'
      await settle()
      expect(focusedIso()).toBe('2026-11-20')
      pressKey(active() as HTMLElement, 'Enter')
      await settle()
      expect(m.value).toBe('2026-11-20')
      // The same button, now the calendar button beside the field.
      expect(active()).toBe(t)
      expect(t.classList.contains('weft-date-picker__button')).toBe(true)
      expect(field().value).toBe('11/20/2026')
    })
  })
})
