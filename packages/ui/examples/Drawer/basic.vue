<script setup vapor lang="ts">
import { Button, Drawer, Stack } from '@weft/ui'
</script>

<template>
  <!-- Drag the handle down to close it. -->
  <Drawer title="Share" description="Send a link to this page.">
    <Button label="Share" icon="share" />
    <template #body>
      <Stack :gap="2">
        <Button label="Copy link" variant="outline" tone="neutral" block />
        <Button label="Email" variant="outline" tone="neutral" block />
      </Stack>
    </template>
  </Drawer>
</template>
