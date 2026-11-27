<script setup vapor lang="ts">
// Counts its setups in `counts[name]`: what a FormField's label slot holds
// should be set up once.
import { counts } from './setup-counts.ts'

const props = defineProps<{ name: string }>()
counts[props.name] = (counts[props.name] ?? 0) + 1
</script>

<template>
  <span class="setup-count">?</span>
</template>
