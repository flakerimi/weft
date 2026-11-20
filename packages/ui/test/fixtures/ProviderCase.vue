<script setup vapor lang="ts">
import WeftProvider from '../../src/adaptive/WeftProvider.vue'
import Probe from './Probe.vue'
import type { Input, Surface } from '../../src/adaptive/detect.ts'
defineProps<{
  outerInput?: Input
  outerSurface?: Surface
  innerInput?: Input
  innerSurface?: Surface
  nested?: boolean
}>()
</script>

<template>
  <WeftProvider class="outer" :input="outerInput" :surface="outerSurface">
    <Probe v-if="!nested" />
    <WeftProvider v-else class="inner" :input="innerInput" :surface="innerSurface">
      <Probe />
    </WeftProvider>
  </WeftProvider>
</template>
