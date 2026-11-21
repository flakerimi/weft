<script setup vapor lang="ts">
import { ref } from 'vue'
import { FormField, Input, Stack } from '@weft/ui'

const email = ref('')
const username = ref('ada')
</script>

<template>
  <Stack :gap="4">
    <FormField label="Email" description="We never share it." required>
      <Input v-model="email" type="email" autocomplete="email" />
    </FormField>
    <FormField label="Username" hint="Optional" error="That name is taken.">
      <Input v-model="username" />
    </FormField>
  </Stack>
</template>
