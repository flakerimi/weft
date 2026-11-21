<script setup vapor lang="ts">
import { Spinner, Stack } from '@weft/ui'
</script>

<template>
  <Stack direction="row" :gap="3" align="center">
    <Spinner label="Loading invoices" />
    <Spinner size="sm" />
    <Spinner size="lg" />
  </Stack>
</template>
