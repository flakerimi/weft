<script setup vapor lang="ts">
import { Input, Stack } from '@weft/ui'
</script>

<template>
  <!-- md lines up with an md Button: 36 px on pointer, 44 px on touch. -->
  <Stack :gap="2">
    <Input size="sm" placeholder="Small" aria-label="Small" />
    <Input placeholder="Medium" aria-label="Medium" />
    <Input size="lg" placeholder="Large" aria-label="Large" />
    <Input placeholder="Disabled" aria-label="Disabled" disabled />
  </Stack>
</template>
