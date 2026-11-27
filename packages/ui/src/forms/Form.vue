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
// - While `disabled`, every control in it is disabled, buttons and links too,
//   and a submit does nothing: a disabled <fieldset> holds the content, so
//   native controls are disabled as well as Weft's own, and Button and Link
//   read the Form to drop their addresses (a fieldset leaves links alone).
// - While busy (`loading`, or an async submit handler still running, with
//   `loading-auto`), a submit does nothing, Weft's fields are disabled and
//   its links have no address. The submit button is not natively disabled:
//   it shows a spinner, is aria-disabled and ignores presses, so it keeps
//   focus; other buttons work, so a Cancel can abort (as in Nuxt UI). Raw
//   native controls in the slot (a plain <input>) are not blocked while
//   busy, only while disabled. Focus that busy took from a field (a
//   disabled control loses it) goes back when it is done, whether busy came
//   from `loading` or from the handler.
// - `setErrors` on a field the schema has no rule for: the error clears the
//   first time the field changes, as there is nothing to check it against.
// - It sends nothing before it is live. The server renders it in full, so
//   until its island hydrates a press on submit, or Enter in a field, would
//   be the browser's own submit. A pre-paint script (usePrePaint, one per
//   page) stops every submit of a `.weft-form` from the first moment, and
//   leaves the fields usable, so a person can type and a password manager
//   fill while the script loads (the island runtime keeps what they hold
//   through the hydration); with scripts off, `method="post"` keeps the
//   values out of the address.
// - An emptied field the schema marks `nullable` (a pointer in Go, as weft
//   gen's schema says) is sent as null, '' too, so the person's clear
//   reaches the server and clears it. Any other null (an emptied number,
//   date or choice) is left out of the values the handler gets, so the
//   server reads it as not sent, as Go refuses null for a field that is not a
//   pointer; with no schema every null is left out. A required field is
//   stopped first, by its own `required` rule.
// The native form keeps `novalidate`: Weft shows the errors, not the browser.
import { computed, isProxy, nextTick, ref, toRaw, useTemplateRef, watch } from 'vue'
import { usePrePaint } from '../head/usePrePaint.ts'
import { formGuard } from './guard.ts'
import { tabbables } from '../headless/dom.ts'
import { provideForm } from './field.ts'
import { validateField, validateFields, type FieldSchema } from './schema.ts'
import '../styles/base.css'
import '../styles/form.css'

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

// Before the body is parsed: once hydrated, onSubmit stops the submit as well.
usePrePaint('form:submit', import.meta.env.SSR ? formGuard(import.meta.env.DEV) : '')

const errors = ref<Record<string, string>>({})
const submitted = ref(false)
const pending = ref(false)
const busy = computed(() => props.loading || pending.value)
const off = computed(() => props.disabled || busy.value)
/** Fields whose error came from setErrors: leaving them keeps it. */
const fromServer = new Set<string>()
const form = useTemplateRef<HTMLFormElement>('form')

/** The control in the form that had focus as it turned busy; undefined when none was noted. */
let had: HTMLElement | null | undefined
let pendingFocusInvalid = false
const inForm = (el: Element | null) => (el instanceof HTMLElement && form.value?.contains(el) ? el : null)
// Synchronous, so the focus is noted before busy disables the field that has it.
watch(busy, (on) => on && (had ??= inForm(document.activeElement)), { flush: 'sync' })
// After the DOM update that enables the fields again: a field that had focus
// was disabled while busy, which drops focus to the body; give it back, unless
// setErrors asked to focus the first invalid field while the form was busy.
watch(
  busy,
  (on) => {
    if (on) return
    const back = had
    had = undefined
    if (pendingFocusInvalid) {
      pendingFocusInvalid = false
      void focusFirstInvalid()
      return
    }
    const now = document.activeElement
    if (back?.isConnected && (!now || now === document.body)) back.focus()
  },
  { flush: 'post' },
)

/** The values the handler gets: a copy, an emptied nullable field as null, and no other null. */
function valuesToSend(): T {
  const values = copy(props.state) as Record<string, unknown>
  for (const [name, value] of Object.entries(values)) {
    if (ruleOf(name)?.nullable) {
      if (value === undefined || value === '') values[name] = null
    } else if (value === null || value === undefined) {
      delete values[name]
    }
  }
  return values as T
}

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
  busy: () => busy.value,
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

/**
 * A deep copy of `value`, out of Vue's proxies: arrays, plain objects, Maps
 * (keys kept, values copied) and Sets (members copied) are copied; anything
 * else (a File, a Date, a class instance) is passed as it is. An object
 * reached twice, or a cycle, is copied once and reached the same way.
 */
function copy<V>(value: V, seen = new Map<object, unknown>()): V {
  const raw = (isProxy(value) ? toRaw(value) : value) as unknown
  if (raw === null || typeof raw !== 'object') return raw as V
  if (seen.has(raw)) return seen.get(raw) as V
  if (Array.isArray(raw)) {
    const out: unknown[] = []
    seen.set(raw, out)
    for (const item of raw) out.push(copy(item, seen))
    return out as V
  }
  if (raw instanceof Map) {
    const out = new Map()
    seen.set(raw, out)
    for (const [k, v] of raw) out.set(k, copy(v, seen))
    return out as V
  }
  if (raw instanceof Set) {
    const out = new Set()
    seen.set(raw, out)
    for (const v of raw) out.add(copy(v, seen))
    return out as V
  }
  if (Object.getPrototypeOf(raw) !== Object.prototype && Object.getPrototypeOf(raw) !== null) return raw as V
  const out: Record<string, unknown> = {}
  seen.set(raw, out)
  for (const [k, v] of Object.entries(raw)) out[k] = copy(v, seen)
  return out as V
}

async function focusFirstInvalid() {
  if (off.value) {
    pendingFocusInvalid = true
    return
  }
  await nextTick()
  if (off.value) {
    pendingFocusInvalid = true
    return
  }
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
  // Noted before the handler runs, which may move focus at once.
  const focused = inForm(document.activeElement)
  const result = props.onSubmit?.(valuesToSend())
  if (!props.loadingAuto || !(result instanceof Promise)) return
  if (!busy.value) had = focused
  pending.value = true
  try {
    await result
  } finally {
    pending.value = false
  }
}

// Typed for apps as FormApi (form-state.ts);
// test/fixtures/forms/FormRef.vue checks the two agree.
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
    method="post"
    novalidate
    :data-disabled="off ? 'true' : undefined"
    :aria-busy="busy ? 'true' : undefined"
    @submit="onSubmit"
  >
    <fieldset class="weft-form__body" :disabled="disabled || undefined">
      <slot />
    </fieldset>
  </form>
</template>
