<script setup vapor lang="ts">
import { ref } from 'vue'
import { FormField, InputNumber } from '@weft/ui'

const seats = ref<number | null>(3)
</script>

<template>
  <FormField label="Seats" description="Between 1 and 50.">
    <InputNumber v-model="seats" :min="1" :max="50" />
  </FormField>
</template>
