<script setup vapor lang="ts">
// A field for a range of dates, a start and an end (the WAI-ARIA APG Date
// Picker Dialog, with a range calendar). On pointer it is two text fields,
// where dates are typed as the locale writes them (an end before the start
// is refused), and a button opening a calendar of two months beside them,
// as does Alt+ArrowDown in either field: a press (or Enter) on a date
// starts the range, a second ends it and closes the calendar, focus
// returning to where it was opened from. On touch the whole field is one
// button showing the range, which opens one month in a modal bottom sheet,
// 44 px dates, a swipe changing the month. The input can change while it is
// open: the same calendar stays, with a half-chosen range, the focused date
// and focus. It binds `{ start, end }` (ISO dates, or CalendarDates with
// value-format="object"), null when both are empty, and submits the ISO
// dates as name[start] and name[end].
import { computed, nextTick, shallowRef, useTemplateRef, watch } from 'vue'
import Icon from '../basics/Icon.vue'
import { useId } from '../headless/useId.ts'
import { useControllable } from '../headless/useControllable.ts'
import type { Placement } from '../headless/useFloating.ts'
import { useControlAttrs, useField } from '../forms/field.ts'
import { joinIds, pressFocuses } from '../forms/dom.ts'
import Popover from '../overlays/Popover.vue'
import Calendar from './Calendar.vue'
import { formatter, fromDate, pattern, toDate, utc, type CalendarDate, type DateInput, type DateRange, type ValueFormat } from './dates.ts'
import { cycleTab, keepFocusOnSwitch, usePicker, useTypedDate, type DateRangePickerLabels } from './picker.ts'
import type { Size } from '../types.ts'
import '../styles/base.css'
import '../styles/input.css'
import '../styles/date-picker.css'

defineOptions({ inheritAttrs: false })


const props = withDefaults(
  defineProps<{
    /** The range, bound with `v-model`: `{ start, end }`, each an ISO string (or a CalendarDate with `value-format="object"`) or null; null when both are empty. */
    modelValue?: DateRange | null
    /** The range it starts with when `v-model` is not bound. */
    defaultValue?: DateRange | null
    /** `iso`: dates are bound as `YYYY-MM-DD` strings. `object`: as `CalendarDate`s. */
    valueFormat?: ValueFormat
    /** The BCP 47 locale the dates are written and read in; a WeftProvider's applies when not set, else en-US. */
    locale?: string
    /** How the fields write a date (Intl options); numeric by default (21/11/2026), so it reads back as typed. */
    format?: Intl.DateTimeFormatOptions
    /** Shown in an empty field; the locale's pattern (dd/mm/yyyy) when not set. */
    placeholder?: string
    /** The earliest date allowed. */
    min?: DateInput
    /** The latest date allowed. */
    max?: DateInput
    /** Dates that cannot be chosen, nor be inside a range. */
    isDateDisabled?: (date: CalendarDate) => boolean
    /** Dates shown struck through, which cannot be chosen either. */
    isDateUnavailable?: (date: CalendarDate) => boolean
    /** The calendar's first day of the week, 0 Sunday to 6 Saturday; the locale's when not set. */
    weekStartsOn?: 0 | 1 | 2 | 3 | 4 | 5 | 6
    /** Months shown side by side; by default two on pointer and one on touch. */
    numberOfMonths?: number
    /** `sm`, `md` or `lg`; a FieldGroup's or FormField's size applies when not set. md is 44 px tall on touch. */
    size?: Size
    /** A Lucide icon before the dates. The leading slot replaces it. */
    icon?: string
    /** Unavailable: nothing can be typed or opened. */
    disabled?: boolean
    /** Shown but not editable: nothing can be typed or opened. */
    readonly?: boolean
    /** Both dates are required (native `required` on the fields, aria-required). */
    required?: boolean
    /** The form field name: the dates submit as `name[start]` and `name[end]`; a FormField's name applies when not set. */
    name?: string
    /** The start field's id; inside a FormField the field's id is used, so its label points here. */
    id?: string
    /** The calendar open or closed, bound with `v-model:open`. */
    open?: boolean
    /** Whether it starts open when `v-model:open` is not bound. */
    defaultOpen?: boolean
    /** Where the calendar floats on pointer (start and end follow the text direction). */
    placement?: Placement
    /** Close the calendar when a range is chosen. */
    closeOnSelect?: boolean
    /** The words it says, to translate them. */
    labels?: Partial<DateRangePickerLabels>
  }>(),
  {
    modelValue: undefined,
    defaultValue: null,
    valueFormat: 'iso',
    open: undefined,
    defaultOpen: false,
    placement: 'bottom-start',
    closeOnSelect: true,
  },
)

