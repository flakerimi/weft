<script setup vapor lang="ts">
import { ref } from 'vue'
import { FormField, InputTime, type TimeRange } from '@weft/ui'

// Opening hours: submits hours[start] and hours[end]; an end before the start is marked invalid.
const hours = ref<TimeRange | null>({ start: '09:00', end: '17:30' })
</script>

<template>
  <FormField label="Opening hours" name="hours">
    <InputTime v-model="hours" range :step="{ minute: 15 }" />
  </FormField>
</template>
