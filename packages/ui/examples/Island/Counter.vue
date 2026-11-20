<script setup vapor lang="ts">
// A helper for the Island examples: the interactive component the island hydrates.
import { ref } from 'vue'
const props = defineProps<{ start: number }>()
const n = ref(props.start)
</script>

<template>
  <button type="button" @click="n++">Clicked {{ n }} times</button>
</template>
