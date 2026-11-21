<script setup vapor lang="ts">
import { Timeline } from '@weft/ui'

const releases = [
  { date: 'Jan', title: 'Alpha' },
  { date: 'Apr', title: 'Beta' },
  { date: 'Jul', title: 'Release candidate' },
  { date: 'Oct', title: '1.0' },
]
</script>

<template>
  <Timeline :items="releases" orientation="horizontal" :model-value="2" tone="success" />
</template>
