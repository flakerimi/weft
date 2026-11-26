<script lang="ts">
// Built once, as a shell, and served for every /products/<id>.
export const runtime = true
</script>

<script setup vapor lang="ts">
defineProps<{ id: string }>()
</script>

<template>
  <main><h1>Product [{{ id }}]</h1></main>
</template>
