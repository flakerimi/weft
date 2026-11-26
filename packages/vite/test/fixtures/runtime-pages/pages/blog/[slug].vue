<script lang="ts">
// A page with params known at build sits beside the runtime pages, as before.
export const getPaths = async () => [{ slug: 'first' }]
// Not true, so not a runtime page.
export const runtime = 'no'
</script>

<script setup vapor lang="ts">
defineProps<{ slug: string }>()
</script>

<template>
  <main><h1>Post {{ slug }}</h1></main>
</template>
