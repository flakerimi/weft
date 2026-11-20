<script setup vapor lang="ts">
import { useWeft } from '@weft/ui'
const { input, surface } = useWeft()
</script>

<template>
  <p class="readout">useWeft(): input <strong>{{ input }}</strong>, surface <strong>{{ surface }}</strong></p>
</template>
