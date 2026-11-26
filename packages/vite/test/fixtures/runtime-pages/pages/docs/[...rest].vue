<script lang="ts">
export const runtime = true
</script>

<script setup vapor lang="ts">
defineProps<{ rest: string }>()
</script>

<template>
  <main><h1>Docs [{{ rest }}]</h1></main>
</template>
