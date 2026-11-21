<script setup vapor lang="ts">
import { ref } from 'vue'
import { Alert, Stack } from '@weft/ui'

const open = ref(true)
</script>

<template>
  <Stack :gap="3">
    <Alert
      v-model:open="open"
      tone="danger"
      icon="circle-alert"
      title="Payment failed"
      description="The card ending 4242 was declined. Update it to keep your plan."
      :actions="[{ label: 'Update card' }]"
      close
    />
    <Alert tone="success" variant="outline" orientation="horizontal" title="Backup finished" description="All 1,284 records were saved." :actions="[{ label: 'View' }]" />
    <Alert tone="warning" variant="solid" icon="info" title="Your trial ends in 3 days." />
  </Stack>
</template>
