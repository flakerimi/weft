<script setup vapor lang="ts">
// A turning ring in the text colour, for work in progress of unknown length.
// Decorative unless it has a label; with one it is a status that announces it.
import type { Size } from '../types.ts'
import '../styles/base.css'
import '../styles/spinner.css'

defineProps<{
  /** `sm` 16 px, `md` 20 px, `lg` 24 px; without it the spinner is 1.25em, sized with the text. */
  size?: Size
  /** What is happening, announced politely ("Loading invoices"); without it the spinner is decorative. */
  label?: string
}>()
</script>

<template>
  <span
    class="weft-spinner"
    :data-size="size"
    :role="label ? 'status' : undefined"
    :aria-hidden="label ? undefined : 'true'"
  >
    <span class="weft-spinner__ring"></span>
    <span v-if="label" class="weft-visually-hidden">{{ label }}</span>
  </span>
</template>
