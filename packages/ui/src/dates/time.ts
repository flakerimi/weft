/**
 * Times for InputTime, through `@internationalized/date`'s `Time` and the
 * locale's `Intl` formats.
 *
 * - `toTime(value)`: a `Time` from `HH:mm` or `HH:mm:ss` (the native time
 *   input's format) or any object with an hour and a minute, else null.
 * - `fromTime(time, granularity, format)`: what a model carries: the string
 *   to the granularity (`HH:mm`, `HH:mm:ss`), or the `Time`.
 * - `hourCycleOf(locale, cycle?)`: 12 or 24, as given or as the locale has it.
 * - `segmentsOf(locale, cycle, granularity)`: the segments in the locale's
 *   order, with the text between them: `hour`, `minute`, `second`,
 *   `dayPeriod`, `literal`.
 * - `dayPeriods(locale)`: the locale's words for AM and PM.
 * - `digits(n, locale, width)`: a number in the locale's digits, padded.
 */
import { Time } from '@internationalized/date'

export { Time }
export type Granularity = 'hour' | 'minute' | 'second'
export type TimeFormat = 'iso' | 'object'
/** A time as a prop takes it: `HH:mm`, `HH:mm:ss`, or a `Time` (anything with an hour, a minute, a second). */
export type TimeInput = string | { readonly hour: number; readonly minute: number; readonly second?: number }

export function toTime(value: unknown): Time | null {
  if (value == null || value === '') return null
  if (typeof value === 'string') {
    const m = /^(\d{2}):(\d{2})(?::(\d{2}))?/.exec(value)
    if (!m) return null
    const [h, min, s] = [Number(m[1]), Number(m[2]), Number(m[3] ?? 0)]
    return h < 24 && min < 60 && s < 60 ? new Time(h, min, s) : null
  }
  if (typeof value === 'object' && 'hour' in value && 'minute' in value) {
    const v = value as { hour: number; minute: number; second?: number }
    return new Time(v.hour, v.minute, v.second ?? 0)
  }
  return null
}

const two = (n: number) => String(n).padStart(2, '0')
export function fromTime(time: Time | null, granularity: Granularity, format: TimeFormat): string | Time | null {
  if (!time) return null
  if (format === 'object') return time
  return granularity === 'second' ? `${two(time.hour)}:${two(time.minute)}:${two(time.second)}` : `${two(time.hour)}:${two(time.minute)}`
}

export function hourCycleOf(locale: string, cycle?: 12 | 24): 12 | 24 {
  if (cycle) return cycle
  const hc = new Intl.DateTimeFormat(locale, { hour: 'numeric' }).resolvedOptions().hourCycle
  return hc === 'h11' || hc === 'h12' ? 12 : 24
}

export type SegmentType = 'hour' | 'minute' | 'second' | 'dayPeriod'
export type Part = { type: SegmentType } | { type: 'literal'; text: string }

const sample = (hour: number) => new Date(Date.UTC(2026, 0, 1, hour, 5, 7))
const formatOf = (locale: string, cycle: 12 | 24, granularity: Granularity) =>
  new Intl.DateTimeFormat(locale, {
    hour: 'numeric',
    minute: granularity === 'hour' ? undefined : '2-digit',
    second: granularity === 'second' ? '2-digit' : undefined,
    hourCycle: cycle === 12 ? 'h12' : 'h23',
    timeZone: 'UTC',
  })

export function segmentsOf(locale: string, cycle: 12 | 24, granularity: Granularity): Part[] {
  const parts: Part[] = []
  for (const p of formatOf(locale, cycle, granularity).formatToParts(sample(13))) {
    if (p.type === 'hour' || p.type === 'minute' || p.type === 'second' || p.type === 'dayPeriod') parts.push({ type: p.type })
    else if (p.type === 'literal' && parts.length) parts.push({ type: 'literal', text: p.value })
  }
  // No text after the last segment.
  while (parts.length && parts[parts.length - 1]!.type === 'literal') parts.pop()
  return parts
}

export function dayPeriods(locale: string): [string, string] {
  const f = new Intl.DateTimeFormat(locale, { hour: 'numeric', hourCycle: 'h12', timeZone: 'UTC' })
  const word = (h: number) => f.formatToParts(sample(h)).find((p) => p.type === 'dayPeriod')?.value ?? (h < 12 ? 'AM' : 'PM')
  return [word(9), word(21)]
}

export function digits(n: number, locale: string, width: number): string {
  return new Intl.NumberFormat(locale, { minimumIntegerDigits: width, useGrouping: false }).format(n)
}

/** The names of the parts, for assistive technology; give some or all to translate them. */
export interface InputTimeLabels {
  hour: string
  minute: string
  second: string
  dayPeriod: string
  empty: string
}
