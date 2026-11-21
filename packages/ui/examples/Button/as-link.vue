<script setup vapor lang="ts">
import { Button, Stack } from '@weft/ui'
</script>

<template>
  <Stack direction="row" :gap="2" wrap>
    <Button to="/pricing" label="See pricing" trailing-icon="arrow-right" />
    <Button to="https://vuejs.org/" label="Vue docs" variant="outline" trailing-icon="external-link" />
    <Button block to="/signup" label="Create an account" variant="soft" />
  </Stack>
</template>
