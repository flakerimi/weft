<script setup vapor lang="ts">
import { ref } from 'vue'
import { useWeft } from '../../src/adaptive/context.ts'
import Probe from './Probe.vue'
const { input } = useWeft()
const late = ref(false)
defineExpose({ show: () => (late.value = true) })
</script>

<template>
  <span class="host" :data-input="input"><Probe v-if="late" /></span>
</template>
