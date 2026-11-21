/**
 * useTimeParts(opts): one time typed part by part, for InputTime (once, or
 * twice for a range). It holds the parts typed so far (the hour as shown,
 * 1–12 in 12-hour time), loads them from the bound time when it changes,
 * and calls `onChange` with the time they make (null while one is missing)
 * each time a part changes.
 *
 * - `segments`: the parts in the locale's order with the text between them,
 *   each with its range, value and text, for spinbuttons.
 * - `keydown(event, type)`: a spinbutton's keys (see InputTime). Left and
 *   Right move to the spinbutton before or after it in `group()`, so they
 *   go on from a range's start to its end.
 * - `focus()`: a part took focus: the digits typed start again.
 * - `native` and `input(event)`: the value and the input handler of the
 *   platform's time field.
 */
import { computed, reactive, watch, type ComputedRef } from 'vue'
import { focus, isRtl } from '../headless/dom.ts'
import { dayPeriods, digits, fromTime, segmentsOf, Time, toTime, type Granularity, type InputTimeLabels, type SegmentType } from './time.ts'

export interface TimePartsOptions {
  locale: ComputedRef<string>
  cycle: ComputedRef<12 | 24>
  granularity: () => Granularity
  step: () => { hour?: number; minute?: number; second?: number } | undefined
  labels: () => InputTimeLabels
  /** The bound time. */
  value: () => Time | null
  onChange: (time: Time | null) => void
  editable: () => boolean
  disabled: () => boolean
  /** The element holding every spinbutton, for Left and Right. */
  group: () => HTMLElement | null | undefined
}

