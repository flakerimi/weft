<script setup vapor lang="ts">
import { useWeft } from '../../src/adaptive/context.ts'
import { seen } from './probe-log.ts'
const { input, surface } = useWeft()
seen.push(input.value)
</script>

<template>
  <span class="probe" :data-input="input" :data-surface="surface"></span>
</template>
