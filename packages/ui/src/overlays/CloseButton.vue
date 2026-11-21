<script setup vapor lang="ts">
// The × that closes an overlay: a 28 px button (drawn smaller for a toast
// with `size="sm"`) whose hit area is at least --weft-hit-min, so 44 px on
// touch. The cross is drawn inline, so a page that never closes anything
// ships no icon for it.
import '../styles/base.css'
import '../styles/close-button.css'

withDefaults(
  defineProps<{
    /** Its name for assistive technology. */
    label?: string
    /** `sm` 24 px (a toast's), `md` 28 px. */
    size?: 'sm' | 'md'
  }>(),
  { label: 'Close', size: 'md' },
)

const emit = defineEmits<{
  /** Pressed. */
  click: [event: MouseEvent]
}>()
</script>

<template>
  <button type="button" class="weft-close-button" :aria-label="label" :data-size="size" @click="emit('click', $event)">
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M18 6 6 18" /><path d="m6 6 12 12" /></svg>
  </button>
</template>
