<script setup vapor lang="ts">
// A number, typed or stepped: the WAI-ARIA spinbutton pattern on a text input
// (inputmode decimal), between a decrease and an increase button. Up and Down
// step by `step`, Page Up and Page Down by ten steps, Home and End go to min
// and max. Typed text is read when it is committed (Enter, leaving the
// field, or a change event, as when it was left before the page was live)
// and kept within min and max; text that is not a number goes back to
// the value. The mouse wheel changes it only with `wheel`, and only while it
// has focus. The steppers stay out of the tab order (the keys do their work)
// and keep focus where it is; on touch they are --weft-hit-min (44 px) wide.
import { computed, ref, useTemplateRef, watch } from 'vue'
import { useControllable } from '../headless/useControllable.ts'
import { useId } from '../headless/useId.ts'
import { useControlAttrs, useField } from './field.ts'
import { pressFocuses } from './dom.ts'
import { clamp, tidy } from './numbers.ts'
import type { Size } from '../types.ts'
import '../styles/base.css'
import '../styles/input.css'
import '../styles/input-number.css'

defineOptions({ inheritAttrs: false })

const props = withDefaults(
  defineProps<{
    /** The number, bound with `v-model`; null when it is empty. */
    modelValue?: number | null
    /** The number to start with when `v-model` is not bound. */
    defaultValue?: number | null
    /** The smallest it may be. */
    min?: number
    /** The largest it may be. */
    max?: number
    /** How much a step (an arrow key or a stepper) changes it. */
    step?: number
    /** Change it with the mouse wheel while it has focus. Off by default, so scrolling the page never changes a value. */
    wheel?: boolean
    /** `sm`, `md` or `lg`; a FieldGroup's size applies when not set. md is 44 px tall on touch, as Button is. */
    size?: Size
    /** Shown while it is empty; not a label. */
    placeholder?: string
    /** The form field name; a FormField's name applies when not set. */
    name?: string
    /** The input's id; inside a FormField the field's id is used. */
    id?: string
    /** The decrease button's name for assistive technology. */
    decrementLabel?: string
    /** The increase button's name for assistive technology. */
    incrementLabel?: string
    /** Unavailable: natively disabled, steppers too. */
    disabled?: boolean
    /** Shown but not changeable. */
    readonly?: boolean
    /** The native `required`. */
    required?: boolean
  }>(),
  { modelValue: undefined, defaultValue: null, step: 1, decrementLabel: 'Decrease', incrementLabel: 'Increase' },
)

const emit = defineEmits<{
  /** The number changed: stepped, or typed and committed. */
  'update:modelValue': [value: number | null]
  /** The input lost focus. */
  blur: [event: FocusEvent]
  /** The input took focus. */
  focus: [event: FocusEvent]
}>()

const f = useField(props)
const { own, box, describedBy } = useControlAttrs(f)
const id = f.id.value ?? (f.name.value ? undefined : useId('input-number'))
const model = useControllable(props, 'modelValue', emit, () => props.defaultValue)
const input = useTemplateRef<HTMLInputElement>('input')

const show = (n: number | null) => (n === null || Number.isNaN(n) ? '' : String(n))
const text = ref(show(model.value))
/** Show `t`, in the element at once (a stepped value reads back before Vue's next render). */
function showText(t: string) {
  text.value = t
  if (input.value) input.value.value = t
}
watch(model, (n) => showText(show(n)))

const locked = computed(() => f.disabled.value || props.readonly)
const atMin = computed(() => props.min !== undefined && model.value !== null && model.value <= props.min)
const atMax = computed(() => props.max !== undefined && model.value !== null && model.value >= props.max)

function set(n: number | null) {
  model.value = n === null ? null : clamp(n, props.min, props.max)
  showText(show(model.value))
  f.input()
}
/** Move by `count` steps; from empty, start at min (or 0). */
function stepBy(count: number) {
  if (locked.value) return
  // Text typed and not yet committed is the value to step from.
  if (text.value !== show(model.value)) commit()
  const from = model.value
  set(from === null ? (props.min ?? 0) : tidy(from + count * props.step, from, props.step))
}

function commit() {
  const raw = text.value.trim().replace(',', '.')
  if (raw === '') return set(null)
  const n = Number(raw)
  if (Number.isFinite(n)) set(n)
  else showText(show(model.value))
}

function onKeydown(event: KeyboardEvent) {
  if (locked.value || event.altKey || event.ctrlKey || event.metaKey) return
  const moves: Record<string, () => void> = {
    ArrowUp: () => stepBy(1),
    ArrowDown: () => stepBy(-1),
    PageUp: () => stepBy(10),
    PageDown: () => stepBy(-10),
    Home: () => props.min !== undefined && set(props.min),
    End: () => props.max !== undefined && set(props.max),
    Enter: commit,
  }
  const move = moves[event.key]
  if (!move || ((event.key === 'Home' && props.min === undefined) || (event.key === 'End' && props.max === undefined))) return
  // Enter commits but still submits the form, as it would from a text field.
  if (event.key !== 'Enter') event.preventDefault()
  move()
}
function onWheel(event: WheelEvent) {
  if (!props.wheel || locked.value || document.activeElement !== input.value || event.deltaY === 0) return
  event.preventDefault()
  stepBy(event.deltaY < 0 ? 1 : -1)
}
function onBlur(event: FocusEvent) {
  commit()
  f.blur()
  emit('blur', event)
}

defineExpose({
  /** Focus the number. */
  focus: (options?: FocusOptions) => input.value?.focus(options),
  /** The <input> element. */
  input,
})
</script>

<template>
  <span
    class="weft-input weft-input-number"
    v-bind="box"
    :data-size="f.size.value"
    :data-invalid="f.invalid.value ? 'true' : undefined"
    :data-disabled="f.disabled.value ? 'true' : undefined"
    @mousedown="pressFocuses($event, input)"
  >
    <button
      class="weft-input-number__step"
      type="button"
      tabindex="-1"
      data-step="down"
      :aria-label="decrementLabel"
      :disabled="locked || atMin || undefined"
      @mousedown.prevent
      @click="stepBy(-1)"
    >
      <span class="weft-input-number__glyph" aria-hidden="true"></span>
    </button>
    <input
      ref="input"
      class="weft-input__field weft-input-number__field"
      type="text"
      inputmode="decimal"
      role="spinbutton"
      autocomplete="off"
      v-bind="own"
      :id="id"
      :name="f.name.value"
      :value="text"
      :placeholder="placeholder"
      :disabled="f.disabled.value || undefined"
      :readonly="readonly || undefined"
      :required="required || undefined"
      :aria-valuenow="model ?? undefined"
      :aria-valuemin="min"
      :aria-valuemax="max"
      :aria-describedby="describedBy()"
      :aria-invalid="f.aria.value['aria-invalid']"
      :aria-required="f.aria.value['aria-required']"
      @input="text = ($event.target as HTMLInputElement).value"
      @change="commit()"
      @keydown="onKeydown"
      @wheel="onWheel"
      @blur="onBlur"
      @focus="emit('focus', $event)"
    />
    <button
      class="weft-input-number__step"
      type="button"
      tabindex="-1"
      data-step="up"
      :aria-label="incrementLabel"
      :disabled="locked || atMax || undefined"
      @mousedown.prevent
      @click="stepBy(1)"
    >
      <span class="weft-input-number__glyph" aria-hidden="true"></span>
    </button>
  </span>
</template>
