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
// outside min and max is marked invalid. With `range` it is two times, a
// start and an end (each a named group, or a platform field, on touch),
// bound as { start, end }. Inside a FormField the label names the group
// (aria-labelledby).
import { computed, nextTick, useTemplateRef, watch } from 'vue'
import Icon from '../basics/Icon.vue'
import { useWeft } from '../adaptive/context.ts'
import { focus } from '../headless/dom.ts'
import { useControllable } from '../headless/useControllable.ts'
import { useControlAttrs, useField } from '../forms/field.ts'
import { leavesGroup } from '../forms/dom.ts'
import { useLocale } from '../adaptive/locale.ts'
import { fromTime, hourCycleOf, toTime, type Granularity, type InputTimeLabels, type Time, type TimeFormat, type TimeInput, type TimeRange } from './time.ts'
import { useTimeParts } from './timeParts.ts'
import type { Size } from '../types.ts'
import '../styles/base.css'
import '../styles/input.css'
import '../styles/input-time.css'

defineOptions({ inheritAttrs: false })


const props = withDefaults(
  defineProps<{
    /** The time, bound with `v-model`: `HH:mm` (`HH:mm:ss` with granularity second), or a Time with `value-format="object"`; null while a part is empty. With `range`, `{ start, end }`. */
    modelValue?: TimeInput | TimeRange | null
    /** The time it starts with when `v-model` is not bound. */
    defaultValue?: TimeInput | TimeRange | null
    /** Two times, a start and an end, bound as `{ start, end }`; an end before the start is marked invalid. */
    range?: boolean
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
  /** The time changed: `HH:mm`, a Time, `{ start, end }` with range, or null. */
  'update:modelValue': [value: TimeInput | TimeRange | null]
  /** The time was changed by a key or by the platform's picker. */
  change: [value: TimeInput | TimeRange | null]
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
const t = computed(() => ({ hour: 'Hour', minute: 'Minute', second: 'Second', dayPeriod: 'AM/PM', empty: 'Empty', start: 'Start time', end: 'End time', ...props.labels }))
const group = useTemplateRef<HTMLElement>('group')

// The bound times: one, or a range's start and end.
const asRange = () => (model.value && typeof model.value === 'object' && 'start' in model.value ? (model.value as TimeRange) : null)
const bound = (which: 'start' | 'end') => toTime(props.range ? asRange()?.[which] : which === 'start' ? model.value : null)
const iso = (time: Time | null) => fromTime(time, props.granularity, 'iso') as string | null
const out = (time: Time | null) => fromTime(time, props.granularity, props.valueFormat) as TimeInput | null
function commit(which: 'start' | 'end', time: Time | null) {
  if (iso(time) === iso(bound(which))) return
  let value: TimeInput | TimeRange | null
  if (!props.range) value = out(time)
  else {
    const start = which === 'start' ? time : bound('start')
    const end = which === 'end' ? time : bound('end')
    value = start || end ? { start: out(start), end: out(end) } : null
  }
  model.value = value
  emit('change', value)
  f.input()
}
const partsFor = (which: 'start' | 'end') =>
  useTimeParts({
    locale,
    cycle,
    granularity: () => props.granularity,
    step: () => props.step,
    labels: () => t.value,
    value: () => bound(which),
    onChange: (time) => commit(which, time),
    editable: () => !props.readonly,
    disabled: () => f.disabled.value,
    group: () => group.value,
  })
const startParts = partsFor('start')
const endParts = partsFor('end')
const fields = computed(() =>
  props.range
    ? [
        { key: 'start', label: t.value.start, parts: startParts, value: bound('start') },
        { key: 'end', label: t.value.end, parts: endParts, value: bound('end') },
      ]
    : [{ key: 'time', label: undefined, parts: startParts, value: bound('start') }],
)

const min = computed(() => toTime(props.min))
const max = computed(() => toTime(props.max))
const outOfRange = (time: Time | null) => !!time && ((!!min.value && time.compare(min.value) < 0) || (!!max.value && time.compare(max.value) > 0))
const invalid = computed(() => {
  const start = bound('start')
  const end = bound('end')
  return f.invalid.value || outOfRange(start) || outOfRange(end) || (!!start && !!end && end.compare(start) < 0)
})

// Touch: the platform's time field.
const nativeStep = computed(() => {
  if (props.granularity === 'second') return props.step?.second ?? 1
  if (props.granularity === 'hour') return 3600
  const minutes = props.step?.minute ?? 1
  return minutes === 1 ? undefined : minutes * 60
})
const nativeGranularity = computed<Granularity>(() => (props.granularity === 'second' ? 'second' : 'minute'))

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
  startParts.blur()
  endParts.blur()
  f.blur()
  emit('blur', event)
}
function onFocusin(event: FocusEvent) {
  const from = event.relatedTarget as Node | null
  if (!from || !group.value?.contains(from)) emit('focus', event)
}
const hidden = (which: 'start' | 'end') => iso(bound(which)) ?? ''
</script>

<template>
  <span
    ref="group"
    class="weft-input weft-input-time"
    v-bind="{ ...box, ...(touch ? {} : own) }"
    :id="touch && !range ? undefined : f.id.value"
    :role="touch && !range ? undefined : 'group'"
    :aria-labelledby="touch && !range ? undefined : f.field?.labelId"
    :aria-describedby="touch && !range ? undefined : describedBy()"
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
    <template v-if="touch">
      <template v-for="(fd, i) in fields" :key="fd.key">
        <span v-if="i > 0" class="weft-input-time__separator" aria-hidden="true">–</span>
        <input
          class="weft-input__field"
          type="time"
          v-bind="range ? {} : own"
          :id="range ? undefined : f.id.value"
          :value="fd.parts.native.value"
          :min="min ? fromTime(min, nativeGranularity, 'iso') as string : undefined"
          :max="max ? fromTime(max, nativeGranularity, 'iso') as string : undefined"
          :step="nativeStep"
          :disabled="f.disabled.value || undefined"
          :readonly="readonly || undefined"
          :required="required || undefined"
          :aria-label="fd.label"
          :aria-labelledby="fd.label ? undefined : f.field?.labelId"
          :aria-describedby="range ? undefined : describedBy()"
          :aria-invalid="invalid ? 'true' : undefined"
          @input="fd.parts.input($event)"
        />
      </template>
    </template>
    <span v-else class="weft-input-time__segments">
      <template v-for="(fd, i) in fields" :key="fd.key">
        <span v-if="i > 0" class="weft-input-time__separator" aria-hidden="true">–</span>
        <span class="weft-input-time__field" :role="fd.label ? 'group' : undefined" :aria-label="fd.label">
          <template v-for="s in fd.parts.segments.value" :key="s.key">
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
              @keydown="fd.parts.keydown($event, s.type)"
              @focus="fd.parts.focus()"
            >{{ s.text }}</span>
          </template>
        </span>
      </template>
    </span>
    <template v-if="f.name.value">
      <template v-if="range">
        <input type="hidden" :name="`${f.name.value}[start]`" :value="hidden('start')" :disabled="f.disabled.value || undefined" />
        <input type="hidden" :name="`${f.name.value}[end]`" :value="hidden('end')" :disabled="f.disabled.value || undefined" />
      </template>
      <input v-else type="hidden" :name="f.name.value" :value="hidden('start')" :disabled="f.disabled.value || undefined" />
    </template>
  </span>
</template>
