<script setup vapor lang="ts">
import { ref } from 'vue'

const props = defineProps<{ start: number }>()
const n = ref(props.start)
</script>

<template>
  <Button class="counter" @click="n++">n is {{ n }}</Button>
</template>

<style>
.counter { min-inline-size: 12ch; }
</style>
