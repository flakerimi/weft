<script setup vapor lang="ts">
import { Button, Stack } from '@weft/ui'
</script>

<template>
  <Stack direction="row" :gap="2">
    <Button label="Save changes" />
    <Button label="Cancel" variant="ghost" tone="neutral" />
  </Stack>
</template>
