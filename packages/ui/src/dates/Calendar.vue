<script setup vapor lang="ts">
// A month of dates to choose one, several or a range from (the grid of the
// WAI-ARIA APG Date Picker Dialog). Each month is a table with role grid,
// labelled by the month's heading, which is live; each date is a button in
// a gridcell (aria-selected), named in full ("Saturday, November 21,
// 2026"), today marked aria-current="date". One date is the tab stop, the
// focused date. Keys: Left and Right a day (mirrored right to left), Up and
// Down a week, Home and End the week's first and last day as the locale
// counts it, Page Up and Page Down a month, with Shift a year; the native
// button press (Enter, Space) chooses. The heading switches to a month view
// and a year view, keyed the same way. Disabled and unavailable dates can
// be reached but not chosen; dates outside min and max cannot be reached.
// Names, digits and the first weekday follow the locale (its prop, else
// the WeftProvider's, else en-US). On touch every date is at least 44 px a
// side and a swipe changes the month.
import { computed, nextTick, onMounted, shallowRef, useTemplateRef, watch } from 'vue'
import { endOfMonth, endOfWeek, getLocalTimeZone, isSameMonth, startOfMonth, startOfWeek, today } from '@internationalized/date'
import { useWeft } from '../adaptive/context.ts'
import { focus, isRtl } from '../headless/dom.ts'
import { useControllable } from '../headless/useControllable.ts'
import { useId } from '../headless/useId.ts'
import { createSwipe } from '../headless/swipe.ts'
import { CalendarDate, clamp, dayOfWeek, format, formatter, fromDate, isoWeek, monthWeeks, sameDay, toDate, utc, weekdays, type CalendarLabels, type CalendarModel, type DateInput, type ValueFormat } from './dates.ts'
import { useLocale } from '../adaptive/locale.ts'
import type { Size, Tone } from '../types.ts'
import '../styles/base.css'
import '../styles/tone.css'
import '../styles/calendar.css'


const props = withDefaults(
  defineProps<{
    /** The chosen date (an ISO string, or a CalendarDate with `value-format="object"`), bound with `v-model`; an array with `multiple`; `{ start, end }` with `range`. */
    modelValue?: CalendarModel
    /** The value it starts with when `v-model` is not bound. */
    defaultValue?: CalendarModel
    /** Choose a range: a start, then an end. */
    range?: boolean
    /** Choose several dates: a press toggles one. */
    multiple?: boolean
    /** `iso`: dates are bound as `YYYY-MM-DD` strings. `object`: as `CalendarDate` objects. */
    valueFormat?: ValueFormat
    /** The earliest date that can be reached or chosen. */
    min?: DateInput
    /** The latest date that can be reached or chosen. */
    max?: DateInput
    /** Dates that cannot be chosen (the keyboard still reaches them). */
    isDateDisabled?: (date: CalendarDate) => boolean
    /** Dates drawn struck through, taken (booked, sold out): they cannot be chosen either. */
    isDateUnavailable?: (date: CalendarDate) => boolean
    /** The BCP 47 locale of names, digits and the first weekday; a WeftProvider's applies when not set, else en-US. */
    locale?: string
    /** The first day of the week, 0 Sunday to 6 Saturday; the locale's when not set. */
    weekStartsOn?: 0 | 1 | 2 | 3 | 4 | 5 | 6
    /** How the weekday headers are written: `narrow` (M), `short` (Mon) or `long` (Monday). */
    weekdayFormat?: 'narrow' | 'short' | 'long'
    /** Months shown side by side. */
    numberOfMonths?: number
    /** Always six weeks, so the grid keeps its height from month to month. */
    fixedWeeks?: boolean
    /** ISO 8601 week numbers before each week. */
    weekNumbers?: boolean
    /** Previous and next month buttons. */
    monthControls?: boolean
    /** Previous and next year buttons (in the days view). */
    yearControls?: boolean
    /** The heading is a button switching to a month view, then a year view. */
    viewControl?: boolean
    /** The date that has the tab stop and whose month shows, bound with `v-model:focused-date`. */
    focusedDate?: DateInput
    /** Nothing can be focused or chosen. */
    disabled?: boolean
    /** Dates can be focused but not chosen. */
    readonly?: boolean
    /** `sm`, `md` or `lg`: the size of a date. On touch a date is at least 44 px whatever the size. */
    size?: Size
    /** The colour of the chosen dates. */
    tone?: Tone
    /** The words it says, to translate them. */
    labels?: Partial<CalendarLabels>
  }>(),
  {
    modelValue: undefined,
    defaultValue: undefined,
    focusedDate: undefined,
    valueFormat: 'iso',
    weekdayFormat: 'short',
    numberOfMonths: 1,
    fixedWeeks: true,
    monthControls: true,
    yearControls: true,
    viewControl: true,
    size: 'md',
    tone: 'primary',
  },
)

