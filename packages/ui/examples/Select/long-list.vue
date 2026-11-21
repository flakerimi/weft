<script setup vapor lang="ts">
import { ref } from 'vue'
import { FormField, Select } from '@weft/ui'

// 10,000 options: only the rows in view are drawn, and the keyboard still
// reaches every one (End goes to the last, typing jumps).
const row = ref(5_000)
const rows = Array.from({ length: 10_000 }, (_, i) => ({ label: `Row ${i + 1}`, value: i }))
</script>

<template>
  <FormField label="Row">
    <Select v-model="row" :items="rows" />
  </FormField>
</template>
