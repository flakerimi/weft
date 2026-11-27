<script setup vapor lang="ts">
// A label, a description, a hint, an error and a required marker around one
// form control. The label is a <label> `for` the control, or, over a control
// that is not a labelable element (a group, a slider), a <span> that names it
// by id (FieldLabel); the description, the hint and the error describe it
// through aria-describedby; an error marks it aria-invalid. Inside a Form, a
// field with a name shows the Form's error for that name and is required when
// the Form's schema says so.
import { computed, hasInjectionContext, inject, onMounted, ref, shallowRef, useSlots } from 'vue'
import { ID_SCOPE, useId, type IdScope } from '../headless/useId.ts'
import { joinIds } from './dom.ts'
import { provideField, useForm } from './field.ts'
import FieldLabel from './parts/FieldLabel.vue'
import type { Size } from '../types.ts'
import '../styles/base.css'
import '../styles/form-field.css'

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
// in its setup, and the label is then no <label> (FieldLabel). The field
// draws its label before the control, so it waits to know:
// - On the server the label is drawn after the control (FieldLabel's prefetch).
// - Hydrating an island, the label is drawn as the server drew it, read from
//   the page before the first render, so the hydration matches; the control,
//   once set up, decides from then on.
// - Mounted in the browser with no server markup, the label is drawn once the
//   field has mounted, its control set up, so it is drawn once, the right
//   element, and its slot is set up once.
const inIsland = hasInjectionContext() && inject<IdScope | null>(ID_SCOPE, null) !== null
const drawn = !import.meta.env.SSR && inIsland && typeof document !== 'undefined' ? document.getElementById(labelId) : null
const asDrawn = !!drawn && drawn.tagName !== 'LABEL'
const own = shallowRef<(() => boolean) | null>(null)
const mounted = ref(false)
onMounted(() => (mounted.value = true))
const isNamed = () => (own.value ? own.value() : !mounted.value && asDrawn)
const hasLabel = () => !!props.label || !!slots.label
const drawLabel = () => hasLabel() && (import.meta.env.SSR || !!drawn || mounted.value)

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
  hasLabel,
  name: () => props.name,
  describedBy: () => joinIds(hasDescription.value && descriptionId, hasHint.value && hintId, showError.value && errorId),
  invalid: () => invalid.value,
  required: () => required.value,
  disabled: () => props.disabled || !!form?.disabled(),
  size: () => props.size,
  blur: () => props.name && form?.blur(props.name),
  input: () => props.name && form?.input(props.name),
  namedByLabel: (on = true) => (own.value = typeof on === 'function' ? on : on ? () => true : null),
})
</script>

<template>
  <div class="weft-form-field" :data-size="size" :data-invalid="invalid ? 'true' : undefined">
    <div v-if="label || $slots.label || hasHint" class="weft-form-field__head">
      <FieldLabel v-if="drawLabel()" :id="labelId" :control="id" :named="isNamed">
        <slot name="label">{{ label }}</slot>
        <span v-if="required" class="weft-form-field__required" aria-hidden="true">*</span>
      </FieldLabel>
      <span v-if="hasHint" :id="hintId" class="weft-form-field__hint"><slot name="hint">{{ hint }}</slot></span>
    </div>
    <p v-if="hasDescription" :id="descriptionId" class="weft-form-field__description"><slot name="description">{{ description }}</slot></p>
    <slot :error="message" />
    <p v-if="showError" :id="errorId" class="weft-form-field__error">
      <slot name="error" :error="message">{{ message }}</slot>
    </p>
  </div>
</template>