const emit = defineEmits<{
  /** Chosen: a date; with `multiple` the sorted array; with `range` `{ start, end }`, once both ends are chosen. */
  'update:modelValue': [value: CalendarModel]
  /** The focused date moved (by key, by page, by a press). */
  'update:focusedDate': [date: DateInput]
}>()

defineSlots<{
  /** A date's content, in place of its number. */
  day?: (props: { date: CalendarDate; selected: boolean; disabled: boolean; today: boolean; outside: boolean }) => unknown
  /** The heading's text, in place of the month and year. */
  heading?: (props: { value: string; view: 'day' | 'month' | 'year' }) => unknown
  /** A weekday header, in place of its name. */
  'week-day'?: (props: { day: { short: string; narrow: string; long: string } }) => unknown
}>()

const LABELS: CalendarLabels = {
  prevMonth: 'Previous month',
  nextMonth: 'Next month',
  prevYear: 'Previous year',
  nextYear: 'Next year',
  prevYears: 'Previous years',
  nextYears: 'Next years',
  week: 'Week',
}
const t = computed(() => ({ ...LABELS, ...props.labels }))
const locale = useLocale(() => props.locale)
const firstDay = computed(() => dayOfWeek(props.weekStartsOn))
const weft = useWeft()
const touch = computed(() => weft.input.value === 'touch')
const id = useId('calendar')
const root = useTemplateRef<HTMLElement>('root')
const min = computed(() => toDate(props.min))
const max = computed(() => toDate(props.max))
const out = (d: CalendarDate) => fromDate(d, props.valueFormat) as DateInput

// The value, bound or not.
const model = useControllable(props, 'modelValue', emit, () => props.defaultValue ?? (props.multiple ? [] : null))
const write = (value: CalendarModel) => (model.value = value)
const single = computed(() => (props.range || props.multiple ? null : toDate(model.value)))
const many = computed(() =>
  props.multiple && Array.isArray(model.value) ? model.value.map(toDate).filter((d): d is CalendarDate => d !== null) : [],
)
const chosenRange = computed(() => {
  const v = model.value
  if (!props.range || !v || typeof v !== 'object' || Array.isArray(v) || !('start' in v)) return null
  return { start: toDate(v.start), end: toDate(v.end) }
})

// Today, marked once mounted: the server and the browser may not agree on the day.
const now = shallowRef<CalendarDate | null>(null)
onMounted(() => (now.value = today(getLocalTimeZone())))

const firstChosen = () => single.value ?? many.value[0] ?? chosenRange.value?.start ?? null
const focused = shallowRef<CalendarDate>(clamp(toDate(props.focusedDate) ?? firstChosen() ?? today(getLocalTimeZone()), min.value, max.value))
watch(
  () => toDate(props.focusedDate),
  (d) => {
    if (d && !sameDay(d, focused.value)) focused.value = clamp(d, min.value, max.value)
  },
)
function setFocused(d: CalendarDate) {
  const next = clamp(d, min.value, max.value)
  if (sameDay(next, focused.value)) return
  focused.value = next
  emit('update:focusedDate', out(next))
}

