<script setup vapor lang="ts">
// A one-line text field, in a box that can hold content before and after the
// text (an icon, a unit, a button). Inside a FormField it takes the field's
// id, description and error state; inside a FieldGroup, the group's size.
// Attributes go to the <input> itself (placeholder, autocomplete, aria-*),
// except class and style, which go to the box. A press anywhere on the box
// outside the text, or on its touch hit area, focuses the text.
import { useTemplateRef } from 'vue'
import Icon from '../basics/Icon.vue'
import Spinner from '../basics/Spinner.vue'
import { useControlAttrs, useField } from './field.ts'
import { pressFocuses } from './dom.ts'
import type { Size } from '../types.ts'
import '../styles/base.css'
import '../styles/input.css'

defineOptions({ inheritAttrs: false })

const props = withDefaults(
  defineProps<{
    /** The text, bound with `v-model`; a number with `v-model.number` or `type="number"`. */
    modelValue?: string | number | null
    /** From `v-model.trim`, `.number` and `.lazy` (update on change, not on every key). */
    modelModifiers?: { trim?: boolean; number?: boolean; lazy?: boolean }
    /** The native input type: `text`, `email`, `password`, `search`, `tel`, `url`, `number`. */
    type?: string
    /** `sm`, `md` or `lg`; a FieldGroup's size applies when not set. md is 44 px tall on touch, as Button is. */
    size?: Size
    /** Shown while it is empty; not a label. */
    placeholder?: string
    /** The form field name; a FormField's name applies when not set. */
    name?: string
    /** The input's id; inside a FormField the field's id is used, so its label points here. */
    id?: string
    /** A Lucide icon before the text. The leading slot replaces it. */
    icon?: string
    /** A Lucide icon after the text. The trailing slot replaces it. */
    trailingIcon?: string
    /** Work in progress (a lookup as it is typed): a spinner after the text, and `aria-busy`. */
    loading?: boolean
    /** Unavailable: natively disabled. */
    disabled?: boolean
    /** Shown but not editable. */
    readonly?: boolean
    /** The native `required`, for a form the browser checks. */
    required?: boolean
  }>(),
  { modelValue: undefined, type: 'text' },
)

const emit = defineEmits<{
  /** The text changed (on each key, or on change with `.lazy`). */
  'update:modelValue': [value: string | number | null]
  /** The input lost focus. */
  blur: [event: FocusEvent]
  /** The input took focus. */
  focus: [event: FocusEvent]
  /** The native change: the text was committed. */
  change: [event: Event]
}>()

defineSlots<{
  /** Before the text, in place of `icon`. */
  leading?: () => unknown
  /** After the text, in place of `trailing-icon`. */
  trailing?: () => unknown
}>()

const f = useField(props)
const { own, box, describedBy } = useControlAttrs(f)

const input = useTemplateRef<HTMLInputElement>('input')

function read(raw: string): string | number | null {
  const m = props.modelModifiers
  let value: string | number | null = m?.trim ? raw.trim() : raw
  if (m?.number || props.type === 'number') {
    if (value === '') return null
    const n = Number.parseFloat(value)
    if (!Number.isNaN(n)) value = n
  }
  return value
}

function commit(event: Event) {
  emit('update:modelValue', read((event.target as HTMLInputElement).value))
  f.input()
}
function onInput(event: Event) {
  if (!props.modelModifiers?.lazy) commit(event)
}
function onChange(event: Event) {
  if (props.modelModifiers?.lazy) commit(event)
  const el = event.target as HTMLInputElement
  if (props.modelModifiers?.trim) el.value = el.value.trim()
  emit('change', event)
}
function onBlur(event: FocusEvent) {
  f.blur()
  emit('blur', event)
}

defineExpose({
  /** Focus the text. */
  focus: (options?: FocusOptions) => input.value?.focus(options),
  /** The <input> element. */
  input,
})
</script>

<template>
  <span
    class="weft-input"
    v-bind="box"
    :data-size="f.size.value"
    :data-invalid="f.invalid.value ? 'true' : undefined"
    :data-disabled="f.disabled.value ? 'true' : undefined"
    @mousedown="pressFocuses($event, input)"
  >
    <span v-if="$slots.leading || icon" class="weft-input__leading">
      <slot name="leading"><Icon v-if="icon" :name="icon" /></slot>
    </span>
    <input
      ref="input"
      class="weft-input__field"
      v-bind="own"
      :id="f.id.value"
      :name="f.name.value"
      :type="type"
      :value="modelValue ?? ''"
      :placeholder="placeholder"
      :disabled="f.disabled.value || undefined"
      :readonly="readonly || undefined"
      :required="required || undefined"
      :aria-describedby="describedBy()"
      :aria-invalid="f.aria.value['aria-invalid']"
      :aria-required="f.aria.value['aria-required']"
      :aria-busy="loading ? 'true' : undefined"
      @input="onInput"
      @change="onChange"
      @blur="onBlur"
      @focus="emit('focus', $event)"
    />
    <span v-if="$slots.trailing || trailingIcon || loading" class="weft-input__trailing">
      <Spinner v-if="loading" />
      <slot v-else name="trailing"><Icon v-if="trailingIcon" :name="trailingIcon" /></slot>
    </span>
  </span>
</template>
