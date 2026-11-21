<script setup vapor lang="ts">
// A label, a description, a hint, an error and a required marker around one
// form control. The label is `for` the control, or, for a control that is not
// a labelable element (a group, a slider), names it by id instead; the
// description, the hint and the error describe it through aria-describedby;
// an error marks it aria-invalid. Inside a Form, a field with a name shows the
// Form's error for that name and is required when the Form's schema says so.
import { computed, ref, useSlots } from 'vue'
import { useId } from '../headless/useId.ts'
import { joinIds } from './dom.ts'
import { provideField, useForm } from './field.ts'
import type { Size } from '../types.ts'

const props = withDefaults(
  defineProps<{
    /** What the control is for. The label slot replaces it. */
    label?: string
    /** Help in a sentence, under the label ("We never share it."). The description slot replaces it. */
    description?: string
    /** A short note beside the label ("Optional", "3 of 10"). The hint slot replaces it. */
    hint?: string
    /** What is wrong, under the control; `true` marks it invalid without a sentence. Wins over the Form's error. */
    error?: string | boolean
    /** Marks the field required (an asterisk, and aria-required on the control); a Form's schema can say so too. */
    required?: boolean
    /** The field's name: the control's name, and the key of its value, rules and error in a Form. */
    name?: string
    /** The control's id; generated when not set. */
    id?: string
    /** `sm`, `md` or `lg`, for the label and the control. */
    size?: Size
    /** Disables the control. */
    disabled?: boolean
  }>(),
  { error: undefined },
)

defineSlots<{
  /** The control, given the field's error. */
  default?: (props: { error: string | undefined }) => unknown
  /** In place of `label`. */
  label?: () => unknown
  /** In place of `description`. */
  description?: () => unknown
  /** In place of `hint`. */
  hint?: () => unknown
  /** In place of the error's sentence; shown only while the field is invalid. */
  error?: (props: { error: string | undefined }) => unknown
}>()

const slots = useSlots()
const form = useForm()
const base = useId('field')
const id = props.id ?? base
const labelId = `${base}-label`
const descriptionId = `${base}-description`
const hintId = `${base}-hint`
const errorId = `${base}-error`
// A control that names itself by the label's id (a group, a slider) says so
// in its setup; the label then points at no control. (The server renders
// the label before the control's setup runs, so there the label keeps `for`
// until hydration drops it.)
const namedByLabel = ref(false)

const message = computed(() => {
  if (typeof props.error === 'string') return props.error || undefined
  if (props.error === undefined && props.name) return form?.error(props.name)
  return undefined
})
const invalid = computed(() => props.error === true || !!message.value)
const required = computed(() => props.required || (!!props.name && !!form?.required(props.name)))
const hasDescription = computed(() => !!props.description || !!slots.description)
const hasHint = computed(() => !!props.hint || !!slots.hint)
const showError = computed(() => !!message.value || (invalid.value && !!slots.error))

provideField({
  id,
  labelId,
  name: () => props.name,
  describedBy: () => joinIds(hasDescription.value && descriptionId, hasHint.value && hintId, showError.value && errorId),
  invalid: () => invalid.value,
  required: () => required.value,
  disabled: () => props.disabled || !!form?.disabled(),
  size: () => props.size,
  blur: () => props.name && form?.blur(props.name),
  input: () => props.name && form?.input(props.name),
  namedByLabel: () => (namedByLabel.value = true),
})
</script>

<template>
  <div class="weft-form-field" :data-size="size" :data-invalid="invalid ? 'true' : undefined">
    <div v-if="label || $slots.label || hasHint" class="weft-form-field__head">
      <label v-if="label || $slots.label" :id="labelId" :for="namedByLabel ? undefined : id" class="weft-form-field__label">
        <slot name="label">{{ label }}</slot>
        <span v-if="required" class="weft-form-field__required" aria-hidden="true">*</span>
      </label>
      <span v-if="hasHint" :id="hintId" class="weft-form-field__hint"><slot name="hint">{{ hint }}</slot></span>
    </div>
    <p v-if="hasDescription" :id="descriptionId" class="weft-form-field__description"><slot name="description">{{ description }}</slot></p>
    <slot :error="message" />
    <p v-if="showError" :id="errorId" class="weft-form-field__error">
      <slot name="error" :error="message">{{ message }}</slot>
    </p>
  </div>
</template>