// The months on show: from `start`, as many as asked; the focused date is always among them.
const start = shallowRef(startOfMonth(focused.value))
function reveal() {
  const d = focused.value
  const n = Math.max(1, props.numberOfMonths)
  if (d.compare(start.value) < 0) start.value = startOfMonth(d)
  else if (d.compare(endOfMonth(start.value.add({ months: n - 1 }))) > 0) start.value = startOfMonth(d).subtract({ months: n - 1 })
}
watch([focused, () => props.numberOfMonths], reveal, { flush: 'sync' })

const view = shallowRef<'day' | 'month' | 'year'>('day')
const fmt = (d: CalendarDate, o: Intl.DateTimeFormatOptions) => format(d, locale.value, o)
const months = computed(() =>
  Array.from({ length: Math.max(1, props.numberOfMonths) }, (_, i) => {
    const m = start.value.add({ months: i })
    const weeks = monthWeeks(m, locale.value, firstDay.value, props.fixedWeeks)
    return { key: m.toString(), date: m, labelId: `${id}-month-${i}`, label: fmt(m, { month: 'long', year: 'numeric' }), weeks: weeks.map((w) => ({ number: weekOf(w), days: w.map((d) => dayState(d, m)) })) }
  }),
)
const names = computed(() => weekdays(locale.value, firstDay.value))
/** The ISO week of the week holding `days`: its Thursday's. */
function weekOf(days: CalendarDate[]) {
  return isoWeek(days.find((d) => utc(d).getUTCDay() === 4) ?? days[0]!)
}
const yearStart = computed(() => Math.floor(focused.value.year / 12) * 12)
const headingText = computed(() => {
  if (view.value === 'month') return fmt(focused.value, { year: 'numeric' })
  if (view.value === 'year') {
    const f = formatter(locale.value, { year: 'numeric' })
    return f.formatRange(utc(new CalendarDate(yearStart.value, 1, 1)), utc(new CalendarDate(yearStart.value + 11, 1, 1)))
  }
  const n = months.value.length
  if (n === 1) return months.value[0]!.label
  return formatter(locale.value, { month: 'long', year: 'numeric' }).formatRange(utc(start.value), utc(start.value.add({ months: n - 1 })))
})
const headingId = `${id}-heading`

// Dates.
const outside = (d: CalendarDate, month: CalendarDate) => !isSameMonth(d, month)
const beyond = (d: CalendarDate) => (min.value !== null && d.compare(min.value) < 0) || (max.value !== null && d.compare(max.value) > 0)
const unavailable = (d: CalendarDate) => !!props.isDateUnavailable?.(d)
const blocked = (d: CalendarDate) => beyond(d) || !!props.isDateDisabled?.(d) || unavailable(d)

// A range being chosen: its start, and the date the pointer or the focus is on.
const pending = shallowRef<CalendarDate | null>(null)
const hovered = shallowRef<CalendarDate | null>(null)
watch(
  chosenRange,
  (r) => {
    // A range bound with a start and no end is still being chosen.
    pending.value = r?.start && !r.end ? r.start : null
  },
  { immediate: true },
)
const ordered = (a: CalendarDate, b: CalendarDate): [CalendarDate, CalendarDate] => (a.compare(b) <= 0 ? [a, b] : [b, a])
const shownRange = computed(() => {
  if (pending.value) {
    const [s, e] = ordered(pending.value, hovered.value ?? pending.value)
    return { start: s, end: e, preview: !sameDay(s, e) }
  }
  const r = chosenRange.value
  if (r?.start && r.end) return { start: r.start, end: r.end, preview: false }
  return null
})
function rangePart(d: CalendarDate): string | undefined {
  const r = shownRange.value
  if (!r || d.compare(r.start) < 0 || d.compare(r.end) > 0) return undefined
  const parts = [sameDay(d, r.start) && 'start', sameDay(d, r.end) && 'end'].filter(Boolean)
  return parts.length ? parts.join(' ') : 'middle'
}
function isSelected(d: CalendarDate): boolean {
  if (props.range) {
    const r = shownRange.value
    if (pending.value) return sameDay(d, pending.value)
    return !!r && d.compare(r.start) >= 0 && d.compare(r.end) <= 0
  }
  if (props.multiple) return many.value.some((m) => sameDay(m, d))
  return sameDay(single.value, d)
}
function dayState(d: CalendarDate, month: CalendarDate) {
  const off = outside(d, month)
  if (off && props.numberOfMonths > 1) return { date: d, iso: d.toString(), off, hidden: true } as const
  const selected = isSelected(d)
  const isToday = sameDay(now.value, d)
  const no = blocked(d)
  return {
    date: d,
    hidden: false,
    iso: d.toString(),
    off,
    selected,
    today: isToday,
    disabled: no,
    unavailable: unavailable(d),
    stop: !off && sameDay(d, focused.value),
    range: props.range ? rangePart(d) : undefined,
    preview: props.range && !!shownRange.value?.preview && !!rangePart(d),
    label: fmt(d, { weekday: 'long', year: 'numeric', month: 'long', day: 'numeric' }),
    text: fmt(d, { day: 'numeric' }),
  }
}

