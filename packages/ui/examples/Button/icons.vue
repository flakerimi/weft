<script setup vapor lang="ts">
import { Button, Stack } from '@weft/ui'
</script>

<template>
  <Stack direction="row" :gap="2" align="center">
    <Button label="New invoice" icon="plus" />
    <Button label="Continue" variant="outline" trailing-icon="arrow-right" />
    <!-- An icon alone makes a square button; it needs an aria-label. -->
    <Button icon="settings" variant="ghost" tone="neutral" aria-label="Settings" />
  </Stack>
</template>
