<script setup vapor lang="ts">
import { ref } from 'vue'
import { Button, FieldGroup, Input } from '@weft/ui'

const email = ref('')
</script>

<template>
  <FieldGroup aria-label="Subscribe">
    <Input v-model="email" type="email" placeholder="you@example.com" aria-label="Email" />
    <Button label="Subscribe" />
  </FieldGroup>
</template>
