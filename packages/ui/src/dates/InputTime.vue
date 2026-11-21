<script setup vapor lang="ts">
// A time of day. On pointer it is a group of spinbuttons, one per part of
// the time in the locale's order (hour, minute, second, AM/PM; 12- or
// 24-hour as the locale has it), each a tab stop: Up and Down step
// (wrapping), Home and End go to the ends, digits type in (moving on when
// the part is full), A and P set AM and PM, Backspace clears, Left and
// Right move between parts (mirrored right to left). On touch it is the
// platform's time field, whose picker the phone draws, with the same min,
// max and step. It binds HH:mm (HH:mm:ss with granularity second) or, with
// value-format="object", a Time; null until every part is filled. A time
// outside min and max is marked invalid. Inside a FormField the label names
// the group (aria-labelledby).
import { computed, nextTick, reactive, useTemplateRef, watch } from 'vue'
import Icon from '../basics/Icon.vue'
import { useWeft } from '../adaptive/context.ts'
import { focus, isRtl } from '../headless/dom.ts'
import { useControllable } from '../headless/useControllable.ts'
import { useControlAttrs, useField } from '../forms/field.ts'
import { leavesGroup } from '../forms/dom.ts'
import { useLocale } from '../adaptive/locale.ts'
import { dayPeriods, digits, fromTime, hourCycleOf, segmentsOf, Time, toTime, type Granularity, type InputTimeLabels, type SegmentType, type TimeFormat, type TimeInput } from './time.ts'
import type { Size } from '../types.ts'
import '../styles/base.css'
import '../styles/input.css'
import '../styles/input-time.css'

defineOptions({ inheritAttrs: false })


const props = withDefaults(
  defineProps<{
    /** The time, bound with `v-model`: `HH:mm` (`HH:mm:ss` with granularity second), or a Time with `value-format="object"`; null while a part is empty. */
    modelValue?: TimeInput | null
    /** The time it starts with when `v-model` is not bound. */
    defaultValue?: TimeInput | null
    /** `iso`: bound as an `HH:mm` string. `object`: as a `Time`. */
    valueFormat?: TimeFormat
    /** The smallest part shown: `hour`, `minute` or `second`. */
    granularity?: Granularity
    /** 12 or 24-hour time; the locale's when not set. */
    hourCycle?: 12 | 24
    /** How far Up and Down step each part. */
    step?: { hour?: number; minute?: number; second?: number }
    /** The earliest time allowed; an earlier one is marked invalid. */
    min?: TimeInput
    /** The latest time allowed; a later one is marked invalid. */
    max?: TimeInput
    /** The BCP 47 locale of the order, digits and AM/PM; a WeftProvider's applies when not set, else en-US. */
    locale?: string
    /** `sm`, `md` or `lg`; a FieldGroup's or FormField's size applies when not set. md is 44 px tall on touch. */
    size?: Size
    /** A Lucide icon before the time. The leading slot replaces it. */
    icon?: string
    /** Unavailable: nothing can be focused or changed. */
    disabled?: boolean
    /** Shown but not editable. */
    readonly?: boolean
    /** A time is required (aria-required; native `required` on touch). */
    required?: boolean
    /** The form field name its time submits under; a FormField's name applies when not set. */
    name?: string
    /** The group's id (the field's on touch); inside a FormField the field's id is used. */
    id?: string
    /** The words it says, to translate them. */
    labels?: Partial<InputTimeLabels>
  }>(),
  { modelValue: undefined, defaultValue: null, valueFormat: 'iso', granularity: 'minute' },
)

const emit = defineEmits<{
  /** The time changed: `HH:mm`, a Time, or null. */
  'update:modelValue': [value: TimeInput | null]
  /** The time was changed by a key or by the platform's picker. */
  change: [value: TimeInput | null]
  /** Focus left the time. */
  blur: [event: FocusEvent]
  /** The time took focus. */
  focus: [event: FocusEvent]
}>()

defineSlots<{
  /** Before the time, in place of `icon`. */
  leading?: () => unknown
}>()

const f = useField(props)
// A group is not a labelable element: the label names it with aria-labelledby.
f.field?.namedByLabel()
const { own, box, describedBy } = useControlAttrs(f)
const model = useControllable(props, 'modelValue', emit, () => props.defaultValue)
const weft = useWeft()
const touch = computed(() => weft.input.value === 'touch')
const locale = useLocale(() => props.locale)
const cycle = computed(() => hourCycleOf(locale.value, props.hourCycle))
const parts = computed(() => segmentsOf(locale.value, cycle.value, props.granularity))
const periods = computed(() => dayPeriods(locale.value))
const t = computed(() => ({ hour: 'Hour', minute: 'Minute', second: 'Second', dayPeriod: 'AM/PM', empty: 'Empty', ...props.labels }))
const group = useTemplateRef<HTMLElement>('group')

