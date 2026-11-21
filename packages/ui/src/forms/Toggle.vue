<script setup vapor lang="ts">
// A button that stays pressed or not (APG button, toggle): aria-pressed says
// which, a press, Enter or Space changes it. Drawn ghost and neutral, soft
// and primary while pressed. With a `name` it posts `value` (default `on`)
// with a plain form while pressed, as a checkbox does. Loading and disabled
// block it as they block a Button (by press, key and form alike); on touch
// it is at least --weft-hit-min tall (Button's rule).
import { computed, useAttrs } from 'vue'
import Button from '../basics/Button.vue'
import { useControllable } from '../headless/useControllable.ts'
import { useField } from './field.ts'
import type { Size, Tone, Variant } from '../types.ts'
import '../styles/base.css'

defineOptions({ inheritAttrs: false })

const props = withDefaults(
  defineProps<{
    /** Pressed or not, bound with `v-model`. */
    modelValue?: boolean
    /** Pressed at first when `v-model` is not bound. */
    defaultValue?: boolean
    /** The text; the default slot replaces it. */
    label?: string
    /** A Lucide icon before the label; alone, the toggle is square and needs an `aria-label`. */
    icon?: string
    /** A Lucide icon after the label. */
    trailingIcon?: string
    /** The form field name: it posts `value` while pressed. */
    name?: string
    /** What a form post sends while pressed (default `on`). */
    value?: string
    /** The button's id. */
    id?: string
    /** `sm`, `md` or `lg`. */
    size?: Size
    /** The colour while not pressed. */
    tone?: Tone
    /** The variant while not pressed. */
    variant?: Variant
    /** The colour while pressed. */
    activeTone?: Tone
    /** The variant while pressed. */
    activeVariant?: Variant
    /** Square, its width equal to its height. */
    square?: boolean
    /** Work in progress (the change is being saved): a spinner, aria-busy, and no toggling. */
    loading?: boolean
    /** Unavailable: natively disabled. */
    disabled?: boolean
  }>(),
  { modelValue: undefined, defaultValue: false, value: 'on', tone: 'neutral', variant: 'ghost', activeTone: 'primary', activeVariant: 'soft' },
)

const emit = defineEmits<{
  /** Pressed or released. */
  'update:modelValue': [value: boolean]
}>()

defineSlots<{
  /** The label, in place of `label`. */
  default?: () => unknown
  /** Before the label, in place of `icon`. */
  leading?: () => unknown
  /** After the label, in place of `trailing-icon`. */
  trailing?: () => unknown
}>()

const attrs = useAttrs()
const f = useField(props)
const on = useControllable(props, 'modelValue', emit, () => props.defaultValue)
const size = computed(() => props.size ?? f.size.value)
</script>

<template>
  <Button
    v-bind="attrs"
    :id="f.id.value"
    class="weft-toggle"
    :label="label"
    :icon="icon"
    :trailing-icon="trailingIcon"
    :size="size"
    :tone="on ? activeTone : tone"
    :variant="on ? activeVariant : variant"
    :square="square"
    :loading="loading"
    :disabled="f.disabled.value"
    :active="on"
    :aria-pressed="on ? 'true' : 'false'"
    :aria-describedby="f.aria.value['aria-describedby']"
    @click="on = !on"
  >
    <template v-if="$slots.leading" #leading><slot name="leading" /></template>
    <template v-if="$slots.default" #default><slot /></template>
    <template v-if="$slots.trailing" #trailing><slot name="trailing" /></template>
  </Button>
  <input v-if="on && f.name.value" type="hidden" :name="f.name.value" :value="value" />
</template>
