/**
 * Dates for Calendar, DatePicker and DateRangePicker: every computation goes
 * through `@internationalized/date`, and every name, digit and order through
 * `Intl` in the locale given. Dates are Gregorian `CalendarDate`s (the ISO
 * strings the components bind are Gregorian too); names, digits, the first
 * day of the week and the order of day, month and year follow the locale.
 *
 * - `toDate(value)`: a `CalendarDate` from an ISO `YYYY-MM-DD` string or any
 *   date object of the library (a `CalendarDateTime`, a `ZonedDateTime`, a
 *   date in another calendar), else null.
 * - `fromDate(date, format)`: what a model carries, `'iso'` (a string) or
 *   `'object'` (the `CalendarDate`).
 * - `monthWeeks(month, locale, first?, fixed?)`: the weeks of a month's grid,
 *   each seven dates from the locale's (or `first`'s) first weekday; six
 *   weeks when `fixed`, so the grid does not change height.
 * - `weekdays(locale, first?)`: the weekday names in grid order.
 * - `formatter(locale, options)`: a cached `DateFormatter` in UTC and the
 *   Gregorian calendar; `format(date, locale, options)` formats one date.
 * - `parseLocalDate(text, locale, reference)`: a date typed as people of the
 *   locale write it: its numeric order (21/11/2026 in en-GB, 11/21/2026 in
 *   en-US, 21.11.2026 in de), any separator, the locale's digits, a month
 *   name or its abbreviation in the locale ("21 Nov 2026"), two-digit years
 *   (within 80 years before and 20 after `reference`), a missing year (the
 *   reference's), and ISO 8601 always. Null when it is not a real date.
 * - `pattern(locale)`: the numeric pattern people type, `dd/mm/yyyy`.
 * - `isoWeek(date)`: the ISO 8601 week number.
 *
 * Nothing here touches the DOM, so it runs on the server.
 */
import {
  CalendarDate,
  DateFormatter,
  GregorianCalendar,
  getWeeksInMonth,
  parseDate,
  startOfMonth,
  startOfWeek,
  toCalendar,
  toCalendarDate,
  type DateValue,
  type DayOfWeek,
} from '@internationalized/date'

export { CalendarDate }

/** How a model carries dates: ISO strings (`2026-11-21`) or `CalendarDate` objects. */
export type ValueFormat = 'iso' | 'object'
/**
 * A date as a prop takes it: an ISO string (`2026-11-21`) or a date object of
 * `@internationalized/date` (a `CalendarDate`, a `CalendarDateTime`, a date
 * in another calendar: anything with a calendar, a year, a month and a day).
 */
export type DateInput = string | { readonly calendar: { readonly identifier: string }; readonly year: number; readonly month: number; readonly day: number }
/** A range of dates; either end may be missing while it is being chosen. */
export interface DateRange {
  start: DateInput | null
  end: DateInput | null
}
/** What a Calendar binds: one date, several (`multiple`) or a range (`range`). */
export type CalendarModel = DateInput | DateInput[] | DateRange | null

const DAYS: DayOfWeek[] = ['sun', 'mon', 'tue', 'wed', 'thu', 'fri', 'sat']
/** The library's day name for a weekday number (0 Sunday … 6 Saturday), or undefined. */
export const dayOfWeek = (n: number | undefined): DayOfWeek | undefined => (n === undefined ? undefined : DAYS[((n % 7) + 7) % 7])

const gregorian = new GregorianCalendar()

export function toDate(value: unknown): CalendarDate | null {
  if (value == null || value === '') return null
  if (typeof value === 'string') {
    if (!/^\d{4}-\d{2}-\d{2}/.test(value)) return null
    try {
      return parseDate(value.slice(0, 10))
    } catch {
      return null
    }
  }
  if (typeof value === 'object' && 'calendar' in value && 'year' in value && 'month' in value && 'day' in value) {
    const v = value as DateValue
    return toCalendarDate(v.calendar.identifier === 'gregory' ? v : toCalendar(v, gregorian))
  }
  return null
}

export function fromDate(date: CalendarDate | null, format: ValueFormat): string | CalendarDate | null {
  if (!date) return null
  return format === 'object' ? date : date.toString()
}

