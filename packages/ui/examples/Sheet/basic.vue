<script setup vapor lang="ts">
import { Button, Link, Sheet, Stack } from '@weft/ui'
</script>

<template>
  <Sheet title="Menu" side="start">
    <Button icon="menu" variant="ghost" tone="neutral" aria-label="Open menu" />
    <template #body>
      <Stack :gap="2">
        <Link href="/">Home</Link>
        <Link href="/settings">Settings</Link>
      </Stack>
    </template>
  </Sheet>
</template>
