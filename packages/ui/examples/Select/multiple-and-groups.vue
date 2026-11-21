<script setup vapor lang="ts">
import { ref } from 'vue'
import { FormField, Select, type SelectItems } from '@weft/ui'

const days = ref<string[]>(['mon', 'wed'])
const week: SelectItems = [
  [
    { type: 'label', label: 'Weekdays' },
    { label: 'Monday', value: 'mon' },
    { label: 'Tuesday', value: 'tue' },
    { label: 'Wednesday', value: 'wed' },
    { label: 'Thursday', value: 'thu' },
    { label: 'Friday', value: 'fri' },
  ],
  [
    { type: 'label', label: 'Weekend' },
    { label: 'Saturday', value: 'sat' },
    { label: 'Sunday', value: 'sun' },
  ],
]
</script>

<template>
  <FormField label="Office days" hint="Several">
    <Select v-model="days" :items="week" multiple placeholder="No days" />
  </FormField>
</template>
