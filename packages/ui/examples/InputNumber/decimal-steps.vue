<script setup vapor lang="ts">
import { ref } from 'vue'
import { FormField, InputNumber } from '@weft/ui'

const rate = ref<number | null>(1.5)
</script>

<template>
  <!-- wheel lets the mouse wheel step it, only while it has focus. -->
  <FormField label="Rate">
    <InputNumber v-model="rate" :step="0.25" :min="0" wheel />
  </FormField>
</template>
