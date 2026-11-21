<script setup vapor lang="ts">
import { ref } from 'vue'
import { FormField, InputTime } from '@weft/ui'

const lap = ref<string | null>('00:01:30')
</script>

<template>
  <FormField label="Lap time">
    <InputTime v-model="lap" granularity="second" :hour-cycle="24" />
  </FormField>
</template>
