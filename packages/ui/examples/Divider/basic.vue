<script setup vapor lang="ts">
import { Divider, Stack } from '@weft/ui'
</script>

<template>
  <Stack :gap="3">
    <p>Signed in as ada@example.com</p>
    <Divider />
    <Stack direction="row" :gap="3" align="center">
      <span>Profile</span>
      <Divider orientation="vertical" />
      <span>Sign out</span>
    </Stack>
  </Stack>
</template>
