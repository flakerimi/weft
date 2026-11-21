<script setup vapor lang="ts">
// A date field (the WAI-ARIA APG Date Picker Dialog). On pointer it is a
// text field, where a date is typed as the locale writes it (read on change
// and Enter: 21/11/2026 in en-GB, 11/21/2026 in en-US, "21 Nov 2026", ISO),
// and a button that opens a calendar beside it, as does Alt+ArrowDown in
// the field. The calendar is a non-modal dialog: focus goes to the chosen
// date (or the date typed, or today), choosing one closes it and focus
// returns to where it was opened from; Escape closes it too, and focus
// leaving it. On touch the whole field is one button showing the date,
// which opens the calendar in a modal bottom sheet titled with the field's
// label, its dates 44 px a side and a swipe changing the month. The input
// can change while it is open: the same calendar stays, with its chosen
// and focused date and focus, and the button is the same element in both
// forms. It binds an ISO date (YYYY-MM-DD) or, with value-format="object",
// a CalendarDate, and submits the ISO date under its name.
//
// What it draws only while open, the popover and the calendar in it, is
// code of its own, loaded when the picker is about to open: when the pointer
// or focus comes to it (useLazyPanel). The popover opens once it is in, so
// focus lands on the calendar's date as ever.
import { computed, nextTick, shallowRef, useTemplateRef, watch } from 'vue'
import Icon from '../basics/Icon.vue'
import { useId } from '../headless/useId.ts'
import { useControllable } from '../headless/useControllable.ts'
import type { Placement } from '../headless/useFloating.ts'
import { useControlAttrs, useField } from '../forms/field.ts'
import { pressFocuses } from '../forms/dom.ts'
import { format, fromDate, pattern, toDate, type CalendarDate, type DateInput, type ValueFormat } from './dates.ts'
import { cycleTab, keepFocusOnSwitch, openingDay, useFormReset, useLazyPanel, usePicker, useTypedDate, type DatePickerLabels } from './picker.ts'
import type { Size } from '../types.ts'
import '../styles/base.css'
import '../styles/input.css'
import '../styles/date-picker.css'

defineOptions({ inheritAttrs: false })


