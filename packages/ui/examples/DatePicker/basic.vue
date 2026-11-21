<script setup vapor lang="ts">
import { ref } from 'vue'
import { DatePicker, FormField } from '@weft/ui'

// Type a date as the locale writes it, or choose one; on touch the calendar opens as a bottom sheet.
const due = ref<string | null>('2026-11-30')
</script>

<template>
  <FormField label="Due date" name="due">
    <DatePicker v-model="due" locale="en-GB" />
  </FormField>
</template>
