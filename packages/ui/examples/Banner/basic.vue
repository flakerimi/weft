<script setup vapor lang="ts">
import { Banner } from '@weft/ui'
</script>

<template>
  <Banner
    icon="info"
    title="Scheduled maintenance on Saturday, 02:00 to 04:00."
    :actions="[{ label: 'Details', to: '/status' }]"
    close
  />
</template>
