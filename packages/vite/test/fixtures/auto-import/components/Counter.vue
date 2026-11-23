<script setup vapor lang="ts">
import { ref } from 'vue'

const props = defineProps<{ start: number }>()
const count = ref(props.start)
</script>

<template>
  <Badge>{{ count }}</Badge>
</template>
