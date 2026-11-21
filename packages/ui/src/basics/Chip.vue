<script setup vapor lang="ts">
// A small indicator on the corner of what it wraps: an unread dot on a bell,
// a status on an avatar. Positions are logical, so `top-end` is the top
// right in a left-to-right document and the top left in a right-to-left one.
import type { Size, Tone } from '../types.ts'
import '../styles/base.css'
import '../styles/tone.css'
import '../styles/chip.css'

withDefaults(
  defineProps<{
    /** A short count or mark inside the indicator; without it the indicator is a dot. */
    text?: string | number
    /** What the indicator means, for assistive technology ("3 unread messages"); the drawn indicator is then hidden from it. */
    label?: string
    /** The indicator's colour and what it says. */
    tone?: Tone
    /** `sm`, `md` or `lg`. */
    size?: Size
    /** Which corner: `top-end`, `bottom-end`, `top-start` or `bottom-start`. */
    position?: 'top-end' | 'bottom-end' | 'top-start' | 'bottom-start'
    /** Sit inside the corner rather than across it, for round content such as an avatar. */
    inset?: boolean
    /** Show the indicator; false hides it and keeps the content. */
    show?: boolean
  }>(),
  { tone: 'primary', size: 'md', position: 'top-end', show: true },
)
</script>

<template>
  <span class="weft-chip">
    <slot />
    <span
      v-if="show"
      class="weft-chip__indicator"
      :data-tone="tone"
      :data-size="size"
      :data-position="position"
      :data-inset="inset ? 'true' : undefined"
      :data-text="text !== undefined && text !== '' ? 'true' : undefined"
      :aria-hidden="label ? 'true' : undefined"
    >{{ text }}</span>
    <span v-if="show && label" class="weft-visually-hidden">{{ label }}</span>
  </span>
</template>
