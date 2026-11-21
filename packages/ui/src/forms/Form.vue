<script setup vapor lang="ts" generic="T extends Record<string, unknown>">
// A form that checks its values against a Weft field schema (see schema.ts)
// before it submits. The values are `state`, an object the fields bind into
// with v-model; each FormField with a `name` shows the error for that name.
//
// - On submit every field in the schema is checked. With errors, `error`
//   fires, each field shows its error and focus moves to the first invalid
//   control; without, the `submit` handler runs with a deep copy of the
//   values.
// - After the first submit, a field is checked again when it is left, and a
//   field showing an error is checked again as it changes, so the error goes
//   as soon as it is fixed.
// - `setErrors({ field: message })` shows errors from elsewhere (a server's
//   answer). Leaving such a field keeps its error; changing it checks it.
// - While `disabled`, or busy (`loading`, or an async submit handler still
//   running, with `loading-auto`), every control in it is disabled, buttons
//   too, and a submit does nothing. A disabled <fieldset> holds the content,
//   so native controls (a Button) are disabled as well as Weft's own.
// The native form keeps `novalidate`: Weft shows the errors, not the browser.
import { computed, isProxy, nextTick, ref, toRaw, useTemplateRef } from 'vue'
import { tabbables } from '../headless/dom.ts'
import { provideForm } from './field.ts'
import { validateField, validateFields, type FieldSchema } from './schema.ts'

const props = withDefaults(
  defineProps<{
    /** The values, by field name: an object the fields bind into with `v-model`. */
    state: T
    /** The rules each field must meet, by name: `{ email: { required: true, email: true } }`. */
    schema?: FieldSchema<Extract<keyof T, string>>
    /** Disables every control in it, buttons too, and a submit does nothing. */
    disabled?: boolean
    /** Busy: as disabled, and `aria-busy`, so a second submit does nothing. */
    loading?: boolean
    /** Busy on its own while the submit handler's promise is pending. */
    loadingAuto?: boolean
    /** Bound with `@submit`: runs once every field is valid, with a copy of the values; a promise it returns keeps the form busy. */
    onSubmit?: (values: T) => unknown
  }>(),
  { schema: () => ({}), loadingAuto: true },
)

const emit = defineEmits<{
  /** Submitted with errors: each failing field's sentence, by name. */
  error: [errors: Record<string, string>]
}>()

defineSlots<{
  /** The fields and the submit button. */
  default?: () => unknown
}>()

const errors = ref<Record<string, string>>({})
const submitted = ref(false)
const pending = ref(false)
const busy = computed(() => props.loading || pending.value)
const off = computed(() => props.disabled || busy.value)
/** Fields whose error came from setErrors: leaving them keeps it. */
const fromServer = new Set<string>()
const form = useTemplateRef<HTMLFormElement>('form')

const ruleOf = (name: string) => (props.schema as FieldSchema)[name]
/** Check one field now: show its error, or clear it. */
function check(name: string) {
  fromServer.delete(name)
  const rule = ruleOf(name)
  const error = rule ? validateField(props.state[name], rule) : undefined
  const { [name]: _gone, ...rest } = errors.value
  errors.value = error === undefined ? rest : { ...rest, [name]: error }
}

provideForm({
  error: (name) => errors.value[name],
  required: (name) => !!ruleOf(name)?.required,
  disabled: () => off.value,
  blur(name) {
    if (submitted.value && !fromServer.has(name)) check(name)
  },
  input(name) {
    if (name in errors.value) check(name)
  },
})

/** Check every field; true when all pass. */
function validate(): boolean {
  fromServer.clear()
  errors.value = validateFields(props.schema as FieldSchema, props.state)
  return Object.keys(errors.value).length === 0
}

/** A deep copy of `value`, out of Vue's proxies: lists and plain objects copied, anything else (a File, a Date) as it is. */
function copy<V>(value: V): V {
  const raw = isProxy(value) ? toRaw(value) : value
  if (Array.isArray(raw)) return raw.map(copy) as V
  if (raw !== null && typeof raw === 'object' && Object.getPrototypeOf(raw) === Object.prototype) {
    return Object.fromEntries(Object.entries(raw).map(([k, v]) => [k, copy(v)])) as V
  }
  return raw
}

async function focusFirstInvalid() {
  await nextTick()
  const invalid = form.value?.querySelector<HTMLElement>('[aria-invalid="true"]')
  if (!invalid) return
  const target = invalid.matches('input, textarea, select, [tabindex]') ? invalid : tabbables(invalid)[0]
  target?.focus()
}

async function onSubmit(event: Event) {
  event.preventDefault()
  if (off.value) return
  submitted.value = true
  if (!validate()) {
    emit('error', { ...errors.value })
    void focusFirstInvalid()
    return
  }
  const result = props.onSubmit?.(copy(props.state))
  if (!props.loadingAuto || !(result instanceof Promise)) return
  pending.value = true
  try {
    await result
  } finally {
    pending.value = false
  }
}

defineExpose({
  /** Show these errors (a server's answer), in place of the ones shown now; `{}` clears them. */
  setErrors(next: Record<string, string>) {
    errors.value = { ...next }
    fromServer.clear()
    for (const name of Object.keys(next)) fromServer.add(name)
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
    fromServer.clear()
    submitted.value = false
  },
  /** The errors shown now, by field name. */
  errors,
  /** Busy: `loading`, or an async submit handler still running. */
  loading: busy,
  /** Submit as the submit button would: check, then run the handler or emit `error`. */
  submit: () => form.value?.requestSubmit(),
})
</script>

<template>
  <form
    ref="form"
    class="weft-form"
    novalidate
    :data-disabled="off ? 'true' : undefined"
    :aria-busy="busy ? 'true' : undefined"
    @submit="onSubmit"
  >
    <fieldset class="weft-form__body" :disabled="off || undefined">
      <slot />
    </fieldset>
  </form>
</template>
