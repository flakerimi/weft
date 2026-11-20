<script setup vapor lang="ts">
import { useWeft } from '../../src/adaptive/context.ts'
const { input, surface } = useWeft()
</script>

<template>
  <span class="probe" :data-input="input" :data-surface="surface"></span>
</template>
