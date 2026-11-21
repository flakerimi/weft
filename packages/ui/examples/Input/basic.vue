<script setup vapor lang="ts">
import { ref } from 'vue'
import { FormField, Input } from '@weft/ui'

const name = ref('')
</script>

<template>
  <FormField label="Full name">
    <Input v-model="name" placeholder="Ada Lovelace" autocomplete="name" />
  </FormField>
</template>
