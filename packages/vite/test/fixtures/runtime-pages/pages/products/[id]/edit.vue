<script lang="ts">
export const runtime = true
</script>

<script setup vapor lang="ts">
defineProps<{ id: string }>()
</script>

<template>
  <main><h1>Edit product [{{ id }}]</h1></main>
</template>
