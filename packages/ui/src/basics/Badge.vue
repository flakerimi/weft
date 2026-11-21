<script setup vapor lang="ts">
// A short label for a status or a count, drawn in a tone. Not interactive.
import Icon from './Icon.vue'
import type { Size, Tone, Variant } from '../types.ts'
import '../styles/base.css'
import '../styles/tone.css'
import '../styles/badge.css'

withDefaults(
  defineProps<{
    /** The badge's text; the default slot replaces it. */
    label?: string | number
    /** `solid` to stand out, `soft` and `outline` to sit quietly beside text, `ghost` for tone alone. */
    variant?: Variant
    /** The colour and what it says: `success` for done, `danger` for failed. */
    tone?: Tone
    /** `sm`, `md` or `lg`. */
    size?: Size
    /** A Lucide icon before the label; alone, the badge is square. */
    icon?: string
    /** A Lucide icon after the label. */
    trailingIcon?: string
  }>(),
  { variant: 'solid', tone: 'primary', size: 'md' },
)
</script>

<template>
  <span
    class="weft-badge"
    :data-variant="variant"
    :data-tone="tone"
    :data-size="size"
    :data-square="!$slots.default && (label === undefined || label === '') && (icon || trailingIcon) ? 'true' : undefined"
  >
    <Icon v-if="icon" :name="icon" />
    <slot>{{ label }}</slot>
    <Icon v-if="trailingIcon" :name="trailingIcon" />
  </span>
</template>
