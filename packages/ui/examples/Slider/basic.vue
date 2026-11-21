<script setup vapor lang="ts">
import { ref } from 'vue'
import { FormField, Slider } from '@weft/ui'

const volume = ref<number | number[]>(60)
</script>

<template>
  <FormField label="Volume">
    <Slider v-model="volume" />
  </FormField>
</template>
