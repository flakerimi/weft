<script setup vapor lang="ts">
import { useCollection } from 'virtual:weft/content'

const titles = useCollection('docs').map((p) => p.title)
</script>

<template>
  <ul><li v-for="t in titles" :key="t">{{ t }}</li></ul>
</template>
