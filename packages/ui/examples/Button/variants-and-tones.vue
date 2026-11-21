<script setup vapor lang="ts">
import { Button, Stack } from '@weft/ui'
</script>

<template>
  <Stack :gap="3">
    <Stack direction="row" :gap="2" wrap>
      <Button label="Solid" />
      <Button label="Soft" variant="soft" />
      <Button label="Outline" variant="outline" />
      <Button label="Ghost" variant="ghost" />
    </Stack>
    <Stack direction="row" :gap="2" wrap>
      <Button label="Neutral" tone="neutral" />
      <Button label="Primary" tone="primary" />
      <Button label="Delete" tone="danger" icon="trash" />
      <Button label="Review" tone="warning" />
      <Button label="Approve" tone="success" icon="check" />
    </Stack>
    <Stack direction="row" :gap="2" align="center">
      <Button label="Small" size="sm" />
      <Button label="Medium" size="md" />
      <Button label="Large" size="lg" />
    </Stack>
  </Stack>
</template>
