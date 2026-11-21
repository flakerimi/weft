<script setup vapor lang="ts">
import { ref } from 'vue'
import { RadioGroup } from '@weft/ui'

const plan = ref<string | number | null>('team')
const plans = [
  { value: 'solo', label: 'Solo', description: 'One person.' },
  { value: 'team', label: 'Team', description: 'Up to 20 people.' },
  { value: 'company', label: 'Company', description: 'Ask us.', disabled: true },
]
</script>

<template>
  <RadioGroup v-model="plan" legend="Plan" :items="plans" />
</template>
