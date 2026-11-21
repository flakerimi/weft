<script setup vapor lang="ts">
import { Badge, Stack } from '@weft/ui'
</script>

<template>
  <Stack :gap="3">
    <Stack direction="row" :gap="2" align="center">
      <Badge label="Solid" />
      <Badge label="Soft" variant="soft" />
      <Badge label="Outline" variant="outline" />
      <Badge label="Ghost" variant="ghost" />
    </Stack>
    <Stack direction="row" :gap="2" align="center">
      <Badge label="Small" size="sm" tone="neutral" />
      <Badge label="Medium" size="md" tone="neutral" />
      <Badge label="Large" size="lg" tone="neutral" />
    </Stack>
  </Stack>
</template>
