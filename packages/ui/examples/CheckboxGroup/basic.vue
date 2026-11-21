<script setup vapor lang="ts">
import { ref } from 'vue'
import { CheckboxGroup } from '@weft/ui'

const channels = ref<(string | number)[]>(['email'])
const items = [
  { value: 'email', label: 'Email' },
  { value: 'sms', label: 'Text message', description: 'Charges may apply.' },
  { value: 'push', label: 'Push notification' },
]
</script>

<template>
  <CheckboxGroup v-model="channels" legend="Notify me by" :items="items" />
</template>
