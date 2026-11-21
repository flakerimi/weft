<script setup vapor lang="ts">
import { Button, Stack } from '@weft/ui'
</script>

<template>
  <Stack direction="row" :gap="2" align="center" justify="between" wrap>
    <strong>3 selected</strong>
    <Stack direction="row" :gap="2">
      <Button label="Archive" size="sm" variant="outline" tone="neutral" />
      <Button label="Delete" size="sm" variant="soft" tone="danger" />
    </Stack>
  </Stack>
</template>
