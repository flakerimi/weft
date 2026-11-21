<script setup vapor lang="ts">
// Centres content at the page width, with inline padding that grows with the
// space around it. On a landing surface the width is capped
// (--weft-page-max, 1200 px in the shipped themes); in an app it runs full
// width, as the surface's density says.
withDefaults(
  defineProps<{
    /** The element to render, for semantics: `section`, `header`, `footer`. */
    as?: string
  }>(),
  { as: 'div' },
)
</script>

<template>
  <component :is="as" class="weft-container"><slot /></component>
</template>
