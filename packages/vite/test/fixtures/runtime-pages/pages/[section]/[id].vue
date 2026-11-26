<script lang="ts">
export const runtime = true
</script>

<script setup vapor lang="ts">
defineProps<{ section: string; id: string }>()
</script>

<template>
  <main><h1>Section [{{ section }}] [{{ id }}]</h1></main>
</template>
