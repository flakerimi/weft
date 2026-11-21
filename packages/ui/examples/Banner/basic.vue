<script setup vapor lang="ts">
import { Banner } from '@weft/ui'
</script>

<template>
  <Banner
    id="maintenance-2026-11"
    icon="info"
    title="Scheduled maintenance on Saturday, 02:00 to 04:00."
    :actions="[{ label: 'Details', to: '/status' }]"
    close
  />
</template>
