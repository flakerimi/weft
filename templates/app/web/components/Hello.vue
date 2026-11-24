<script setup vapor lang="ts">
// An island: the server draws "Loading", and the browser asks the Go API.
import { onMounted, ref } from 'vue'

const message = ref('Loading')

onMounted(async () => {
  try {
    const res = await fetch('/api/hello')
    message.value = ((await res.json()) as { message: string }).message
  } catch {
    message.value = 'The API did not answer'
  }
})
</script>

<template>
  <Badge tone="success" variant="soft" :label="message" />
</template>
