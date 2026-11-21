<script setup vapor lang="ts">
import { ref } from 'vue'
import { DatePicker, FormField } from '@weft/ui'
import type { CalendarDate } from '@internationalized/date'

// Only weekdays in 2026 and 2027; a date typed outside them is marked invalid.
const meeting = ref<string | null>(null)
const weekend = (date: CalendarDate) => date.toDate('UTC').getUTCDay() % 6 === 0
</script>

<template>
  <FormField label="Meeting" name="meeting" description="A weekday in 2026 or 2027">
    <DatePicker v-model="meeting" min="2026-01-01" max="2027-12-31" :is-date-disabled="weekend" required />
  </FormField>
</template>