const emit = defineEmits<{
  /** The range changed, by typing or by the calendar. */
  'update:modelValue': [value: DateRange | null]
  /** The calendar opened or closed. */
  'update:open': [open: boolean]
  /** The range was committed (a date typed and left, or a range chosen). */
  change: [value: DateRange | null]
  /** Focus left the fields, the button and the calendar. */
  blur: [event: FocusEvent]
  /** A field or the button took focus. */
  focus: [event: FocusEvent]
}>()

defineSlots<{
  /** Before the dates, in place of `icon`. */
  leading?: () => unknown
  /** A date's content in the calendar, in place of its number. */
  day?: (props: { date: CalendarDate; selected: boolean; disabled: boolean; today: boolean; outside: boolean }) => unknown
}>()

const f = useField(props)
const { own, box, describedBy } = useControlAttrs(f)
const model = useControllable(props, 'modelValue', emit, () => props.defaultValue)
const picker = usePicker(props, emit, f)
const { open, touch, locale, blocked } = picker
const startDate = computed(() => toDate(model.value?.start))
const endDate = computed(() => toDate(model.value?.end))
const min = computed(() => toDate(props.min))
const max = computed(() => toDate(props.max))
const uid = useId('date-range')
const ids = { value: `${uid}-value`, start: `${uid}-start`, end: `${uid}-end` }
const boxEl = useTemplateRef<HTMLElement>('boxEl')
const startInput = useTemplateRef<HTMLInputElement>('startInput')
const endInput = useTemplateRef<HTMLInputElement>('endInput')
const button = useTemplateRef<HTMLButtonElement>('button')
const t = computed(() => ({ choose: 'Choose dates', start: 'Start date', end: 'End date', ...props.labels }))

const fieldFormat = () => props.format ?? { year: 'numeric', month: '2-digit', day: '2-digit' }
const allowed = (d: CalendarDate) =>
  !(min.value && d.compare(min.value) < 0) && !(max.value && d.compare(max.value) > 0) && !props.isDateDisabled?.(d) && !props.isDateUnavailable?.(d)

function set(start: CalendarDate | null, end: CalendarDate | null) {
  const same = (a: CalendarDate | null, b: CalendarDate | null) => (a && b ? a.compare(b) === 0 : a === b)
  if (same(start, startDate.value) && same(end, endDate.value)) return
  const out = (d: CalendarDate | null) => fromDate(d, props.valueFormat) as DateInput | null
  const value = start || end ? { start: out(start), end: out(end) } : null
  model.value = value
  emit('change', value)
  f.input()
}
const startText = useTypedDate({
  locale,
  value: () => startDate.value,
  format: fieldFormat,
  allowed: (d) => allowed(d) && !(endDate.value && d.compare(endDate.value) > 0),
  onCommit: (d) => set(d, endDate.value),
})
const endText = useTypedDate({
  locale,
  value: () => endDate.value,
  format: fieldFormat,
  allowed: (d) => allowed(d) && !(startDate.value && d.compare(startDate.value) < 0),
  onCommit: (d) => set(startDate.value, d),
})
const invalid = computed(() => startText.invalid.value || endText.invalid.value || f.invalid.value)
const placeholderText = computed(() => props.placeholder ?? pattern(locale.value))
const shown = computed(() =>
  startText.text.value || endText.text.value ? `${startText.text.value || '…'} – ${endText.text.value || '…'}` : `${placeholderText.value} – ${placeholderText.value}`,
)
const buttonLabel = computed(() => {
  const s = startDate.value
  const e = endDate.value
  if (!s || !e) return t.value.choose
  return `${t.value.choose}, ${formatter(locale.value, { year: 'numeric', month: 'long', day: 'numeric' }).formatRange(utc(s), utc(e))}`
})
const dialogTitle = computed(() => (open.value ? picker.title(t.value.choose) : undefined))
const months = computed(() => props.numberOfMonths ?? (touch.value ? 1 : 2))
/** A name for a field: the FormField's label and which end it is. */
const nameFor = (word: string) => (f.field ? `${f.field.labelId} ${word}` : undefined)

