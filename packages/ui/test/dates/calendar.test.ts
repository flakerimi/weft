import { CalendarDate, DateFormatter } from '@internationalized/date'
import { afterEach, describe, expect, it, vi } from 'vitest'
import CalendarCases from '../fixtures/dates/CalendarCases.vue'
import { mount } from '../mount.ts'
import { loadStyles, ruleFor } from '../styles.ts'
import { px } from '../forms/helpers.ts'
import { press as pressKey } from '../keyboard.ts'
import { $, $$, active, key, reset, settle } from '../overlays/helpers.ts'
import { day, focusedIso, heading, swipe, tabStop, wrapFunctions } from './helpers.ts'

let unmount: (() => void) | undefined
afterEach(() => {
  unmount?.()
  unmount = undefined
  reset()
})
type Exposed = { value: unknown; focusedDate: unknown; input: string; changes: number; months: number | undefined }
async function cases(props: Record<string, unknown> = {}) {
  const m = mount(CalendarCases, { initial: '2026-11-21', ...wrapFunctions(props) })
  unmount = m.unmount
  await settle()
  return m.exposed as unknown as Exposed
}
const grid = () => $('[role="grid"]')
/** Focus the date's button and press a key there, as the grid hears it. */
async function from(iso: string, ...keys: (string | [string, KeyboardEventInit])[]) {
  day(iso).focus()
  for (const k of keys) {
    if (Array.isArray(k)) key(k[0], k[1])
    else key(k)
    await settle()
  }
}

