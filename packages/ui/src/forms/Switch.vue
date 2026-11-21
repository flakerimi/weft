<script setup vapor lang="ts">
// On or off, taking effect at once: a native checkbox with role="switch", so
// it posts with a plain form, Space toggles it and assistive technology says
// "on" and "off". Loading keeps it focusable, sets aria-busy and
// aria-disabled, and blocks toggling by press, Space and label alike. The
// input lies over the drawn track at least --weft-hit-min square, so on touch
// the track keeps its size but takes a 44 px press.
import { computed, useTemplateRef } from 'vue'
import { useControllable } from '../headless/useControllable.ts'
import { useId } from '../headless/useId.ts'
import { useControlAttrs, useField } from './field.ts'
import Spinner from '../basics/Spinner.vue'
import type { Size, Tone } from '../types.ts'

defineOptions({ inheritAttrs: false })

const props = withDefaults(
  defineProps<{
    /** On or off, bound with `v-model`. */
    modelValue?: boolean
    /** The state to start in when `v-model` is not bound. */
    defaultValue?: boolean
    /** The text beside the switch. The default slot replaces it. */
    label?: string
    /** A sentence under the label, which also describes the switch. */
    description?: string
    /** What a form post sends while it is on (default `on`). */
    value?: string
    /** The form field name; a FormField's applies when not set. */
    name?: string
    /** The input's id; generated when not set. */
    id?: string
    /** `sm`, `md` or `lg`. */
    size?: Size
    /** The colour while on. */
    tone?: Tone
    /** Work in progress (the change is being saved): a spinner in the thumb, aria-busy, and no toggling. */
    loading?: boolean
    /** Unavailable: natively disabled. */
    disabled?: boolean
    /** The native `required`: it must be on. */
    required?: boolean
  }>(),
  { modelValue: undefined, defaultValue: false, tone: 'primary' },
)

const emit = defineEmits<{
  /** Turned on or off. */
  'update:modelValue': [value: boolean]
  /** The native change. */
  change: [event: Event]
}>()

defineSlots<{
  /** The label, in place of `label`. */
  default?: () => unknown
}>()

const f = useField(props)
const { own, box, describedBy } = useControlAttrs(f)
const model = useControllable(props, 'modelValue', emit, () => props.defaultValue)
const id = f.id.value ?? props.id ?? useId('switch')
const descriptionId = `${id}-description`
const input = useTemplateRef<HTMLInputElement>('input')

/** While loading, a press (or Space, or the label, which reach it as a click) changes nothing. */
function onClick(event: MouseEvent) {
  if (props.loading) event.preventDefault()
}
function onChange(event: Event) {
  const el = event.target as HTMLInputElement
  model.value = el.checked
  f.input()
  el.checked = model.value
  emit('change', event)
}

const on = computed(() => model.value)

defineExpose({
  /** Focus the switch. */
  focus: (options?: FocusOptions) => input.value?.focus(options),
})
</script>

<template>
  <div
    class="weft-switch"
    v-bind="box"
    :data-size="f.size.value"
    :data-tone="tone"
    :data-state="on ? 'checked' : 'unchecked'"
    :data-disabled="f.disabled.value ? 'true' : undefined"
    :data-loading="loading ? 'true' : undefined"
    :data-invalid="f.invalid.value ? 'true' : undefined"
  >
    <span class="weft-switch__control">
      <input
        ref="input"
        class="weft-switch__input"
        type="checkbox"
        role="switch"
        v-bind="own"
        :id="id"
        :name="f.name.value"
        :value="value"
        :checked="on"
        :disabled="f.disabled.value || undefined"
        :required="required || undefined"
        :aria-busy="loading ? 'true' : undefined"
        :aria-disabled="loading ? 'true' : undefined"
        :aria-describedby="describedBy(description && descriptionId)"
        :aria-invalid="f.aria.value['aria-invalid']"
        :aria-required="f.aria.value['aria-required']"
        @click="onClick"
        @change="onChange"
        @blur="f.blur()"
      />
      <span class="weft-switch__track" aria-hidden="true">
        <span class="weft-switch__thumb"><Spinner v-if="loading" /></span>
      </span>
    </span>
    <span v-if="label || $slots.default || description" class="weft-switch__text">
      <label v-if="label || $slots.default" class="weft-switch__label" :for="id"><slot>{{ label }}</slot></label>
      <span v-if="description" :id="descriptionId" class="weft-switch__description">{{ description }}</span>
    </span>
  </div>
</template>
