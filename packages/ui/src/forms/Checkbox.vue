<script setup vapor lang="ts">
// A box to tick, with its label beside it: a native checkbox, so it takes part
// in a plain form post and Space toggles it. It can be indeterminate (some of
// the things it stands for are chosen), which a press turns to ticked. In a
// CheckboxGroup it is ticked while its value is in the group's list. The
// native input lies over the drawn box at least --weft-hit-min square, so on
// touch the box keeps its size but takes a 44 px press.
import { computed, useTemplateRef } from 'vue'
import { useControllable } from '../headless/useControllable.ts'
import { useId } from '../headless/useId.ts'
import { useCheckboxGroup } from './choice.ts'
import { useControlAttrs, useField } from './field.ts'
import type { Size, Tone } from '../types.ts'

defineOptions({ inheritAttrs: false })

const props = withDefaults(
  defineProps<{
    /** Ticked (`true`), not (`false`), or `'indeterminate'`; bound with `v-model`. */
    modelValue?: boolean | 'indeterminate'
    /** The state to start in when `v-model` is not bound. */
    defaultValue?: boolean | 'indeterminate'
    /** The text beside the box. The default slot replaces it. */
    label?: string
    /** A sentence under the label, which also describes the box. */
    description?: string
    /** The value it stands for: what a CheckboxGroup lists, and what a form post sends (default `on`). */
    value?: string | number
    /** The form field name; a group's or a FormField's applies when not set. */
    name?: string
    /** The input's id; generated when not set. */
    id?: string
    /** `sm`, `md` or `lg`. */
    size?: Size
    /** The colour of the ticked box. */
    tone?: Tone
    /** Unavailable: natively disabled. */
    disabled?: boolean
    /** The native `required`. */
    required?: boolean
  }>(),
  { modelValue: undefined, defaultValue: false },
)

const emit = defineEmits<{
  /** Ticked or unticked by a press, Space or its label. */
  'update:modelValue': [value: boolean | 'indeterminate']
  /** The native change. */
  change: [event: Event]
}>()

defineSlots<{
  /** The label, in place of `label`. */
  default?: () => unknown
}>()

const group = useCheckboxGroup()
const f = useField(props)
const { own, describedBy } = useControlAttrs(f)
const model = useControllable(props, 'modelValue', emit, () => props.defaultValue)
const id = f.id.value ?? props.id ?? useId('checkbox')
const descriptionId = `${id}-description`

const state = computed<boolean | 'indeterminate'>(() =>
  group && props.value !== undefined ? group.has(props.value) : model.value,
)
const size = computed(() => props.size ?? group?.size() ?? f.size.value)
const tone = computed(() => props.tone ?? group?.tone() ?? 'primary')
const disabled = computed(() => f.disabled.value || !!group?.disabled())
const input = useTemplateRef<HTMLInputElement>('input')

function onChange(event: Event) {
  const el = event.target as HTMLInputElement
  if (group && props.value !== undefined) group.toggle(props.value, el.checked)
  else model.value = el.checked
  f.input()
  // A bound value the parent did not change stays as it was.
  el.checked = state.value === true
  el.indeterminate = state.value === 'indeterminate'
  emit('change', event)
}

defineExpose({
  /** Focus the box. */
  focus: (options?: FocusOptions) => input.value?.focus(options),
})
</script>

<template>
  <div
    class="weft-checkbox"
    :class="$attrs.class"
    :style="$attrs.style"
    :data-size="size"
    :data-tone="tone"
    :data-state="state === 'indeterminate' ? 'indeterminate' : state ? 'checked' : 'unchecked'"
    :data-disabled="disabled ? 'true' : undefined"
    :data-invalid="f.invalid.value ? 'true' : undefined"
  >
    <span class="weft-checkbox__control">
      <input
        ref="input"
        class="weft-checkbox__input"
        type="checkbox"
        v-bind="own"
        :id="id"
        :name="f.name.value ?? group?.name()"
        :value="value"
        :checked="state === true"
        :indeterminate="state === 'indeterminate' || undefined"
        :disabled="disabled || undefined"
        :required="required || undefined"
        :aria-describedby="describedBy(description && descriptionId)"
        :aria-invalid="f.aria.value['aria-invalid']"
        :aria-required="f.aria.value['aria-required']"
        @change="onChange"
        @blur="f.blur()"
      />
      <span class="weft-checkbox__box" aria-hidden="true">
        <svg class="weft-checkbox__mark" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round">
          <path v-if="state === 'indeterminate'" d="M6 12h12"></path>
          <path v-else d="M20 6 9 17l-5-5"></path>
        </svg>
      </span>
    </span>
    <span v-if="label || $slots.default || description" class="weft-checkbox__text">
      <label v-if="label || $slots.default" class="weft-checkbox__label" :for="id"><slot>{{ label }}</slot></label>
      <span v-if="description" :id="descriptionId" class="weft-checkbox__description">{{ description }}</span>
    </span>
  </div>
</template>
