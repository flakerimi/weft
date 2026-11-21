<script setup vapor lang="ts">
import { ref } from 'vue'
import { Calendar, type DateRange } from '@weft/ui'

// A billing period of whole months: from the 1st of the first to the last day of the last.
const period = ref<DateRange | null>({ start: '2026-01-01', end: '2026-03-31' })
</script>

<template>
  <Calendar v-model="period" type="month" range aria-label="Billing period" />
</template>
