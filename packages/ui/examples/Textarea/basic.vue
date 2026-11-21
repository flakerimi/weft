<script setup vapor lang="ts">
import { ref } from 'vue'
import { FormField, Textarea } from '@weft/ui'

const message = ref('')
</script>

<template>
  <FormField label="Message" description="It grows as you write, up to eight lines.">
    <Textarea v-model="message" autoresize :rows="2" :maxrows="8" placeholder="How can we help?" />
  </FormField>
</template>
