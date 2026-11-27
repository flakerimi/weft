<script setup vapor lang="ts">
// An island: the server draws "Loading", and the browser asks the Go API
// through the client weft gen writes from it.
import { onMounted, ref } from 'vue'
import { api } from '../.weft-types/api.ts'

const message = ref('Loading')

onMounted(async () => {
  try {
    message.value = (await api.hello.hello()).message
  } catch {
    message.value = 'The API did not answer'
  }
})
</script>

<template>
  <Badge tone="success" variant="soft" :label="message" />
</template>
