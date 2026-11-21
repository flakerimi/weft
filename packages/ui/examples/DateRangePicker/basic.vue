<script setup vapor lang="ts">
import { ref } from 'vue'
import { DateRangePicker, FormField, type DateRange } from '@weft/ui'

// Two months on pointer, one in the touch sheet; submits stay[start] and stay[end].
const stay = ref<DateRange | null>({ start: '2026-12-20', end: '2026-12-27' })
</script>

<template>
  <FormField label="Stay" name="stay">
    <DateRangePicker v-model="stay" min="2026-11-21" />
  </FormField>
</template>
