<script setup vapor lang="ts">
import { Link, Separator, Stack } from '@weft/ui'
</script>

<template>
  <Stack direction="row" :gap="3" align="center">
    <Link standalone to="/terms">Terms</Link>
    <Separator orientation="vertical" decorative />
    <Link standalone to="/privacy">Privacy</Link>
  </Stack>
</template>
