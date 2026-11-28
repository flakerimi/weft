// Dates and datetimes as Warp sends them, and as a browser shows them.
//
// On the wire a date is midnight in UTC (2026-03-29T00:00:00Z): a date has no
// zone, so it is shown and entered as written, in every zone. A datetime is
// an instant in UTC (2026-03-29T01:30:00Z), shown and entered in the zone the
// browser is in. Go's zero time, 0001-01-01T00:00:00Z, is read as empty.
//
// A local time the clocks skip (02:30 in Berlin on 2026-03-29, when they go
// from 02:00 to 03:00) is moved forward by the hour skipped, as the browser's
// Date moves it: 02:30 is sent as 01:30Z, which is 03:30 there. A local time
// the clocks repeat, when they go back, is the earlier of the two.

/** Which of the two a time field is. */
export type WireTimeFormat = 'date' | 'datetime'

/** Go's zero time, which Warp answers for a time that was never set. */
const ZERO = '0001-01-01T00:00:00Z'
const DAY = /^(\d{4})-(\d{2})-(\d{2})$/
const TIME = /^(\d{2}):(\d{2})(?::(\d{2}))?$/

const pad = (n: number, width = 2) => String(n).padStart(width, '0')
const empty = (value: string | null | undefined): value is null | undefined | '' => !value || value === ZERO

/** A wire date (`2026-03-29T00:00:00Z`) as a DatePicker binds it (`2026-03-29`); null for nothing or Go's zero time. */
export function wireToDate(value: string | null | undefined): string | null {
  if (empty(value)) return null
  const day = /^\d{4}-\d{2}-\d{2}(?=T)/.exec(value)
  return day ? day[0] : null
}

/** A DatePicker's date (`2026-03-29`) as the wire has it (`2026-03-29T00:00:00Z`); null for none. */
export function dateToWire(date: string | null): string | null {
  return date && DAY.test(date) ? `${date}T00:00:00Z` : null
}

/** A wire datetime as a date and a time of day (`HH:mm`) in the browser's zone; null for nothing, Go's zero time or what is not a time. */
export function wireToLocal(value: string | null | undefined): { date: string; time: string } | null {
  if (empty(value)) return null
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return null
  return {
    date: `${pad(d.getFullYear(), 4)}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`,
    time: `${pad(d.getHours())}:${pad(d.getMinutes())}`,
  }
}

/** A date and a time of day (`HH:mm` or `HH:mm:ss`) in the browser's zone as the instant in UTC, `YYYY-MM-DDTHH:MM:SSZ`; '' when either is not one. */
export function localToWire(date: string, time: string): string {
  const day = DAY.exec(date)
  const at = TIME.exec(time)
  if (!day || !at) return ''
  const d = new Date(2000, 0, 1)
  // setFullYear, since the constructor reads a year below 100 as 19xx.
  d.setFullYear(Number(day[1]), Number(day[2]) - 1, Number(day[3]))
  d.setHours(Number(at[1]), Number(at[2]), Number(at[3] ?? 0), 0)
  if (Number.isNaN(d.getTime())) return ''
  return `${pad(d.getUTCFullYear(), 4)}-${pad(d.getUTCMonth() + 1)}-${pad(d.getUTCDate())}T${pad(d.getUTCHours())}:${pad(d.getUTCMinutes())}:${pad(d.getUTCSeconds())}Z`
}

/**
 * A wire date or datetime as text, for a table column's `format`: a date as
 * written (`Mar 29, 2026`), a datetime in the browser's zone
 * (`Mar 29, 2026, 1:30 AM`), in `locale` or the browser's; '' for nothing.
 */
export function wireTimeText(value: unknown, format: WireTimeFormat, locale?: string): string {
  if (typeof value !== 'string') return ''
  if (format === 'date') {
    const date = wireToDate(value)
    if (!date) return ''
    const [y, m, d] = date.split('-').map(Number) as [number, number, number]
    const at = new Date(0)
    at.setUTCFullYear(y, m - 1, d)
    return new Intl.DateTimeFormat(locale, { dateStyle: 'medium', timeZone: 'UTC' }).format(at)
  }
  if (empty(value)) return ''
  const at = new Date(value)
  if (Number.isNaN(at.getTime())) return ''
  return new Intl.DateTimeFormat(locale, { dateStyle: 'medium', timeStyle: 'short' }).format(at)
}
