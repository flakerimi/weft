<script setup vapor lang="ts">
import { ref } from 'vue'
const props = defineProps<{ start: number }>()
const n = ref(props.start)
</script>

<template>
  <button type="button" @click="n++">n is {{ n }}</button>
</template>