/** The same day, the same date: compares by value (null equals null). */
export const sameDay = (a: CalendarDate | null | undefined, b: CalendarDate | null | undefined) =>
  a == null || b == null ? a == b : a.compare(b) === 0

/** `date` held within `min` and `max` (either may be missing). */
export function clamp(date: CalendarDate, min: CalendarDate | null, max: CalendarDate | null): CalendarDate {
  if (min && date.compare(min) < 0) return min
  if (max && date.compare(max) > 0) return max
  return date
}

export function monthWeeks(month: CalendarDate, locale: string, first?: DayOfWeek, fixed = true): CalendarDate[][] {
  const start = startOfWeek(startOfMonth(month), locale, first)
  const count = fixed ? 6 : getWeeksInMonth(month, locale, first)
  return Array.from({ length: count }, (_, w) => Array.from({ length: 7 }, (_, d) => start.add({ days: w * 7 + d })))
}

const cache = new Map<string, DateFormatter>()
export function formatter(locale: string, options: Intl.DateTimeFormatOptions): DateFormatter {
  const key = `${locale}|${JSON.stringify(options)}`
  let f = cache.get(key)
  if (!f) {
    f = new DateFormatter(locale, { calendar: 'gregory', ...options, timeZone: 'UTC' })
    cache.set(key, f)
  }
  return f
}
/** A JavaScript Date at midnight UTC of `date`, for the formatters (which run in UTC). */
export const utc = (date: CalendarDate) => date.toDate('UTC')
export const format = (date: CalendarDate, locale: string, options: Intl.DateTimeFormatOptions) =>
  formatter(locale, options).format(utc(date))
/** A thin, narrow or no-break space next to a range's dash. */
const RANGE_SPACE = /[\u00a0\u2009\u202f](?=[\u2013\u2014-])|(?<=[\u2013\u2014-])[\u00a0\u2009\u202f]/g
/**
 * Two dates as one range, "December 2026 – January 2027". ICU versions differ
 * in the spaces round the dash (Node's newer one writes thin spaces, a
 * browser's may write plain or no-break ones), and the server's text must be
 * the browser's for hydration to match, so a space next to the dash (en,
 * em or hyphen), thin, narrow or no-break, is a plain one here. The
 * no-break spaces inside each date, which keep "1 déc." whole, stay.
 */
export const formatRange = (locale: string, options: Intl.DateTimeFormatOptions, start: CalendarDate, end: CalendarDate) =>
  formatter(locale, options).formatRange(utc(start), utc(end)).replace(RANGE_SPACE, ' ')

export function weekdays(locale: string, first?: DayOfWeek): { short: string; narrow: string; long: string }[] {
  const start = startOfWeek(new CalendarDate(2026, 1, 7), locale, first)
  return Array.from({ length: 7 }, (_, i) => {
    const d = start.add({ days: i })
    return {
      short: format(d, locale, { weekday: 'short' }),
      narrow: format(d, locale, { weekday: 'narrow' }),
      long: format(d, locale, { weekday: 'long' }),
    }
  })
}

/** The order of day, month and year in the locale's numeric dates, and the parts of its pattern. */
function numericParts(locale: string): Intl.DateTimeFormatPart[] {
  return formatter(locale, { year: 'numeric', month: '2-digit', day: '2-digit' }).formatToParts(utc(new CalendarDate(2026, 11, 21)))
}

export function pattern(locale: string): string {
  const letters: Record<string, string> = { day: 'dd', month: 'mm', year: 'yyyy' }
  return numericParts(locale)
    .map((p) => letters[p.type] ?? (p.type === 'literal' ? p.value : ''))
    .join('')
}

/** Text folded for matching: lower case, no accents, no dots. */
const fold = (s: string) => s.normalize('NFD').replace(/\p{M}/gu, '').replace(/\./g, '').toLocaleLowerCase()

/** The locale's digits 0 to 9, for reading typed numbers in Arabic-Indic, Devanagari… */
function digitsOf(locale: string): string[] {
  const nf = new Intl.NumberFormat(locale, { useGrouping: false })
  return Array.from({ length: 10 }, (_, i) => nf.format(i))
}

