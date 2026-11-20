<script setup vapor lang="ts">
import { Card, WeftProvider } from '@weft/ui'
</script>

<template>
  <!-- A kiosk or a preview: every component inside behaves as on a touch screen. -->
  <WeftProvider input="touch">
    <Card>Large hit targets everywhere in here.</Card>
  </WeftProvider>
</template>