let openOn: CalendarDate | null = null
const startOn = computed(() => (open.value ? openOn?.toString() : undefined))
function show() {
  openOn = startText.typed() ?? endText.typed()
  picker.show()
}
function toggle() {
  if (open.value) picker.hide()
  else show()
}
function onOpenChange(on: boolean) {
  if (on) show()
  else picker.hide()
}
function pick(value: unknown) {
  const r = value as { start: unknown; end: unknown } | null
  set(toDate(r?.start), toDate(r?.end))
  if (props.closeOnSelect) picker.hide()
}
const calendarValue = computed(() => (startDate.value || endDate.value ? { start: startDate.value, end: endDate.value } : null))

function onFieldKeydown(event: KeyboardEvent, which: typeof startText) {
  if (event.key === 'Enter') which.commit()
  else if (event.key === 'ArrowDown' && event.altKey) {
    event.preventDefault()
    show()
  }
}
function onButtonDown(event: MouseEvent) {
  if (open.value && !touch.value) event.preventDefault()
}
const panel = () => (typeof document === 'undefined' ? null : document.querySelector(`[data-weft-picker="${uid}"]`)?.closest<HTMLElement>('.weft-popover'))
const panelId = shallowRef<string>()
watch(open, (on) => {
  panelId.value = undefined
  if (on) void nextTick(() => (panelId.value = panel()?.id))
})
function onFocusout(event: FocusEvent) {
  picker.focusout(event, () => boxEl.value, panel, () => {
    f.blur()
    emit('blur', event)
  })
}
function onFocusin(event: FocusEvent) {
  const from = event.relatedTarget as Node | null
  if (!from || !(boxEl.value?.contains(from) || panel()?.contains(from))) emit('focus', event)
}
keepFocusOnSwitch(touch, () => [startInput.value, endInput.value], () => button.value)

defineExpose({
  /** Focus the start field (the button on touch). */
  focus: (options?: FocusOptions) => (touch.value ? button.value : startInput.value)?.focus(options),
})
</script>