function monthNames(locale: string): string[][] {
  return Array.from({ length: 12 }, (_, i) => {
    const d = new CalendarDate(2026, i + 1, 1)
    return [format(d, locale, { month: 'long' }), format(d, locale, { month: 'short' }), format(d, locale, { month: 'long', day: 'numeric' }).replace(/[\p{Nd}\s,.]+/gu, ' ').trim()].map(fold)
  })
}

/** The month a word names (1–12): a whole name, or the start of one, at least three letters long. */
function monthOf(word: string, locale: string): number | null {
  const w = fold(word)
  const names = monthNames(locale)
  const exact = names.findIndex((forms) => forms.includes(w))
  if (exact >= 0) return exact + 1
  if (w.length < 3) return null
  const starts = names.map((forms, i) => (forms.some((f) => f.startsWith(w)) ? i + 1 : 0)).filter(Boolean)
  return starts.length === 1 ? starts[0]! : null
}

function fullYear(digits: string, value: number, reference: number): number {
  if (digits.length > 2) return value
  // Within 80 years before the reference year and 20 after.
  let year = Math.floor(reference / 100) * 100 + value
  if (year >= reference + 20) year -= 100
  else if (year < reference - 80) year += 100
  return year
}

function build(year: number, month: number, day: number): CalendarDate | null {
  if (!Number.isInteger(year) || month < 1 || month > 12 || day < 1 || year < 1 || year > 9999) return null
  const date = new CalendarDate(year, month, 1)
  if (day > date.calendar.getDaysInMonth(date)) return null
  return date.set({ day })
}

export function parseLocalDate(text: string, locale: string, reference: CalendarDate): CalendarDate | null {
  const local = digitsOf(locale)
  const ascii = [...text.trim()].map((ch) => {
    const i = local.indexOf(ch)
    return i >= 0 ? String(i) : ch
  }).join('')
  if (!ascii) return null
  const iso = /^(\d{4})-(\d{1,2})-(\d{1,2})$/.exec(ascii)
  if (iso) return build(+iso[1]!, +iso[2]!, +iso[3]!)

  const tokens = ascii.match(/\d+|\p{L}+/gu) ?? []
  const numbers = tokens.filter((t) => /^\d+$/.test(t))
  const words = tokens.filter((t) => !/^\d+$/.test(t))
  const order = numericParts(locale).map((p) => p.type).filter((t): t is 'day' | 'month' | 'year' => t === 'day' || t === 'month' || t === 'year')
  let named: number | null = null
  for (const w of words) {
    const m = monthOf(w, locale)
    if (m !== null) {
      if (named !== null) return null
      named = m
    }
  }
  const slots = named === null ? order : order.filter((t) => t !== 'month')
  // Day and month (and year) as the locale orders them; with one fewer number, no year.
  const fields: Partial<Record<'day' | 'month' | 'year', string>> = {}
  if (numbers.length === slots.length) slots.forEach((t, i) => (fields[t] = numbers[i]))
  else if (numbers.length === slots.length - 1) slots.filter((t) => t !== 'year').forEach((t, i) => (fields[t] = numbers[i]))
  else return null
  const month = named ?? Number(fields.month)
  const day = Number(fields.day)
  const year = fields.year === undefined ? reference.year : fullYear(fields.year, Number(fields.year), reference.year)
  return build(year, month, day)
}

export function isoWeek(date: CalendarDate): number {
  const d = new Date(Date.UTC(date.year, date.month - 1, date.day))
  const day = d.getUTCDay() || 7
  d.setUTCDate(d.getUTCDate() + 4 - day)
  const yearStart = Date.UTC(d.getUTCFullYear(), 0, 1)
  return Math.ceil(((d.getTime() - yearStart) / 86_400_000 + 1) / 7)
}

/** The words a Calendar says, for assistive technology; give some or all to translate them. */
export interface CalendarLabels {
  prevMonth: string
  nextMonth: string
  prevYear: string
  nextYear: string
  prevYears: string
  nextYears: string
  week: string
}