describe('Calendar', () => {
  it('is an APG date grid: a table of date buttons in gridcells, labelled by its live month heading', async () => {
    await cases({ locale: 'en-US' })
    const g = grid()
    expect(g.tagName).toBe('TABLE')
    expect(heading()).toBe('November 2026')
    expect(g.getAttribute('aria-labelledby')).toBe($('.weft-calendar__heading').id)
    expect($('.weft-calendar__heading').getAttribute('aria-live')).toBe('polite')
    const heads = $$('thead th', g)
    expect(heads.map((h) => h.textContent!.trim())).toEqual(['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'])
    expect(heads.map((h) => h.getAttribute('abbr'))[0]).toBe('Sunday')
    expect(heads.every((h) => h.getAttribute('scope') === 'col')).toBe(true)
    expect($$('tbody tr', g)).toHaveLength(6)
    const chosen = day('2026-11-21')
    expect(chosen.closest('td')!.getAttribute('role')).toBe('gridcell')
    expect(chosen.closest('td')!.getAttribute('aria-selected')).toBe('true')
    expect(day('2026-11-20').closest('td')!.getAttribute('aria-selected')).toBe('false')
    expect(chosen.getAttribute('aria-label')).toBe('Saturday, November 21, 2026')
    expect(chosen.textContent!.trim()).toBe('21')
    // One tab stop: the chosen date.
    expect(tabStop()).toEqual([chosen])
    expect(chosen.hasAttribute('data-weft-autofocus')).toBe(true)
    // Days of the next and previous month fill the weeks, drawn as outside.
    expect($('.weft-calendar__day[data-date="2026-12-05"]').dataset.outside).toBe('true')
    expect($('[aria-label="Previous month"]').tagName).toBe('BUTTON')
    expect($('[aria-label="Next month"]')).not.toBeNull()
    expect($('[aria-label="Previous year"]')).not.toBeNull()
    expect($('[aria-label="Next year"]')).not.toBeNull()
  })

  it('speaks the locale: names, digits, the first day of the week, from its prop or its WeftProvider', async () => {
    await cases({ locale: 'de-DE' })
    expect(heading()).toBe('November 2026')
    expect($$('thead th').map((h) => h.textContent!.trim())[0]).toBe('Mo')
    expect(day('2026-11-21').getAttribute('aria-label')).toBe('Samstag, 21. November 2026')
    unmount!()
    await cases({ providerLocale: 'ar-EG' })
    expect(heading()).toBe('نوفمبر ٢٠٢٦')
    expect(day('2026-11-21').textContent!.trim()).toBe('٢١')
    expect($$('thead th')[0]!.getAttribute('abbr')).toBe('السبت')
    unmount!()
    // en-US by default, never the browser's.
    await cases()
    expect($$('thead th')[0]!.getAttribute('abbr')).toBe('Sunday')
  })

  it('starts the week where week-starts-on says, overriding the locale', async () => {
    await cases({ locale: 'en-US', weekStartsOn: 1 })
    expect($$('thead th')[0]!.getAttribute('abbr')).toBe('Monday')
    expect($('tbody td button').dataset.date).toBe('2026-10-26')
  })

  describe('keyboard', () => {
    it('moves a day with Left and Right and a week with Up and Down, focus following', async () => {
      await cases()
      await from('2026-11-21', 'ArrowRight')
      expect(focusedIso()).toBe('2026-11-22')
      key('ArrowLeft')
      key('ArrowLeft')
      await settle()
      expect(focusedIso()).toBe('2026-11-20')
      key('ArrowDown')
      await settle()
      expect(focusedIso()).toBe('2026-11-27')
      key('ArrowUp')
      key('ArrowUp')
      await settle()
      expect(focusedIso()).toBe('2026-11-13')
      expect(tabStop()).toEqual([day('2026-11-13')])
    })

    it('crosses into the next and previous month, the heading following', async () => {
      await cases()
      await from('2026-11-30', 'ArrowRight')
      expect(heading()).toBe('December 2026')
      expect(focusedIso()).toBe('2026-12-01')
      key('ArrowUp')
      await settle()
      expect(heading()).toBe('November 2026')
      expect(focusedIso()).toBe('2026-11-24')
    })

    it('goes to the ends of the week with Home and End, as the locale counts it', async () => {
      await cases({ locale: 'en-GB' })
      await from('2026-11-19', 'Home')
      expect(focusedIso()).toBe('2026-11-16')
      key('End')
      await settle()
      expect(focusedIso()).toBe('2026-11-22')
    })

    it('moves a month with Page Up and Page Down, keeping the day or the month’s last', async () => {
      await cases({ initial: '2026-01-31' })
      await from('2026-01-31', 'PageDown')
      expect(heading()).toBe('February 2026')
      expect(focusedIso()).toBe('2026-02-28')
      key('PageUp')
      await settle()
      expect(focusedIso()).toBe('2026-01-28')
      expect(heading()).toBe('January 2026')
    })

    it('moves a year with Shift+Page Up and Shift+Page Down', async () => {
      await cases({ initial: '2028-02-29' })
      await from('2028-02-29', ['PageDown', { shiftKey: true }])
      expect(focusedIso()).toBe('2029-02-28')
      expect(heading()).toBe('February 2029')
      key('PageUp', { shiftKey: true })
      key('PageUp', { shiftKey: true })
      await settle()
      expect(focusedIso()).toBe('2027-02-28')
    })

    it('prevents the default of the keys it handles, and leaves Tab alone', async () => {
      await cases()
      day('2026-11-21').focus()
      for (const k of ['ArrowLeft', 'ArrowRight', 'ArrowUp', 'ArrowDown', 'Home', 'End', 'PageUp', 'PageDown']) {
        expect(key(k).defaultPrevented, k).toBe(true)
        await settle()
      }
      expect(key('Tab').defaultPrevented).toBe(false)
    })

    it('mirrors Left and Right right to left', async () => {
      await cases({ dir: 'rtl' })
      await from('2026-11-21', 'ArrowRight')
      expect(focusedIso()).toBe('2026-11-20')
      key('ArrowLeft')
      key('ArrowLeft')
      await settle()
      expect(focusedIso()).toBe('2026-11-22')
    })

    it('chooses with Enter and Space (the native button press)', async () => {
      const e = await cases()
      await from('2026-11-21', 'ArrowRight')
      pressKey(active() as HTMLElement, 'Enter')
      await settle()
      expect(e.value).toBe('2026-11-22')
      key('ArrowRight')
      await settle()
      pressKey(active() as HTMLElement, ' ')
      await settle()
      expect(e.value).toBe('2026-11-23')
      expect(day('2026-11-23').closest('td')!.getAttribute('aria-selected')).toBe('true')
    })
  })

  it('chooses with a press, an outside day too, which shows its month', async () => {
    const e = await cases()
    day('2026-11-03').click()
    await settle()
    expect(e.value).toBe('2026-11-03')
    $<HTMLButtonElement>('.weft-calendar__day[data-date="2026-12-02"]').click()
    await settle()
    expect(e.value).toBe('2026-12-02')
    expect(heading()).toBe('December 2026')
    // Choosing the chosen date again keeps it (a single date is chosen as a radio is) and reports nothing.
    const before = e.changes
    day('2026-12-02').click()
    await settle()
    expect(e.value).toBe('2026-12-02')
    expect(e.changes).toBe(before)
  })

  it('moves focus to an outside day pressed, in the month it shows', async () => {
    await cases()
    const dec = $<HTMLButtonElement>('.weft-calendar__day[data-date="2026-12-02"]')
    dec.focus()
    dec.click()
    await settle()
    expect(heading()).toBe('December 2026')
    expect(active()).toBe(day('2026-12-02'))
    expect((active() as HTMLElement).dataset.outside).toBeUndefined()
  })

  it('moves focus with an outside day pressed even when the press focuses nothing (Safari, Firefox on macOS)', async () => {
    await cases()
    ;(document.activeElement as HTMLElement | null)?.blur()
    expect(active()).toBe(document.body)
    const dec = $<HTMLButtonElement>('.weft-calendar__day[data-date="2026-12-02"]')
    dec.dispatchEvent(new PointerEvent('pointerdown', { bubbles: true, cancelable: true, pointerType: 'mouse', pointerId: 1 }))
    dec.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true, detail: 1 }))
    await settle()
    expect(heading()).toBe('December 2026')
    expect(active()).toBe(day('2026-12-02'))
  })

  it('leaves focus where it is when a script chooses an outside day', async () => {
    await cases()
    const outsideButton = $<HTMLButtonElement>('.after')
    outsideButton.focus()
    $<HTMLButtonElement>('.weft-calendar__day[data-date="2026-12-02"]').click()
    await settle()
    expect(heading()).toBe('December 2026')
    expect(active()).toBe(outsideButton)
  })

  it('binds CalendarDate objects with value-format="object"', async () => {
    const e = await cases({ valueFormat: 'object', initial: new CalendarDate(2026, 11, 21) })
    expect(tabStop()).toEqual([day('2026-11-21')])
    day('2026-11-25').click()
    await settle()
    expect(e.value).toBeInstanceOf(CalendarDate)
    expect(String(e.value)).toBe('2026-11-25')
  })

  it('starts on today’s month with no value, today marked aria-current="date"', async () => {
    await cases({ initial: null })
    const now = new Date()
    const iso = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}-${String(now.getDate()).padStart(2, '0')}`
    expect(day(iso).getAttribute('aria-current')).toBe('date')
    expect(tabStop()).toEqual([day(iso)])
    expect($$('[aria-current="date"]')).toHaveLength(1)
  })

  it('follows and sets the focused date with v-model:focused-date', async () => {
    const e = await cases({ focused: '2027-03-10' })
    expect(heading()).toBe('March 2027')
    expect(tabStop()[0]!.dataset.date).toBe('2027-03-10')
    await from('2027-03-10', 'ArrowDown')
    expect(e.focusedDate).toBe('2027-03-17')
  })

  describe('min, max and disabled dates', () => {
    it('keeps the keyboard within min and max and refuses the days outside them', async () => {
      const e = await cases({ min: '2026-11-10', max: '2026-11-25' })
      await from('2026-11-21', 'PageDown')
      expect(focusedIso()).toBe('2026-11-25')
      key('PageUp')
      await settle()
      expect(focusedIso()).toBe('2026-11-10')
      key('ArrowLeft')
      await settle()
      expect(focusedIso()).toBe('2026-11-10')
      const early = day('2026-11-09')
      expect(early.getAttribute('aria-disabled')).toBe('true')
      expect(early.closest('td')!.getAttribute('aria-disabled')).toBe('true')
      early.click()
      await settle()
      expect(e.value).toBe('2026-11-21')
      expect($<HTMLButtonElement>('[aria-label="Previous month"]').disabled).toBe(true)
      expect($<HTMLButtonElement>('[aria-label="Next month"]').disabled).toBe(true)
    })

    it('lets the keyboard reach disabled dates, which cannot be chosen, and marks unavailable ones', async () => {
      const weekend = (d: CalendarDate) => d.toDate('UTC').getUTCDay() % 6 === 0
      const e = await cases({ isDateDisabled: weekend, isDateUnavailable: (d: CalendarDate) => d.day === 24, initial: '2026-11-20' })
      await from('2026-11-20', 'ArrowRight')
      expect(focusedIso()).toBe('2026-11-21')
      expect(day('2026-11-21').getAttribute('aria-disabled')).toBe('true')
      pressKey(active() as HTMLElement, 'Enter')
      await settle()
      expect(e.value).toBe('2026-11-20')
      expect(day('2026-11-24').dataset.unavailable).toBe('true')
      expect(day('2026-11-24').getAttribute('aria-disabled')).toBe('true')
      day('2026-11-24').click()
      await settle()
      expect(e.value).toBe('2026-11-20')
    })
  })

  it('chooses several with multiple, a press toggling, the dates sorted', async () => {
    const e = await cases({ multiple: true, initial: ['2026-11-21'] })
    expect(grid().getAttribute('aria-multiselectable')).toBe('true')
    day('2026-11-03').click()
    await settle()
    day('2026-11-25').click()
    await settle()
    expect(e.value).toEqual(['2026-11-03', '2026-11-21', '2026-11-25'])
    day('2026-11-21').click()
    await settle()
    expect(e.value).toEqual(['2026-11-03', '2026-11-25'])
    expect(day('2026-11-03').closest('td')!.getAttribute('aria-selected')).toBe('true')
  })

  describe('range', () => {
    it('chooses a start then an end, showing the range on the way, and binds it once complete', async () => {
      const e = await cases({ range: true, initial: null, focused: '2026-11-10' })
      day('2026-11-10').click()
      await settle()
      expect(e.changes).toBe(0)
      expect(day('2026-11-10').dataset.range).toBe('start end')
      // The pointer over a later day previews the range.
      day('2026-11-13').dispatchEvent(new PointerEvent('pointerenter'))
      await settle()
      expect(day('2026-11-12').dataset.range).toBe('middle')
      expect(day('2026-11-12').dataset.preview).toBe('true')
      day('2026-11-13').click()
      await settle()
      expect(e.value).toEqual({ start: '2026-11-10', end: '2026-11-13' })
      expect(['2026-11-10', '2026-11-11', '2026-11-12', '2026-11-13'].map((d) => day(d).closest('td')!.getAttribute('aria-selected'))).toEqual(['true', 'true', 'true', 'true'])
      expect(day('2026-11-13').dataset.range).toBe('end')
      expect(day('2026-11-12').dataset.preview).toBeUndefined()
    })

    it('takes the ends in either order, and previews with the keyboard too', async () => {
      const e = await cases({ range: true, initial: null, focused: '2026-11-20' })
      day('2026-11-20').click()
      await settle()
      await from('2026-11-20', 'ArrowLeft', 'ArrowLeft')
      expect(day('2026-11-19').dataset.range).toBe('middle')
      pressKey(active() as HTMLElement, 'Enter')
      await settle()
      expect(e.value).toEqual({ start: '2026-11-18', end: '2026-11-20' })
    })

    it('previews without writing every date’s name again', async () => {
      await cases({ range: true, initial: null, focused: '2026-11-10' })
      day('2026-11-10').click()
      await settle()
      const formats = vi.spyOn(DateFormatter.prototype, 'format')
      day('2026-11-14').dispatchEvent(new PointerEvent('pointerenter'))
      await settle()
      expect(day('2026-11-12').dataset.range).toBe('middle')
      expect(formats).not.toHaveBeenCalled()
      formats.mockRestore()
    })

    it('does not span a disabled date: a press past one starts again from it', async () => {
      const e = await cases({ range: true, initial: null, focused: '2026-11-10', isDateDisabled: (d: CalendarDate) => d.day === 12 })
      day('2026-11-10').click()
      await settle()
      day('2026-11-14').click()
      await settle()
      expect(e.changes).toBe(0)
      expect(day('2026-11-14').dataset.range).toBe('start end')
      day('2026-11-16').click()
      await settle()
      expect(e.value).toEqual({ start: '2026-11-14', end: '2026-11-16' })
    })
  })

  it('pages with its month and year buttons, the tab stop moving and focus staying on the button', async () => {
    await cases()
    const next = $<HTMLButtonElement>('[aria-label="Next month"]')
    next.focus()
    next.click()
    await settle()
    expect(heading()).toBe('December 2026')
    expect(tabStop()[0]!.dataset.date).toBe('2026-12-21')
    expect(active()).toBe(next)
    $<HTMLButtonElement>('[aria-label="Previous year"]').click()
    await settle()
    expect(heading()).toBe('December 2025')
  })

  it('switches from its heading to a month view and a year view, and back to the days', async () => {
    await cases({ initial: '2026-11-21' })
    const view = $<HTMLButtonElement>('.weft-calendar__view')
    view.focus()
    view.click()
    await settle()
    expect(heading()).toBe('2026')
    expect(active()).toBe($('.weft-calendar__view'))
    const months = $$<HTMLButtonElement>('.weft-calendar__cell')
    expect(months.map((m) => m.textContent!.trim())).toHaveLength(12)
    expect(months.find((m) => m.tabIndex === 0)!.textContent!.trim()).toBe('Nov')
    months.find((m) => m.tabIndex === 0)!.focus()
    key('ArrowDown')
    await settle()
    // Three months a row: November, then the next year's February.
    expect((active() as HTMLElement).textContent!.trim()).toBe('Feb')
    expect(heading()).toBe('2027')
    pressKey(active() as HTMLElement, 'Enter')
    await settle()
    expect(heading()).toBe('February 2027')
    expect(focusedIso()).toBe('2027-02-21')
    $<HTMLButtonElement>('.weft-calendar__view').click()
    await settle()
    $<HTMLButtonElement>('.weft-calendar__view').click()
    await settle()
    const years = $$<HTMLButtonElement>('.weft-calendar__cell')
    expect(years).toHaveLength(12)
    expect(years.find((y) => y.tabIndex === 0)!.textContent!.trim()).toBe('2027')
    years.find((y) => y.textContent!.trim() === '2020')!.click()
    await settle()
    expect(heading()).toBe('2020')
    $$<HTMLButtonElement>('.weft-calendar__cell').find((m) => m.textContent!.trim() === 'Jun')!.click()
    await settle()
    expect(heading()).toBe('June 2020')
  })

  describe('type month and year', () => {
    it('with type="month" chooses a month: a grid of months, the heading switching to years and back', async () => {
      const e = await cases({ type: 'month', initial: '2026-03-01' })
      expect(heading()).toBe('2026')
      const cells = () => $$<HTMLButtonElement>('.weft-calendar__cell')
      expect(cells()).toHaveLength(12)
      expect(cells().find((c) => c.tabIndex === 0)!.textContent!.trim()).toBe('Mar')
      expect(cells().find((c) => c.textContent!.trim() === 'Mar')!.closest('td')!.getAttribute('aria-selected')).toBe('true')
      cells().find((c) => c.tabIndex === 0)!.focus()
      key('ArrowRight')
      key('ArrowRight')
      await settle()
      pressKey(active() as HTMLElement, 'Enter')
      await settle()
      expect(e.value).toBe('2026-05-01')
      expect(heading()).toBe('2026')
      expect((active() as HTMLElement).textContent!.trim()).toBe('May')
      // The heading goes to the years, and a year back to its months, never to days.
      $<HTMLButtonElement>('.weft-calendar__view').click()
      await settle()
      expect(heading().replace(/\s*–\s*/, '–')).toBe('2016–2027')
      cells().find((c) => c.textContent!.trim() === '2020')!.click()
      await settle()
      expect(heading()).toBe('2020')
      $<HTMLButtonElement>('.weft-calendar__view').click()
      await settle()
      $<HTMLButtonElement>('.weft-calendar__view').click()
      await settle()
      expect(heading()).toBe('2020')
      expect($('.weft-calendar__day')).toBeNull()
      expect($('[aria-label="Previous year"]')).not.toBeNull()
    })

    it('with type="month" and range chooses from the first day of one month to the last of another', async () => {
      const e = await cases({ type: 'month', range: true, initial: null, focused: '2026-01-15' })
      const cell = (m: string) => $$<HTMLButtonElement>('.weft-calendar__cell').find((c) => c.textContent!.trim() === m)!
      cell('Mar').click()
      await settle()
      expect(cell('Mar').dataset.range).toBe('start end')
      cell('May').dispatchEvent(new PointerEvent('pointerenter'))
      await settle()
      expect(cell('Apr').dataset.range).toBe('middle')
      cell('May').click()
      await settle()
      expect(e.value).toEqual({ start: '2026-03-01', end: '2026-05-31' })
    })

    it('with type="year" chooses a year, or several', async () => {
      const e = await cases({ type: 'year', initial: '2024-01-01' })
      const cell = (y: string) => $$<HTMLButtonElement>('.weft-calendar__cell').find((c) => c.textContent!.trim() === y)!
      expect(cell('2024').closest('td')!.getAttribute('aria-selected')).toBe('true')
      expect($('.weft-calendar__view')).toBeNull()
      cell('2021').click()
      await settle()
      expect(e.value).toBe('2021-01-01')
      unmount!()
      const m = await cases({ type: 'year', multiple: true, initial: ['2020-01-01'] })
      cell('2022').click()
      await settle()
      expect(m.value).toEqual(['2020-01-01', '2022-01-01'])
    })
  })

  it('shows several months side by side, each grid labelled, no outside days', async () => {
    await cases({ numberOfMonths: 2 })
    const grids = $$('[role="grid"]')
    expect(grids).toHaveLength(2)
    expect(heading()).toBe('November – December 2026')
    expect(grids.map((g) => document.getElementById(g.getAttribute('aria-labelledby')!)!.textContent!.replace(/\s+/g, ' ').trim())).toEqual(['November 2026', 'December 2026'])
    expect($('.weft-calendar__day[data-outside]')).toBeNull()
    await from('2026-11-30', 'ArrowRight')
    expect(heading()).toBe('November – December 2026')
    await from('2026-12-31', 'ArrowRight')
    expect(heading()).toBe('December 2026 – January 2027')
  })

  it('keeps focus on the focused date when fewer months are shown (its month is drawn on, not again)', async () => {
    const e = await cases({ numberOfMonths: 2 })
    await from('2026-11-21', 'ArrowDown', 'ArrowDown', 'ArrowDown')
    expect(focusedIso()).toBe('2026-12-12')
    e.months = 1
    await settle()
    await settle()
    expect(heading()).toBe('December 2026')
    expect(focusedIso()).toBe('2026-12-12')
  })

  it('draws as many weeks as the month spans without fixed-weeks, and ISO week numbers with week-numbers', async () => {
    await cases({ initial: '2026-02-10', locale: 'en-GB', fixedWeeks: false, weekNumbers: true })
    expect($$('tbody tr')).toHaveLength(5)
    expect($$('tbody th[scope="row"]').map((t) => t.textContent!.trim())).toEqual(['5', '6', '7', '8', '9'])
  })

  it('is read-only: it moves but chooses nothing; disabled: nothing works', async () => {
    const e = await cases({ readonly: true })
    expect(grid().getAttribute('aria-readonly')).toBe('true')
    await from('2026-11-21', 'ArrowRight')
    expect(focusedIso()).toBe('2026-11-22')
    pressKey(active() as HTMLElement, 'Enter')
    await settle()
    expect(e.value).toBe('2026-11-21')
    unmount!()
    await cases({ disabled: true })
    expect(grid().getAttribute('aria-disabled')).toBe('true')
    expect($$<HTMLButtonElement>('.weft-calendar button').every((b) => b.disabled)).toBe(true)
  })

  it('shows the focus ring on the focused date, and mirrors its arrows right to left', async () => {
    const unstyle = loadStyles()
    try {
      await cases()
      const ring = ruleFor('.weft-calendar__day:focus-visible')!.style
      expect(ring.getPropertyValue('outline-color')).toBe('var(--weft-color-focus, Highlight)')
      expect(ruleFor('[dir="rtl"] .weft-calendar__nav svg')!.style.getPropertyValue('scale')).toBe('-1 1')
    } finally {
      unstyle()
    }
  })

  describe('on touch', () => {
    it('makes every date at least 44 px a side', async () => {
      const unstyle = loadStyles()
      try {
        await cases({ input: 'touch' })
        const s = getComputedStyle(day('2026-11-21'))
        expect(px(s.minBlockSize)).toBe(44)
        expect(px(s.minInlineSize)).toBe(44)
        expect(px(getComputedStyle($('.weft-calendar__nav')).minBlockSize)).toBe(44)
      } finally {
        unstyle()
      }
    })

    // (Under a forced pointer the rule stands back through :not([data-weft-input="pointer"] *),
    // a complex :not() happy-dom does not match; browsers do.)
    it('spreads the dates over the width on a coarse pointer when no input is forced', async () => {
      const unstyle = loadStyles()
      const settings = (window as unknown as { happyDOM: { settings: { navigator: { maxTouchPoints: number } } } }).happyDOM.settings
      settings.navigator.maxTouchPoints = 5
      try {
        await cases()
        expect(getComputedStyle($('.weft-calendar')).display).toBe('flex')
      } finally {
        settings.navigator.maxTouchPoints = 0
        unstyle()
      }
    })

    it('swipes to the next month leftwards and the previous rightwards, choosing nothing', async () => {
      const e = await cases({ input: 'touch' })
      const body = $('.weft-calendar__body')
      await swipe(day('2026-11-18'), -120)
      expect(heading()).toBe('December 2026')
      expect(tabStop()[0]!.dataset.date).toBe('2026-12-21')
      await swipe(body, 120)
      await swipe(body, 120)
      expect(heading()).toBe('October 2026')
      // A short drag is not a swipe.
      await swipe(body, 10)
      expect(heading()).toBe('October 2026')
      expect(e.changes).toBe(0)
    })

    it('mirrors the swipe right to left', async () => {
      await cases({ input: 'touch', dir: 'rtl' })
      await swipe($('.weft-calendar__body'), 120)
      expect(heading()).toBe('December 2026')
    })

    it('does not swipe with a mouse', async () => {
      await cases({ input: 'touch' })
      const body = $('.weft-calendar__body')
      body.dispatchEvent(new PointerEvent('pointerdown', { bubbles: true, pointerType: 'mouse', pointerId: 1, clientX: 200 }))
      body.dispatchEvent(new PointerEvent('pointerup', { bubbles: true, pointerType: 'mouse', pointerId: 1, clientX: 50 }))
      await settle()
      expect(heading()).toBe('November 2026')
    })
  })
})