// The parts typed so far; the hour as shown (1–12 in 12-hour time).
const state = reactive<{ hour: number | null; minute: number | null; second: number | null; period: 0 | 1 | null }>({ hour: null, minute: null, second: null, period: null })
function load(time: Time | null) {
  state.hour = time ? (cycle.value === 12 ? time.hour % 12 || 12 : time.hour) : null
  state.minute = time?.minute ?? null
  state.second = time?.second ?? null
  state.period = time ? (time.hour >= 12 ? 1 : 0) : null
}
/** The time the parts make, or null while one is missing. */
function current(): Time | null {
  const { hour, minute, second, period } = state
  if (hour === null || (cycle.value === 12 && period === null)) return null
  if (props.granularity !== 'hour' && minute === null) return null
  if (props.granularity === 'second' && second === null) return null
  const h = cycle.value === 12 ? (hour % 12) + (period === 1 ? 12 : 0) : hour
  return new Time(h, props.granularity === 'hour' ? 0 : minute!, props.granularity === 'second' ? second! : 0)
}
const iso = (time: Time | null) => fromTime(time, props.granularity, 'iso') as string | null
watch(
  [() => iso(toTime(model.value)), cycle, () => props.granularity],
  () => {
    const bound = toTime(model.value)
    if (iso(bound) !== iso(current())) load(bound)
  },
  { immediate: true },
)
function commit() {
  const time = current()
  if (iso(time) === iso(toTime(model.value))) return
  const value = fromTime(time, props.granularity, props.valueFormat) as TimeInput | null
  model.value = value
  emit('change', value)
  f.input()
}

const min = computed(() => toTime(props.min))
const max = computed(() => toTime(props.max))
const invalid = computed(() => {
  const time = toTime(model.value)
  return f.invalid.value || (!!time && ((!!min.value && time.compare(min.value) < 0) || (!!max.value && time.compare(max.value) > 0)))
})

// Each part's range, value and text.
const range = (type: SegmentType): [number, number] =>
  type === 'hour' ? (cycle.value === 12 ? [1, 12] : [0, 23]) : type === 'dayPeriod' ? [0, 1] : [0, 59]
const valueOf = (type: SegmentType): number | null => (type === 'dayPeriod' ? state.period : state[type])
function setPart(type: SegmentType, n: number | null) {
  if (type === 'dayPeriod') state.period = n as 0 | 1 | null
  else state[type] = n
  commit()
}
function textOf(type: SegmentType): string {
  const n = valueOf(type)
  if (n === null) return '––'
  if (type === 'dayPeriod') return periods.value[n]!
  return digits(n, locale.value, type === 'hour' && cycle.value === 12 ? 1 : 2)
}
const segments = computed(() =>
  parts.value.map((p, i) =>
    p.type === 'literal'
      ? { key: `l${i}`, literal: true as const, text: p.text }
      : { key: p.type, literal: false as const, type: p.type, label: t.value[p.type], range: range(p.type), value: valueOf(p.type), text: textOf(p.type) },
  ),
)

