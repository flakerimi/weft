<script setup vapor lang="ts">
import { ref } from 'vue'
import { Calendar, type DateRange } from '@weft/ui'

// Two months side by side; a first press starts the range, a second ends it.
const nights = ref<DateRange | null>({ start: '2026-12-20', end: '2026-12-27' })
</script>

<template>
  <Calendar v-model="nights" range :number-of-months="2" min="2026-11-21" />
</template>
