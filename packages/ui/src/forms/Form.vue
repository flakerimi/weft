<script setup vapor lang="ts" generic="T extends Record<string, unknown>">
// A form that checks its values against a Weft field schema (see schema.ts)
// before it submits. The values are `state`, an object the fields bind into
// with v-model; each FormField with a `name` shows the error for that name.
//
// - On submit every field in the schema is checked. With errors, `error`
//   fires, each field shows its error and focus moves to the first invalid
//   control; without, `submit` fires with the values.
// - After the first submit, a field is checked again when it is left, and a
//   field showing an error is checked again as it changes, so the error goes
//   as soon as it is fixed.
// - `setErrors({ field: message })` shows errors from elsewhere (a server's
//   answer); a field's error from there stays until the field is checked again.
// - While `disabled`, its fields are disabled and a submit does nothing.
// The native form keeps `novalidate`: Weft shows the errors, not the browser.
import { nextTick, ref, toRaw, useTemplateRef } from 'vue'
import { tabbables } from '../headless/dom.ts'
import { provideForm } from './field.ts'
import { validateField, validateFields, type FieldSchema } from './schema.ts'

const props = withDefaults(
  defineProps<{
    /** The values, by field name: an object the fields bind into with `v-model`. */
    state: T
    /** The rules each field must meet, by name: `{ email: { required: true, email: true } }`. */
    schema?: FieldSchema<Extract<keyof T, string>>
    /** Disables every field, and a submit does nothing. */
    disabled?: boolean
  }>(),
  { schema: () => ({}) },
)

const emit = defineEmits<{
  /** Submitted with every field valid: a copy of the values. */
  submit: [values: T]
  /** Submitted with errors: each failing field's sentence, by name. */
  error: [errors: Record<string, string>]
}>()

defineSlots<{
  /** The fields and the submit button. */
  default?: () => unknown
}>()

const errors = ref<Record<string, string>>({})
const submitted = ref(false)
const form = useTemplateRef<HTMLFormElement>('form')

const ruleOf = (name: string) => (props.schema as FieldSchema)[name]
/** Check one field now: show its error, or clear it. */
function check(name: string) {
  const rule = ruleOf(name)
  const error = rule ? validateField(props.state[name], rule) : undefined
  const { [name]: _gone, ...rest } = errors.value
  errors.value = error === undefined ? rest : { ...rest, [name]: error }
}

provideForm({
  error: (name) => errors.value[name],
  required: (name) => !!ruleOf(name)?.required,
  disabled: () => props.disabled,
  blur(name) {
    if (submitted.value) check(name)
  },
  input(name) {
    if (name in errors.value) check(name)
  },
})

/** Check every field; true when all pass. */
function validate(): boolean {
  errors.value = validateFields(props.schema as FieldSchema, props.state)
  return Object.keys(errors.value).length === 0
}

async function focusFirstInvalid() {
  await nextTick()
  const invalid = form.value?.querySelector<HTMLElement>('[aria-invalid="true"]')
  if (!invalid) return
  const target = invalid.matches('input, textarea, select, [tabindex]') ? invalid : tabbables(invalid)[0]
  target?.focus()
}

function onSubmit(event: Event) {
  event.preventDefault()
  if (props.disabled) return
  submitted.value = true
  if (validate()) emit('submit', { ...toRaw(props.state) })
  else {
    emit('error', { ...errors.value })
    void focusFirstInvalid()
  }
}

defineExpose({
  /** Show these errors (a server's answer), in place of the ones shown now; `{}` clears them. */
  setErrors(next: Record<string, string>) {
    errors.value = { ...next }
    submitted.value = true
    if (Object.keys(next).length) void focusFirstInvalid()
  },
  /** Check every field and show the errors, as a submit would, without submitting; true when all pass. */
  validate() {
    submitted.value = true
    return validate()
  },
  /** Clear every error, and forget the submit (fields are checked on leaving only after the next one). */
  clear() {
    errors.value = {}
    submitted.value = false
  },
  /** The errors shown now, by field name. */
  errors,
  /** Submit as the submit button would: check, then emit `submit` or `error`. */
  submit: () => form.value?.requestSubmit(),
})
</script>

<template>
  <form ref="form" class="weft-form" novalidate :data-disabled="disabled ? 'true' : undefined" @submit="onSubmit">
    <slot />
  </form>
</template>