/** Digits typed into the focused part so far. */
let typed = ''
const stepOf = (type: SegmentType) => (type === 'dayPeriod' ? 1 : Math.max(1, props.step?.[type] ?? 1))
function bump(type: SegmentType, k: 1 | -1) {
  const [lo, hi] = range(type)
  const n = valueOf(type)
  if (n === null) return setPart(type, k > 0 ? lo : hi)
  const span = hi - lo + 1
  setPart(type, ((n - lo + k * stepOf(type)) % span + span) % span + lo)
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
/** Focus the part `k` places from `el` (in the time's order). */
function move(el: HTMLElement, k: 1 | -1) {
  const all = [...(group.value?.querySelectorAll<HTMLElement>('[role="spinbutton"]') ?? [])]
  const next = all[all.indexOf(el) + k]
  if (next) focus(next)
}
/** Digits in the locale's numbering read as 0–9. */
const digitOf = (key: string) => {
  for (let i = 0; i <= 9; i++) if (digits(i, locale.value, 1) === key || String(i) === key) return String(i)
  return null
}

function onKeydown(event: KeyboardEvent, type: SegmentType) {
  if (f.disabled.value || event.ctrlKey || event.metaKey || event.altKey) return
  const el = event.currentTarget as HTMLElement
  const rtl = isRtl(group.value)
  const editable = !props.readonly
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
function onSegmentFocus() {
  typed = ''
}

// Touch: the platform's time field.
const nativeStep = computed(() => {
  if (props.granularity === 'second') return props.step?.second ?? 1
  if (props.granularity === 'hour') return 3600
  const minutes = props.step?.minute ?? 1
  return minutes === 1 ? undefined : minutes * 60
})
const nativeGranularity = computed<Granularity>(() => (props.granularity === 'second' ? 'second' : 'minute'))
const nativeValue = computed(() => (fromTime(toTime(model.value), nativeGranularity.value, 'iso') as string | null) ?? '')
function onNative(event: Event) {
  load(toTime((event.target as HTMLInputElement).value))
  commit()
}

// The input switching moves focus to the same place in the other form.
watch(
  touch,
  () => {
    if (typeof document === 'undefined' || !group.value?.contains(document.activeElement)) return
    void nextTick(() => focus(group.value?.querySelector<HTMLElement>('input, [role="spinbutton"]')))
  },
  { flush: 'pre' },
)

function onFocusout(event: FocusEvent) {
  if (!leavesGroup(event)) return
  typed = ''
  f.blur()
  emit('blur', event)
}
function onFocusin(event: FocusEvent) {
  const from = event.relatedTarget as Node | null
  if (!from || !group.value?.contains(from)) emit('focus', event)
}
const hidden = computed(() => iso(toTime(model.value)) ?? '')
</script>

<template>
  <span
    ref="group"
    class="weft-input weft-input-time"
    v-bind="{ ...box, ...(touch ? {} : own) }"
    :id="touch ? undefined : f.id.value"
    :role="touch ? undefined : 'group'"
    :aria-labelledby="touch ? undefined : f.field?.labelId"
    :aria-describedby="touch ? undefined : describedBy()"
    :data-form="touch ? 'native' : 'segments'"
    :data-size="f.size.value"
    :data-invalid="invalid ? 'true' : undefined"
    :data-disabled="f.disabled.value ? 'true' : undefined"
    @focusout="onFocusout"
    @focusin="onFocusin"
  >
    <span v-if="$slots.leading || icon" class="weft-input__leading">
      <slot name="leading"><Icon v-if="icon" :name="icon" /></slot>
    </span>
    <input
      v-if="touch"
      class="weft-input__field"
      type="time"
      v-bind="own"
      :id="f.id.value"
      :value="nativeValue"
      :min="min ? fromTime(min, nativeGranularity, 'iso') as string : undefined"
      :max="max ? fromTime(max, nativeGranularity, 'iso') as string : undefined"
      :step="nativeStep"
      :disabled="f.disabled.value || undefined"
      :readonly="readonly || undefined"
      :required="required || undefined"
      :aria-labelledby="f.field?.labelId"
      :aria-describedby="describedBy()"
      :aria-invalid="invalid ? 'true' : undefined"
      @input="onNative"
    />
    <span v-else class="weft-input-time__segments">
      <template v-for="s in segments" :key="s.key">
        <span v-if="s.literal" class="weft-input-time__literal" aria-hidden="true">{{ s.text }}</span>
        <span
          v-else
          role="spinbutton"
          class="weft-input-time__segment"
          inputmode="numeric"
          :data-segment="s.type"
          :data-placeholder="s.value === null ? 'true' : undefined"
          :tabindex="f.disabled.value ? -1 : 0"
          :aria-label="s.label"
          :aria-valuemin="s.range[0]"
          :aria-valuemax="s.range[1]"
          :aria-valuenow="s.value ?? undefined"
          :aria-valuetext="s.value === null ? t.empty : s.text"
          :aria-invalid="invalid ? 'true' : undefined"
          :aria-required="required || f.aria.value['aria-required'] ? 'true' : undefined"
          :aria-disabled="f.disabled.value ? 'true' : undefined"
          :aria-readonly="readonly ? 'true' : undefined"
          @keydown="onKeydown($event, s.type)"
          @focus="onSegmentFocus"
        >{{ s.text }}</span>
      </template>
    </span>
    <input v-if="f.name.value" type="hidden" :name="f.name.value" :value="hidden" :disabled="f.disabled.value || undefined" />
  </span>
</template>
