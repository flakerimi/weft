<script setup vapor lang="ts">
import { ref } from 'vue'
import { DatePicker, FormField } from '@weft/ui'

const day = ref<string | null>('2026-12-03')
</script>

<template>
  <FormField label="Day" name="day">
    <DatePicker v-model="day" locale="en-GB" />
  </FormField>
</template>