/** Whether no date between `a` and `b` (exclusive) is blocked. */
function clear(a: CalendarDate, b: CalendarDate): boolean {
  for (let d = a.add({ days: 1 }); d.compare(b) < 0; d = d.add({ days: 1 })) if (blocked(d)) return false
  return true
}

/** When a swipe last changed the month: the click the lifted finger then sends chooses nothing. */
let swipedAt = -Infinity
function choose(d: CalendarDate, event?: MouseEvent) {
  // A pointer's click (detail > 0) just after a swipe; a key's click always counts.
  if (event && event.detail > 0 && Date.now() - swipedAt < 500) return
  if (props.disabled) return
  setFocused(d)
  if (props.readonly || blocked(d)) return
  if (props.range) {
    const p = pending.value
    if (!p) {
      pending.value = d
      hovered.value = null
      return
    }
    const [s, e] = ordered(p, d)
    if (!clear(s, e)) {
      pending.value = d
      hovered.value = null
      return
    }
    pending.value = null
    hovered.value = null
    write({ start: out(s), end: out(e) })
    return
  }
  if (props.multiple) {
    const has = many.value.some((m) => sameDay(m, d))
    const next = has ? many.value.filter((m) => !sameDay(m, d)) : [...many.value, d].sort((a, b) => a.compare(b))
    write(next.map(out))
    return
  }
  if (sameDay(single.value, d)) return
  write(out(d))
}
function hover(d: CalendarDate) {
  if (pending.value) hovered.value = d
}

/** Focus the tab stop once the render that moved it is done; unless `force`, only when focus is in the calendar now. */
function focusStop(force = false) {
  if (!force && !root.value?.contains(document.activeElement)) return
  void nextTick(() => focus(root.value?.querySelector<HTMLElement>('[data-weft-autofocus]')))
}
function moveTo(d: CalendarDate) {
  setFocused(d)
  if (pending.value) hovered.value = focused.value
  focusStop(true)
}

