<script setup vapor lang="ts">
import { ref } from 'vue'
import { ErrorState } from '@weft/ui'

const retrying = ref(false)
function retry() {
  retrying.value = true
  setTimeout(() => (retrying.value = false), 1200)
}
</script>

<template>
  <ErrorState
    title="Customers could not be loaded"
    :error="{ error: { code: 'unavailable', message: 'The server did not answer in time.' } }"
    retry
    :loading="retrying"
    @retry="retry"
  />
</template>