export function useTimeParts(opts: TimePartsOptions) {
  const state = reactive<{ hour: number | null; minute: number | null; second: number | null; period: 0 | 1 | null }>({ hour: null, minute: null, second: null, period: null })
  const parts = computed(() => segmentsOf(opts.locale.value, opts.cycle.value, opts.granularity()))
  const periods = computed(() => dayPeriods(opts.locale.value))
  const iso = (time: Time | null) => fromTime(time, opts.granularity(), 'iso') as string | null

  function load(time: Time | null) {
    state.hour = time ? (opts.cycle.value === 12 ? time.hour % 12 || 12 : time.hour) : null
    state.minute = time?.minute ?? null
    state.second = time?.second ?? null
    state.period = time ? (time.hour >= 12 ? 1 : 0) : null
  }
  /** The time the parts make, or null while one is missing. */
  function current(): Time | null {
    const { hour, minute, second, period } = state
    const g = opts.granularity()
    if (hour === null || (opts.cycle.value === 12 && period === null)) return null
    if (g !== 'hour' && minute === null) return null
    if (g === 'second' && second === null) return null
    const h = opts.cycle.value === 12 ? (hour % 12) + (period === 1 ? 12 : 0) : hour
    return new Time(h, g === 'hour' ? 0 : minute!, g === 'second' ? second! : 0)
  }
  watch(
    [() => iso(opts.value()), opts.cycle, opts.granularity],
    () => {
      const bound = opts.value()
      if (iso(bound) !== iso(current())) load(bound)
    },
    { immediate: true },
  )
  const changed = () => opts.onChange(current())

  const range = (type: SegmentType): [number, number] =>
    type === 'hour' ? (opts.cycle.value === 12 ? [1, 12] : [0, 23]) : type === 'dayPeriod' ? [0, 1] : [0, 59]
  const valueOf = (type: SegmentType): number | null => (type === 'dayPeriod' ? state.period : state[type])
  function setPart(type: SegmentType, n: number | null) {
    if (type === 'dayPeriod') state.period = n as 0 | 1 | null
    else state[type] = n
    changed()
  }
  function textOf(type: SegmentType): string {
    const n = valueOf(type)
    if (n === null) return '––'
    if (type === 'dayPeriod') return periods.value[n]!
    return digits(n, opts.locale.value, type === 'hour' && opts.cycle.value === 12 ? 1 : 2)
  }
  const segments = computed(() =>
    parts.value.map((p, i) =>
      p.type === 'literal'
        ? { key: `l${i}`, literal: true as const, text: p.text }
        : { key: p.type, literal: false as const, type: p.type, label: opts.labels()[p.type], range: range(p.type), value: valueOf(p.type), text: textOf(p.type) },
    ),
  )

  /** Digits typed into the focused part so far. */
  let typed = ''
  const stepOf = (type: SegmentType) => (type === 'dayPeriod' ? 1 : Math.max(1, opts.step()?.[type] ?? 1))
  function bump(type: SegmentType, k: 1 | -1) {
    const [lo, hi] = range(type)
    const n = valueOf(type)
    if (n === null) return setPart(type, k > 0 ? lo : hi)
    const span = hi - lo + 1
    setPart(type, ((n - lo + k * stepOf(type)) % span + span) % span + lo)
  }
  /** Focus the spinbutton `k` places from `el`. */
  function move(el: HTMLElement, k: 1 | -1) {
    const all = [...(opts.group()?.querySelectorAll<HTMLElement>('[role="spinbutton"]') ?? [])]
    const next = all[all.indexOf(el) + k]
    if (next) focus(next)
  }
  function typeDigit(type: SegmentType, d: string, el: HTMLElement) {
    const [lo, hi] = range(type)
    let text = typed + d
    if (Number(text) > hi) text = d
    typed = text
    const n = Number(text)
    if (n >= lo) setPart(type, n)
    if (text.length >= 2 || n * 10 > hi) move(el, 1)
  }
  /** Digits in the locale's numbering read as 0–9. */
  const digitOf = (key: string) => {
    for (let i = 0; i <= 9; i++) if (digits(i, opts.locale.value, 1) === key || String(i) === key) return String(i)
    return null
  }

  function keydown(event: KeyboardEvent, type: SegmentType) {
    if (opts.disabled() || event.ctrlKey || event.metaKey || event.altKey) return
    const el = event.currentTarget as HTMLElement
    const rtl = isRtl(opts.group())
    const editable = opts.editable()
    const k = event.key
    if (k === 'ArrowRight' || k === 'ArrowLeft') move(el, (k === 'ArrowRight') !== rtl ? 1 : -1)
    else if (k === 'ArrowUp' || k === 'ArrowDown') editable && bump(type, k === 'ArrowUp' ? 1 : -1)
    else if (k === 'Home' || k === 'End') editable && setPart(type, range(type)[k === 'Home' ? 0 : 1])
    else if (k === 'Backspace' || k === 'Delete') {
      if (!editable) return
      typed = typed.slice(0, -1)
      setPart(type, typed && type !== 'dayPeriod' ? Number(typed) : null)
    } else if (type === 'dayPeriod' && k.length === 1 && /\p{L}/u.test(k)) {
      if (!editable) return
      const fold = (s: string) => s.normalize('NFD').replace(/\p{M}/gu, '').toLowerCase()
      const which = periods.value.findIndex((w, i) => fold(w).startsWith(fold(k)) || fold(k) === (i ? 'p' : 'a'))
      if (which < 0) return
      setPart(type, which)
    } else if (type !== 'dayPeriod' && digitOf(k) !== null) {
      if (editable) typeDigit(type, digitOf(k)!, el)
    } else if (k.length === 1 && /[\s:.,]/.test(k)) move(el, 1)
    else return
    event.preventDefault()
  }

  const nativeGranularity = (): Granularity => (opts.granularity() === 'second' ? 'second' : 'minute')
  return {
    segments,
    keydown,
    focus() {
      typed = ''
    },
    blur() {
      typed = ''
    },
    native: computed(() => (fromTime(opts.value(), nativeGranularity(), 'iso') as string | null) ?? ''),
    input(event: Event) {
      load(toTime((event.target as HTMLInputElement).value))
      changed()
    },
  }
}