function onKeydown(event: KeyboardEvent) {
  if (event.defaultPrevented || event.altKey || event.metaKey || event.ctrlKey || props.disabled) return
  const target = event.target as HTMLElement
  if (!target.closest('.weft-calendar__day, .weft-calendar__cell')) return
  const rtl = isRtl(root.value)
  const d = focused.value
  const forward = rtl ? -1 : 1
  // Per view: one step, a row, a page.
  const step = view.value === 'day' ? { days: 1 } : view.value === 'month' ? { months: 1 } : { years: 1 }
  const n = (k: number) => Object.fromEntries(Object.entries(step).map(([u, v]) => [u, v * k]))
  let next: CalendarDate | null = null
  switch (event.key) {
    case 'ArrowRight': next = d.add(n(forward)); break
    case 'ArrowLeft': next = d.add(n(-forward)); break
    case 'ArrowDown': next = d.add(n(view.value === 'day' ? 7 : 3)); break
    case 'ArrowUp': next = d.add(n(view.value === 'day' ? -7 : -3)); break
    case 'Home':
    case 'End': {
      const end = event.key === 'End'
      if (view.value === 'day') next = end ? endOfWeek(d, locale.value, firstDay.value) : startOfWeek(d, locale.value, firstDay.value)
      else if (view.value === 'month') next = d.set({ month: Math.floor((d.month - 1) / 3) * 3 + (end ? 3 : 1) })
      else next = d.set({ year: d.year - ((d.year - yearStart.value) % 3) + (end ? 2 : 0) })
      break
    }
    case 'PageUp':
    case 'PageDown': {
      const k = event.key === 'PageDown' ? 1 : -1
      if (view.value === 'day') next = event.shiftKey ? d.add({ years: k }) : d.add({ months: k })
      else next = d.add({ years: view.value === 'month' ? k : 12 * k })
      break
    }
  }
  if (!next) return
  event.preventDefault()
  moveTo(next)
}
function onFocusin(event: FocusEvent) {
  // A date focused by a press becomes the focused date (not one of another month's).
  const iso = (event.target as HTMLElement).closest<HTMLElement>('[data-date]:not([data-outside])')?.dataset.date
  const d = iso ? toDate(iso) : null
  if (d && !sameDay(d, focused.value)) setFocused(d)
}

// Paging with the buttons (and swipes): the months on show move by a month (a
// year with the year buttons), the focused date with them.
function page(k: 1 | -1, years = false) {
  if (props.disabled) return
  if (view.value === 'day') {
    const by = years ? { years: k } : { months: k }
    start.value = start.value.add(by)
    setFocused(focused.value.add(by))
    reveal()
  } else setFocused(focused.value.add({ years: view.value === 'month' ? k : 12 * k }))
}
/** Whether a page back (or forward) of `by` months would show only dates outside min (or max). */
function stop(k: 1 | -1, by: number): boolean {
  if (props.disabled) return true
  if (view.value !== 'day') {
    const years = view.value === 'month' ? 1 : 12
    if (k < 0) return min.value !== null && (view.value === 'month' ? focused.value.year - 1 : yearStart.value - 1) < min.value.year
    return max.value !== null && (view.value === 'month' ? focused.value.year + 1 : yearStart.value + years) > max.value.year
  }
  const n = months.value.length
  if (k < 0) return min.value !== null && endOfMonth(start.value.subtract({ months: by }).add({ months: n - 1 })).compare(min.value) < 0
  return max.value !== null && start.value.add({ months: by }).compare(max.value) > 0
}
const prevLabel = computed(() => (view.value === 'day' ? t.value.prevMonth : view.value === 'month' ? t.value.prevYear : t.value.prevYears))
const nextLabel = computed(() => (view.value === 'day' ? t.value.nextMonth : view.value === 'month' ? t.value.nextYear : t.value.nextYears))

function cycleView() {
  view.value = view.value === 'day' ? 'month' : view.value === 'month' ? 'year' : 'day'
}
const monthRows = computed(() =>
  [0, 1, 2, 3].map((r) =>
    [1, 2, 3].map((c) => {
      const m = r * 3 + c
      const date = focused.value.set({ month: m, day: 1 })
      const last = endOfMonth(date)
      const chosen = firstChosen()
      return {
        key: m,
        text: fmt(date, { month: 'short' }),
        label: fmt(date, { month: 'long', year: 'numeric' }),
        stop: focused.value.month === m,
        selected: !!chosen && chosen.year === date.year && chosen.month === m,
        disabled: (min.value !== null && last.compare(min.value) < 0) || (max.value !== null && date.compare(max.value) > 0),
        pick: () => pickMonth(m),
      }
    }),
  ),
)
const yearRows = computed(() =>
  [0, 1, 2, 3].map((r) =>
    [0, 1, 2].map((c) => {
      const y = yearStart.value + r * 3 + c
      const date = new CalendarDate(y, 1, 1)
      return {
        key: y,
        text: fmt(date, { year: 'numeric' }),
        label: undefined,
        stop: focused.value.year === y,
        selected: firstChosen()?.year === y,
        disabled: (min.value !== null && y < min.value.year) || (max.value !== null && y > max.value.year),
        pick: () => pickYear(y),
      }
    }),
  ),
)
function pickMonth(m: number) {
  if (props.disabled) return
  setFocused(focused.value.set({ month: m }))
  view.value = 'day'
  focusStop()
}
function pickYear(y: number) {
  if (props.disabled) return
  setFocused(focused.value.set({ year: y }))
  view.value = 'month'
  focusStop()
}

