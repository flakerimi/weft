<script setup vapor lang="ts">
import { ref } from 'vue'
import { FormField, RadioGroup } from '@weft/ui'

const size = ref<string | number | null>(null)
</script>

<template>
  <FormField label="Size" required>
    <RadioGroup v-model="size" orientation="horizontal" :items="['S', 'M', 'L', 'XL']" />
  </FormField>
</template>
