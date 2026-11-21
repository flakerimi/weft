<script setup vapor lang="ts">
// A placeholder in the shape of content still loading. Hidden from assistive
// technology: mark the region that is loading with aria-busy instead. It
// pulses, and stands still under reduced motion.
import '../styles/base.css'
import '../styles/skeleton.css'

defineProps<{
  /** Any CSS length; without it, the full width. */
  width?: string
  /** Any CSS length; without it, one line of text (1em). */
  height?: string
  /** A circle, for an avatar; set width and height alike. */
  circle?: boolean
}>()
</script>

<template>
  <span
    class="weft-skeleton"
    aria-hidden="true"
    :data-circle="circle ? 'true' : undefined"
    :style="{ inlineSize: width, blockSize: height }"
  ></span>
</template>
