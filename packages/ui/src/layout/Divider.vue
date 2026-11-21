<script setup vapor lang="ts">
import '../styles/base.css'
import '../styles/divider.css'

withDefaults(
  defineProps<{
    /** `horizontal` separates stacked content; `vertical` separates content in a row. */
    orientation?: 'horizontal' | 'vertical'
  }>(),
  { orientation: 'horizontal' },
)
</script>

<template>
  <hr v-if="orientation === 'horizontal'" class="weft-divider" aria-orientation="horizontal" />
  <div v-else class="weft-divider" role="separator" aria-orientation="vertical" data-orientation="vertical"></div>
</template>