const props = withDefaults(
  defineProps<{
    /** The date, bound with `v-model`: an ISO string (`2026-11-21`), or a CalendarDate with `value-format="object"`; null when empty. */
    modelValue?: DateInput | null
    /** The date it starts with when `v-model` is not bound. */
    defaultValue?: DateInput | null
    /** `iso`: the date is bound as a `YYYY-MM-DD` string. `object`: as a `CalendarDate`. */
    valueFormat?: ValueFormat
    /** The BCP 47 locale the date is written and read in; a WeftProvider's applies when not set, else en-US. */
    locale?: string
    /** How the field writes the date (Intl options); numeric by default (21/11/2026), so it reads back as typed. */
    format?: Intl.DateTimeFormatOptions
    /** Shown while it is empty; the locale's pattern (dd/mm/yyyy) when not set. */
    placeholder?: string
    /** The earliest date allowed. */
    min?: DateInput
    /** The latest date allowed. */
    max?: DateInput
    /** Dates that cannot be chosen. */
    isDateDisabled?: (date: CalendarDate) => boolean
    /** Dates shown struck through, which cannot be chosen either. */
    isDateUnavailable?: (date: CalendarDate) => boolean
    /** The calendar's first day of the week, 0 Sunday to 6 Saturday; the locale's when not set. */
    weekStartsOn?: 0 | 1 | 2 | 3 | 4 | 5 | 6
    /** ISO week numbers in the calendar. */
    weekNumbers?: boolean
    /** `sm`, `md` or `lg`; a FieldGroup's or FormField's size applies when not set. md is 44 px tall on touch. */
    size?: Size
    /** A Lucide icon before the date. The leading slot replaces it. */
    icon?: string
    /** Unavailable: nothing can be typed or opened. */
    disabled?: boolean
    /** Shown but not editable: nothing can be typed or opened. */
    readonly?: boolean
    /** A date is required (native `required` on the field, aria-required). */
    required?: boolean
    /** The form field name its ISO date submits under; a FormField's name applies when not set. */
    name?: string
    /** The field's id; inside a FormField the field's id is used, so its label points here. */
    id?: string
    /** The calendar open or closed, bound with `v-model:open`. */
    open?: boolean
    /** Whether it starts open when `v-model:open` is not bound. */
    defaultOpen?: boolean
    /** Where the calendar floats on pointer (start and end follow the text direction). */
    placement?: Placement
    /** Close the calendar when a date is chosen. */
    closeOnSelect?: boolean
    /** The words it says, to translate them. */
    labels?: Partial<DatePickerLabels>
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
  /** The date changed, by typing or by the calendar: ISO, a CalendarDate, or null. */
  'update:modelValue': [value: DateInput | null]
  /** The calendar opened or closed. */
  'update:open': [open: boolean]
  /** The date was committed (typed and left, or chosen). */
  change: [value: DateInput | null]
  /** Focus left the field, its button and its calendar. */
  blur: [event: FocusEvent]
  /** The field or its button took focus. */
  focus: [event: FocusEvent]
}>()

defineSlots<{
  /** Before the date, in place of `icon`. */
  leading?: () => unknown
  /** A date's content in the calendar, in place of its number. */
  day?: (props: { date: CalendarDate; selected: boolean; disabled: boolean; today: boolean; outside: boolean }) => unknown
}>()

const f = useField(props)
const { own, box, describedBy } = useControlAttrs(f)
const model = useControllable(props, 'modelValue', emit, () => props.defaultValue)
const picker = usePicker(props, emit, f)
const { open, touch, locale, blocked } = picker
const lazy = useLazyPanel(open)
/** Whether the popover is open: once it is loaded (at once after the first time). */
const shown = computed(() => open.value && lazy.ready.value)
const date = computed(() => toDate(model.value))
const min = computed(() => toDate(props.min))
const max = computed(() => toDate(props.max))
const valueId = useId('date-picker-value')
const boxEl = useTemplateRef<HTMLElement>('boxEl')
const input = useTemplateRef<HTMLInputElement>('input')
const button = useTemplateRef<HTMLButtonElement>('button')

const fieldFormat = () => props.format ?? { year: 'numeric', month: '2-digit', day: '2-digit' }
const allowed = (d: CalendarDate) =>
  !(min.value && d.compare(min.value) < 0) && !(max.value && d.compare(max.value) > 0) && !props.isDateDisabled?.(d) && !props.isDateUnavailable?.(d)

function set(d: CalendarDate | null) {
  const before = date.value
  if (before === d || (before && d && before.compare(d) === 0)) return
  const value = fromDate(d, props.valueFormat) as DateInput | null
  model.value = value
  emit('change', value)
  f.input()
}
const typed = useTypedDate({ locale, value: () => date.value, format: fieldFormat, allowed, onCommit: set })
const invalid = computed(() => typed.invalid.value || f.invalid.value)

const placeholderText = computed(() => props.placeholder ?? pattern(locale.value))
const t = computed(() => ({ choose: 'Choose date', ...props.labels }))
const buttonLabel = computed(() =>
  date.value ? `${t.value.choose}, ${format(date.value, locale.value, { year: 'numeric', month: 'long', day: 'numeric' })}` : t.value.choose,
)
const dialogTitle = computed(() => (open.value ? picker.title(t.value.choose) : undefined))

/** The date the calendar opens on: the date typed, else the bound one. */
let openOn: CalendarDate | null = null
const startOn = computed(() => (open.value ? openOn?.toString() : undefined))
function show() {
  openOn = typed.typed() ?? openingDay()
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
  set(toDate(value))
  if (props.closeOnSelect) picker.hide()
}

function onFieldKeydown(event: KeyboardEvent) {
  if (event.key === 'Enter') typed.commit()
  else if (event.key === 'ArrowDown' && event.altKey) {
    event.preventDefault()
    show()
  }
}
/** While the calendar is open, a press on the button keeps focus in it, so the press closes it. */
function onButtonDown(event: MouseEvent) {
  if (open.value && !touch.value) event.preventDefault()
}
/** The calendar's dialog, while it is open. */
const panel = () => (typeof document === 'undefined' ? null : document.querySelector(`[data-weft-picker="${valueId}"]`)?.closest<HTMLElement>('.weft-popover'))
const panelId = shallowRef<string>()
watch(shown, (on) => {
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
  lazy.load()
  const from = event.relatedTarget as Node | null
  if (!from || !(boxEl.value?.contains(from) || panel()?.contains(from))) emit('focus', event)
}
keepFocusOnSwitch(touch, () => [input.value], () => button.value)
useFormReset(() => boxEl.value, () => [{ typed, el: input.value }])

defineExpose({
  /** Focus the field (the button on touch). */
  focus: (options?: FocusOptions) => (touch.value ? button.value : input.value)?.focus(options),
})
</script>

<template>
  <span
    ref="boxEl"
    class="weft-input weft-date-picker"
    v-bind="box"
    :data-form="touch ? 'button' : 'field'"
    :data-size="f.size.value"
    :data-invalid="invalid ? 'true' : undefined"
    :data-disabled="f.disabled.value ? 'true' : undefined"
    :data-open="open ? 'true' : undefined"
    @mousedown="!touch && pressFocuses($event, input)"
    @focusout="onFocusout"
    @focusin="onFocusin"
    @pointerenter="lazy.load"
  >
    <span v-if="$slots.leading || icon" class="weft-input__leading">
      <slot name="leading"><Icon v-if="icon" :name="icon" /></slot>
    </span>
    <input
      v-if="!touch"
      ref="input"
      class="weft-input__field"
      type="text"
      autocomplete="off"
      v-bind="own"
      :id="f.id.value"
      :value="typed.text.value"
      :placeholder="placeholderText"
      :disabled="f.disabled.value || undefined"
      :readonly="readonly || undefined"
      :required="required || undefined"
      :aria-describedby="describedBy()"
      :aria-invalid="invalid ? 'true' : undefined"
      :aria-required="f.aria.value['aria-required']"
      @input="typed.input(($event.target as HTMLInputElement).value)"
      @change="typed.commit()"
      @keydown="onFieldKeydown"
    />
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
      :aria-labelledby="touch && f.field ? `${f.field.labelId} ${valueId}` : undefined"
      :aria-describedby="touch ? describedBy() : undefined"
      :aria-invalid="touch && invalid ? 'true' : undefined"
      :aria-required="touch && (required || f.aria.value['aria-required']) ? 'true' : undefined"
      :disabled="blocked || undefined"
      @mousedown="onButtonDown"
      @click="toggle"
    >
      <span v-if="touch" :id="valueId" class="weft-date-picker__value" :data-placeholder="typed.text.value ? undefined : 'true'">{{ typed.text.value || placeholderText }}</span>
      <svg class="weft-date-picker__glyph" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M8 2v4" /><path d="M16 2v4" /><rect width="18" height="18" x="3" y="4" rx="2" /><path d="M3 10h18" /></svg>
    </button>
    <input v-if="f.name.value" type="hidden" :name="f.name.value" :value="date?.toString() ?? ''" :disabled="f.disabled.value || undefined" />
    <component
      :is="lazy.view.value.Popover"
      v-if="lazy.view.value"
      :open="shown"
      :anchor="boxEl ?? undefined"
      :title="dialogTitle"
      :placement="placement"
      :offset="4"
      class="weft-date-picker__popup"
      @update:open="onOpenChange"
    >
      <template #content>
        <component
          :is="lazy.view.value.Calendar"
          :data-weft-picker="valueId"
          :model-value="date"
          value-format="object"
          :focused-date="startOn"
          :locale="locale"
          :min="min ?? undefined"
          :max="max ?? undefined"
          :is-date-disabled="isDateDisabled"
          :is-date-unavailable="isDateUnavailable"
          :week-starts-on="weekStartsOn"
          :week-numbers="weekNumbers"
          :size="f.size.value"
          :labels="labels"
          @update:model-value="pick"
          @focusout="onFocusout"
          @keydown="cycleTab($event, panel())"
        >
          <template v-if="$slots.day" #day="p"><slot name="day" v-bind="p" /></template>
        </component>
      </template>
    </component>
  </span>
</template>
