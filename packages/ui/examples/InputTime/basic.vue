<script setup vapor lang="ts">
import { ref } from 'vue'
import { FormField, InputTime } from '@weft/ui'

// HH:mm; 12-hour in en-US. On touch it is the platform's own time picker.
const opens = ref<string | null>('09:30')
</script>

<template>
  <FormField label="Opens at" name="opens">
    <InputTime v-model="opens" :step="{ minute: 15 }" min="06:00" max="22:00" />
  </FormField>
</template>
