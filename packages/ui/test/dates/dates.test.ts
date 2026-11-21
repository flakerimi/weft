// @vitest-environment node
// The date helpers under Node (no DOM): values, grids, names and typed dates in a locale.
import { CalendarDate, CalendarDateTime, toCalendar, BuddhistCalendar } from '@internationalized/date'
import { describe, expect, it, vi } from 'vitest'
import { clamp, dayOfWeek, formatRange, fromDate, isoWeek, monthWeeks, parseLocalDate, pattern, toDate, weekdays } from '../../src/dates/dates.ts'

const ref = new CalendarDate(2026, 11, 21)
const iso = (d: CalendarDate | null) => d?.toString() ?? null

describe('date values', () => {
  it('reads ISO strings and the library’s date objects, in any calendar, and nothing else', () => {
    expect(iso(toDate('2026-11-21'))).toBe('2026-11-21')
    expect(iso(toDate('2026-11-21T10:00'))).toBe('2026-11-21')
    expect(iso(toDate(new CalendarDateTime(2026, 11, 21, 9)))).toBe('2026-11-21')
    expect(iso(toDate(toCalendar(ref, new BuddhistCalendar())))).toBe('2026-11-21')
    for (const bad of ['', null, undefined, '21/11/2026', '2026-02-30', 42, {}]) expect(toDate(bad), String(bad)).toBeNull()
  })

  it('gives a model ISO strings or the objects themselves', () => {
    expect(fromDate(ref, 'iso')).toBe('2026-11-21')
    expect(fromDate(ref, 'object')).toBe(ref)
    expect(fromDate(null, 'iso')).toBeNull()
  })

  it('clamps within min and max', () => {
    const min = new CalendarDate(2026, 11, 10)
    const max = new CalendarDate(2026, 11, 25)
    expect(iso(clamp(new CalendarDate(2026, 11, 1), min, max))).toBe('2026-11-10')
    expect(iso(clamp(new CalendarDate(2026, 12, 1), min, max))).toBe('2026-11-25')
    expect(iso(clamp(ref, null, null))).toBe('2026-11-21')
  })
})

describe('the month grid', () => {
  it('starts each week on the locale’s first day: Sunday in en-US, Monday in en-GB, Saturday in ar-EG', () => {
    expect(iso(monthWeeks(ref, 'en-US')[0]![0]!)).toBe('2026-11-01')
    expect(iso(monthWeeks(ref, 'en-GB')[0]![0]!)).toBe('2026-10-26')
    expect(iso(monthWeeks(ref, 'ar-EG')[0]![0]!)).toBe('2026-10-31')
    expect(iso(monthWeeks(ref, 'en-US', dayOfWeek(1))[0]![0]!)).toBe('2026-10-26')
  })

  it('has six weeks of seven days when fixed, else as many as the month spans', () => {
    const fixed = monthWeeks(new CalendarDate(2026, 2, 1), 'en-GB')
    expect(fixed).toHaveLength(6)
    expect(fixed.every((w) => w.length === 7)).toBe(true)
    // February 2026 starts on a Sunday: Monday-first it spans five weeks.
    expect(monthWeeks(new CalendarDate(2026, 2, 1), 'en-GB', undefined, false)).toHaveLength(5)
  })

  it('names the weekdays in grid order in the locale', () => {
    expect(weekdays('en-US').map((d) => d.short)).toEqual(['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'])
    expect(weekdays('de-DE').map((d) => d.long)[0]).toBe('Montag')
    expect(weekdays('fr-FR', 'sun')[0]!.long).toBe('dimanche')
  })

  it('numbers weeks as ISO 8601 does', () => {
    expect(isoWeek(new CalendarDate(2026, 1, 1))).toBe(1)
    expect(isoWeek(new CalendarDate(2027, 1, 1))).toBe(53)
    expect(isoWeek(ref)).toBe(47)
  })
})

