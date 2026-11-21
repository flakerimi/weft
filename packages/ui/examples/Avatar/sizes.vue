<script setup vapor lang="ts">
import { Avatar, Stack } from '@weft/ui'
</script>

<template>
  <Stack direction="row" :gap="2" align="center">
    <Avatar alt="Alan Turing" size="sm" />
    <Avatar alt="Alan Turing" size="md" />
    <Avatar alt="Alan Turing" size="lg" />
    <Avatar text="AI" icon="sparkles" size="lg" />
  </Stack>
</template>
