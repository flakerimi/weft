<script setup vapor lang="ts">
import { Button, Sheet, Stack } from '@weft/ui'

const sides = ['start', 'end', 'top', 'bottom'] as const
</script>

<template>
  <Stack direction="row" :gap="2">
    <Sheet v-for="side in sides" :key="side" :side="side" :title="`From ${side}`">
      <Button :label="side" variant="outline" tone="neutral" />
      <template #body><p>A sheet from the {{ side }} side.</p></template>
    </Sheet>
  </Stack>
</template>