describe('typed dates', () => {
  it('read in the locale’s order', () => {
    expect(iso(parseLocalDate('11/21/2026', 'en-US', ref))).toBe('2026-11-21')
    expect(iso(parseLocalDate('21/11/2026', 'en-GB', ref))).toBe('2026-11-21')
    expect(iso(parseLocalDate('21.11.2026', 'de-DE', ref))).toBe('2026-11-21')
    expect(iso(parseLocalDate('2026. 11. 21.', 'ko-KR', ref))).toBe('2026-11-21')
    // Any separator, no padding.
    expect(iso(parseLocalDate('3-4-2026', 'en-US', ref))).toBe('2026-03-04')
    expect(iso(parseLocalDate('3 4 2026', 'en-GB', ref))).toBe('2026-04-03')
  })

  it('read ISO 8601 in every locale', () => {
    expect(iso(parseLocalDate('2026-11-21', 'en-US', ref))).toBe('2026-11-21')
    expect(iso(parseLocalDate('2026-11-21', 'de-DE', ref))).toBe('2026-11-21')
  })

  it('read month names and abbreviations in the locale', () => {
    expect(iso(parseLocalDate('21 November 2026', 'en-GB', ref))).toBe('2026-11-21')
    expect(iso(parseLocalDate('Nov 21, 2026', 'en-US', ref))).toBe('2026-11-21')
    expect(iso(parseLocalDate('21. März 2026', 'de-DE', ref))).toBe('2026-03-21')
    expect(iso(parseLocalDate('21 févr. 2026', 'fr-FR', ref))).toBe('2026-02-21')
    // "Ju" is June or July: not enough to say.
    expect(parseLocalDate('21 Ju 2026', 'en-GB', ref)).toBeNull()
  })

  it('read the locale’s digits', () => {
    expect(iso(parseLocalDate('٢١/١١/٢٠٢٦', 'ar-EG', ref))).toBe('2026-11-21')
  })

  it('take the reference year when none is typed, and place two-digit years near it', () => {
    expect(iso(parseLocalDate('21/3', 'en-GB', ref))).toBe('2026-03-21')
    expect(iso(parseLocalDate('21/3/30', 'en-GB', ref))).toBe('2030-03-21')
    expect(iso(parseLocalDate('21/3/70', 'en-GB', ref))).toBe('1970-03-21')
  })

  it('refuse what is not a real date', () => {
    for (const bad of ['', '31/02/2026', '13/13/2026', 'soon', '1/2/3/4', '00/01/2026']) {
      expect(parseLocalDate(bad, 'en-GB', ref), bad).toBeNull()
    }
  })

  it('show people the pattern to type', () => {
    expect(pattern('en-US')).toBe('mm/dd/yyyy')
    expect(pattern('en-GB')).toBe('dd/mm/yyyy')
    expect(pattern('de-DE')).toBe('dd.mm.yyyy')
  })
})

describe('ranges written out', () => {
  // ICU versions differ in the spaces round a range's dash: Node's writes thin
  // spaces where a browser may write plain ones, and the server's text must be
  // the browser's, or the hydration does not match.
  it('use plain spaces, whichever ICU wrote them', () => {
    const text = formatRange('en-US', { month: 'long', year: 'numeric' }, new CalendarDate(2026, 12, 1), new CalendarDate(2027, 1, 1))
    expect(text).toBe('December 2026 – January 2027')
    expect(formatRange('en-US', { year: 'numeric', month: 'long', day: 'numeric' }, new CalendarDate(2026, 12, 20), new CalendarDate(2026, 12, 27))).not.toMatch(/[\u2009\u202f]/)
  })

  it('make only the spaces round the dash plain, no-break ones too, and keep the no-break spaces elsewhere', () => {
    // As an ICU might write it: no-break spaces inside each date, and a different one each side of the dash.
    const written = '1\u00a0déc.\u202f2026\u00a0\u2013\u20093\u00a0janv.\u202f2027'
    const spy = vi.spyOn(Intl.DateTimeFormat.prototype, 'formatRange').mockReturnValue(written)
    try {
      expect(formatRange('fr-FR', { day: 'numeric', month: 'short', year: 'numeric' }, new CalendarDate(2026, 12, 1), new CalendarDate(2027, 1, 3))).toBe(
        '1\u00a0déc.\u202f2026 \u2013 3\u00a0janv.\u202f2027',
      )
      spy.mockReturnValue('20\u202f\u2014\u00a027 декабря')
      expect(formatRange('ru-RU', { day: 'numeric', month: 'long' }, new CalendarDate(2026, 12, 20), new CalendarDate(2026, 12, 27))).toBe('20 \u2014 27 декабря')
    } finally {
      spy.mockRestore()
    }
  })
})