<template>
  <span
    ref="boxEl"
    class="weft-input weft-date-picker weft-date-range-picker"
    v-bind="box"
    :data-form="touch ? 'button' : 'field'"
    :data-size="f.size.value"
    :data-invalid="invalid ? 'true' : undefined"
    :data-disabled="f.disabled.value ? 'true' : undefined"
    :data-open="open ? 'true' : undefined"
    @mousedown="!touch && pressFocuses($event, startInput)"
    @focusout="onFocusout"
    @focusin="onFocusin"
  >
    <span v-if="$slots.leading || icon" class="weft-input__leading">
      <slot name="leading"><Icon v-if="icon" :name="icon" /></slot>
    </span>
    <span :id="ids.start" class="weft-visually-hidden">{{ t.start }}</span>
    <span :id="ids.end" class="weft-visually-hidden">{{ t.end }}</span>
    <template v-if="!touch">
      <input
        ref="startInput"
        class="weft-input__field"
        type="text"
        autocomplete="off"
        v-bind="own"
        :id="f.id.value"
        :value="startText.text.value"
        :placeholder="placeholderText"
        :disabled="f.disabled.value || undefined"
        :readonly="readonly || undefined"
        :required="required || undefined"
        :aria-label="f.field ? undefined : t.start"
        :aria-labelledby="nameFor(ids.start)"
        :aria-describedby="describedBy()"
        :aria-invalid="startText.invalid.value || f.invalid.value ? 'true' : undefined"
        :aria-required="f.aria.value['aria-required']"
        @input="startText.input(($event.target as HTMLInputElement).value)"
        @change="startText.commit()"
        @keydown="onFieldKeydown($event, startText)"
      />
      <span class="weft-date-picker__range-separator" aria-hidden="true">–</span>
      <input
        ref="endInput"
        class="weft-input__field"
        type="text"
        autocomplete="off"
        :value="endText.text.value"
        :placeholder="placeholderText"
        :disabled="f.disabled.value || undefined"
        :readonly="readonly || undefined"
        :required="required || undefined"
        :aria-label="f.field ? undefined : t.end"
        :aria-labelledby="nameFor(ids.end)"
        :aria-describedby="describedBy()"
        :aria-invalid="endText.invalid.value || f.invalid.value ? 'true' : undefined"
        :aria-required="f.aria.value['aria-required']"
        @input="endText.input(($event.target as HTMLInputElement).value)"
        @change="endText.commit()"
        @keydown="onFieldKeydown($event, endText)"
      />
    </template>
    <button
      ref="button"
      type="button"
      :class="touch ? 'weft-date-picker__trigger' : 'weft-date-picker__button'"
      v-bind="touch ? own : {}"
      :id="touch ? f.id.value : undefined"
      aria-haspopup="dialog"
      :aria-expanded="open ? 'true' : 'false'"
      :aria-controls="open ? panelId : undefined"
      :aria-label="touch ? (own['aria-label'] as string | undefined) : buttonLabel"
      :aria-labelledby="touch && f.field ? joinIds(f.field.labelId, ids.value) : undefined"
      :aria-describedby="touch ? describedBy() : undefined"
      :aria-invalid="touch && invalid ? 'true' : undefined"
      :aria-required="touch && (required || f.aria.value['aria-required']) ? 'true' : undefined"
      :disabled="blocked || undefined"
      @mousedown="onButtonDown"
      @click="toggle"
    >
      <span v-if="touch" :id="ids.value" class="weft-date-picker__value" :data-placeholder="startText.text.value || endText.text.value ? undefined : 'true'">{{ shown }}</span>
      <svg class="weft-date-picker__glyph" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M8 2v4" /><path d="M16 2v4" /><rect width="18" height="18" x="3" y="4" rx="2" /><path d="M3 10h18" /></svg>
    </button>
    <template v-if="f.name.value">
      <input type="hidden" :name="`${f.name.value}[start]`" :value="startDate?.toString() ?? ''" :disabled="f.disabled.value || undefined" />
      <input type="hidden" :name="`${f.name.value}[end]`" :value="endDate?.toString() ?? ''" :disabled="f.disabled.value || undefined" />
    </template>
    <Popover
      :open="open"
      :anchor="boxEl ?? undefined"
      :title="dialogTitle"
      :placement="placement"
      :offset="4"
      class="weft-date-picker__popup"
      @update:open="onOpenChange"
    >
      <template #content>
        <Calendar
          :data-weft-picker="uid"
          range
          :model-value="calendarValue"
          value-format="object"
          :focused-date="startOn"
          :number-of-months="months"
          :locale="locale"
          :min="min ?? undefined"
          :max="max ?? undefined"
          :is-date-disabled="isDateDisabled"
          :is-date-unavailable="isDateUnavailable"
          :week-starts-on="weekStartsOn"
          :size="f.size.value"
          :labels="labels"
          @update:model-value="pick"
          @focusout="onFocusout"
          @keydown="cycleTab($event, panel())"
        >
          <template v-if="$slots.day" #day="p"><slot name="day" v-bind="p" /></template>
        </Calendar>
      </template>
    </Popover>
  </span>
</template>
