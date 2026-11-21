<script setup vapor lang="ts">
import { ref } from 'vue'
import { Calendar } from '@weft/ui'
import type { CalendarDate } from '@internationalized/date'

// Several dates, in German (Monday first), weekends not choosable, with ISO week numbers.
const days = ref<string[]>(['2026-11-24', '2026-11-26'])
const weekend = (date: CalendarDate) => date.toDate('UTC').getUTCDay() % 6 === 0
</script>

<template>
  <Calendar v-model="days" multiple locale="de-DE" week-numbers :is-date-disabled="weekend" />
</template>
