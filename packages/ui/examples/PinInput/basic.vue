<script setup vapor lang="ts">
import { ref } from 'vue'
import { FormField, PinInput } from '@weft/ui'

const code = ref('')
const done = ref(false)
</script>

<template>
  <FormField label="Verification code" :description="done ? 'Checking…' : 'We sent it to your phone.'">
    <PinInput v-model="code" :length="6" type="number" otp @complete="done = true" />
  </FormField>
</template>
