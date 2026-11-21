<script setup vapor lang="ts">
import { ref } from 'vue'
import { FormField, Select } from '@weft/ui'

const status = ref('open')
const statuses = [
  { label: 'Open', value: 'open' },
  { label: 'In progress', value: 'doing' },
  { label: 'Blocked', value: 'blocked', description: 'Waiting on someone else' },
  { label: 'Done', value: 'done' },
]
</script>

<template>
  <!-- On touch the list opens as a bottom sheet with a search field at the top. -->
  <FormField label="Status" name="status">
    <Select v-model="status" :items="statuses" placeholder="Choose a status" />
  </FormField>
</template>
