<script setup vapor lang="ts">
// Controls joined into one: an input with a button at its end, a row of
// buttons, a currency picker before an amount. Their borders meet and only
// the outer corners are round. Form controls in the group take its size
// unless they set their own; give Buttons in it the same `size`.
import { provideFieldGroup } from './field.ts'
import type { Size } from '../types.ts'
import '../styles/base.css'
import '../styles/field-group.css'

const props = withDefaults(
  defineProps<{
    /** `horizontal` (a row) or `vertical` (a column). */
    orientation?: 'horizontal' | 'vertical'
    /** `sm`, `md` or `lg`, for the form controls in it. */
    size?: Size
  }>(),
  { orientation: 'horizontal' },
)

defineSlots<{
  /** The controls. */
  default?: () => unknown
}>()

provideFieldGroup({ size: () => props.size, orientation: () => props.orientation })
</script>

<template>
  <div class="weft-field-group" role="group" :data-orientation="orientation" :data-size="size">
    <slot />
  </div>
</template>
