<script setup vapor lang="ts">
import { ref } from 'vue'
import { FormField, Listbox } from '@weft/ui'

const plan = ref('team')
const plans = [
  { label: 'Hobby', value: 'hobby', description: 'One project, community support' },
  { label: 'Team', value: 'team', description: 'Unlimited projects, five seats' },
  { label: 'Business', value: 'business', description: 'Single sign-on and audit logs' },
  { label: 'Enterprise', value: 'enterprise', description: 'Talk to us', disabled: true },
]
</script>

<template>
  <FormField label="Plan" name="plan">
    <Listbox v-model="plan" :items="plans" value-key="value" />
  </FormField>
</template>
