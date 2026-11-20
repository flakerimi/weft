<script setup vapor lang="ts">
import { Stack } from '@weft/ui'
</script>

<template>
  <Stack direction="row" :gap="2" align="center" justify="between" wrap>
    <strong>3 selected</strong>
    <Stack direction="row" :gap="2">
      <button type="button">Archive</button>
      <button type="button">Delete</button>
    </Stack>
  </Stack>
</template>