// Touch: a swipe across the dates changes the month (towards the end of the
// line: the next one; mirrored right to left). A mouse never swipes.
const towardsEnd = createSwipe(() => ({ axis: 'x', sign: 1 }))
const towardsStart = createSwipe(() => ({ axis: 'x', sign: -1 }))
const SWIPE = 40
function onPointerdown(event: PointerEvent) {
  if (!touch.value || event.pointerType === 'mouse' || view.value !== 'day' || props.disabled) return
  towardsEnd.start(event)
  towardsStart.start(event)
}
function onPointermove(event: PointerEvent) {
  towardsEnd.move(event)
  towardsStart.move(event)
}
function onPointerup(event: PointerEvent) {
  const right = towardsEnd.end(event)
  const left = towardsStart.end(event)
  const far = (s: { distance: number; speed: number } | null) => !!s && (s.distance >= SWIPE || (s.distance >= SWIPE / 3 && s.speed > 0.5))
  if (!far(right) && !far(left)) return
  swipedAt = Date.now()
  // A finger moving right shows the past, left to right.
  const back = far(right) !== isRtl(root.value)
  page(back ? -1 : 1)
}
function onPointercancel(event: PointerEvent) {
  towardsEnd.end(event)
  towardsStart.end(event)
}
function onPointerleave() {
  if (pending.value) hovered.value = null
}

defineExpose({
  /** Focus the focused date (or month, or year). */
  focus: () => focusStop(true),
})
</script>

