<script setup vapor lang="ts">
import { ref } from 'vue'
import { FormField, Slider } from '@weft/ui'

const price = ref<number | number[]>([20, 80])
</script>

<template>
  <FormField label="Price" :hint="Array.isArray(price) ? `€${price[0]} to €${price[1]}` : ''">
    <Slider v-model="price" :min="0" :max="200" :step="5" :min-steps-between-thumbs="2" />
  </FormField>
</template>
