<script setup vapor lang="ts">
// A rule that separates content, with an optional label or icon in its
// middle ("or", a date). A `separator` named by its label, or, when
// decorative, hidden from assistive technology. For a plain rule, Divider.
import Icon from '../basics/Icon.vue'
import '../styles/base.css'
import '../styles/separator.css'

withDefaults(
  defineProps<{
    /** Text in the middle of the rule ("or"); the default slot replaces it. */
    label?: string
    /** A Lucide icon in the middle of the rule, before the label. */
    icon?: string
    /** `horizontal` between stacked content; `vertical` between content in a row. */
    orientation?: 'horizontal' | 'vertical'
    /** Only visual: hidden from assistive technology, for a rule that separates nothing a reader needs told about. */
    decorative?: boolean
  }>(),
  { orientation: 'horizontal' },
)
</script>

<template>
  <div
    class="weft-separator"
    :role="decorative ? 'none' : 'separator'"
    :aria-hidden="decorative ? 'true' : undefined"
    :aria-orientation="decorative ? undefined : orientation"
    :aria-label="!decorative && label ? label : undefined"
    :data-orientation="orientation"
  >
    <span class="weft-separator__line"></span>
    <template v-if="label || icon || $slots.default">
      <span class="weft-separator__content">
        <Icon v-if="icon" :name="icon" />
        <slot>{{ label }}</slot>
      </span>
      <span class="weft-separator__line"></span>
    </template>
  </div>
</template>