<template>
  <div ref="root" class="weft-calendar" :data-size="size" :data-tone="tone" :data-view="view" :data-disabled="disabled ? 'true' : undefined">
    <div class="weft-calendar__header">
      <button
        v-if="view === 'day' && yearControls"
        type="button"
        class="weft-calendar__nav"
        :aria-label="t.prevYear"
        :disabled="stop(-1, 12) || undefined"
        @click="page(-1, true)"
      ><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="m11 17-5-5 5-5" /><path d="m18 17-5-5 5-5" /></svg></button>
      <button
        v-if="view !== 'day' || monthControls"
        type="button"
        class="weft-calendar__nav"
        :aria-label="prevLabel"
        :disabled="stop(-1, 1) || undefined"
        @click="page(-1)"
      ><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="m15 18-6-6 6-6" /></svg></button>
      <h2 :id="headingId" class="weft-calendar__heading" aria-live="polite">
        <button v-if="viewControl" type="button" class="weft-calendar__view" :disabled="disabled || undefined" @click="cycleView">
          <slot name="heading" :value="headingText" :view="view">{{ headingText }}</slot>
        </button>
        <slot v-else name="heading" :value="headingText" :view="view">{{ headingText }}</slot>
      </h2>
      <button
        v-if="view !== 'day' || monthControls"
        type="button"
        class="weft-calendar__nav"
        :aria-label="nextLabel"
        :disabled="stop(1, 1) || undefined"
        @click="page(1)"
      ><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="m9 18 6-6-6-6" /></svg></button>
      <button
        v-if="view === 'day' && yearControls"
        type="button"
        class="weft-calendar__nav"
        :aria-label="t.nextYear"
        :disabled="stop(1, 12) || undefined"
        @click="page(1, true)"
      ><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="m6 17 5-5-5-5" /><path d="m13 17 5-5-5-5" /></svg></button>
    </div>
    <div
      class="weft-calendar__body"
      @keydown="onKeydown"
      @focusin="onFocusin"
      @pointerdown="onPointerdown"
      @pointermove="onPointermove"
      @pointerup="onPointerup"
      @pointercancel="onPointercancel"
      @pointerleave="onPointerleave"
    >
      <template v-if="view === 'day'">
        <div v-for="m in months" :key="m.key" class="weft-calendar__month">
          <div v-if="months.length > 1" :id="m.labelId" class="weft-calendar__month-label">{{ m.label }}</div>
          <table
            class="weft-calendar__grid"
            role="grid"
            :aria-labelledby="months.length > 1 ? m.labelId : headingId"
            :aria-multiselectable="multiple || range ? 'true' : undefined"
            :aria-readonly="readonly ? 'true' : undefined"
            :aria-disabled="disabled ? 'true' : undefined"
          >
            <thead>
              <tr>
                <th v-if="weekNumbers" scope="col" class="weft-calendar__week"><span class="weft-visually-hidden">{{ t.week }}</span></th>
                <th v-for="w in names" :key="w.long" scope="col" :abbr="w.long" class="weft-calendar__weekday">
                  <slot name="week-day" :day="w">{{ w[weekdayFormat] }}</slot>
                </th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="(week, wi) in m.weeks" :key="wi">
                <th v-if="weekNumbers" scope="row" class="weft-calendar__week">{{ week.number }}</th>
                <td
                  v-for="s in week.days"
                  :key="s.iso"
                  role="gridcell"
                  :aria-selected="s.hidden ? undefined : s.selected ? 'true' : 'false'"
                  :aria-disabled="!s.hidden && s.disabled ? 'true' : undefined"
                >
                  <button
                    v-if="!s.hidden"
                    type="button"
                    class="weft-calendar__day"
                    :tabindex="s.stop ? 0 : -1"
                    :data-date="s.iso"
                    :data-weft-autofocus="s.stop ? '' : undefined"
                    :data-outside="s.off ? 'true' : undefined"
                    :data-selected="s.selected ? 'true' : undefined"
                    :data-today="s.today ? 'true' : undefined"
                    :data-unavailable="s.unavailable ? 'true' : undefined"
                    :data-disabled="s.disabled ? 'true' : undefined"
                    :data-range="s.range"
                    :data-preview="s.preview ? 'true' : undefined"
                    :aria-label="s.label"
                    :aria-current="s.today ? 'date' : undefined"
                    :aria-disabled="s.disabled ? 'true' : undefined"
                    :disabled="disabled || undefined"
                    @click="choose(s.date, $event)"
                    @pointerenter="hover(s.date)"
                  >
                    <slot name="day" :date="s.date" :selected="s.selected" :disabled="s.disabled" :today="s.today" :outside="s.off">{{ s.text }}</slot>
                  </button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </template>
      <table v-else class="weft-calendar__grid" role="grid" :data-view="view" :aria-labelledby="headingId" :aria-disabled="disabled ? 'true' : undefined">
        <tbody>
          <tr v-for="(row, ri) in view === 'month' ? monthRows : yearRows" :key="ri">
            <td v-for="c in row" :key="c.key" role="gridcell" :aria-selected="c.selected ? 'true' : 'false'" :aria-disabled="c.disabled ? 'true' : undefined">
              <button
                type="button"
                class="weft-calendar__cell"
                :tabindex="c.stop ? 0 : -1"
                :data-weft-autofocus="c.stop ? '' : undefined"
                :data-selected="c.selected ? 'true' : undefined"
                :data-disabled="c.disabled ? 'true' : undefined"
                :aria-label="c.label"
                :aria-disabled="c.disabled ? 'true' : undefined"
                :disabled="disabled || undefined"
                @click="!c.disabled && c.pick()"
              >{{ c.text }}</button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>
