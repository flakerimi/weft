<script setup vapor lang="ts">
import { ref } from 'vue'

const text = ref('lazy')
async function load() {
  text.value = (await import('weft-test-external')).default
}
</script>

<template>
  <button type="button" @click="load">{{ text }}</button>
</template>
