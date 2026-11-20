<script lang="ts">
export async function getPaths() {
  return [{ slug: 'first' }, { slug: 'second' }]
}
</script>

<script setup vapor lang="ts">
defineProps<{ slug: string }>()
</script>

<template>
  <main>
    <h1>Post {{ slug }}</h1>
  </main>
</template>
