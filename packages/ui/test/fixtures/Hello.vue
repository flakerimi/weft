<script setup vapor lang="ts">
import { ref } from 'vue'
const props = defineProps<{ name: string }>()
const count = ref(1)
</script>

<template>
  <p class="hello" @click="count++">Hello {{ props.name }} {{ count }}</p>
</template>
