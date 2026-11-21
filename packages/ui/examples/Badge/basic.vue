<script setup vapor lang="ts">
import { Badge, Stack } from '@weft/ui'
</script>

<template>
  <Stack direction="row" :gap="2" align="center">
    <Badge label="Paid" tone="success" variant="soft" icon="check" />
    <Badge label="Overdue" tone="danger" variant="soft" />
    <Badge label="Beta" variant="outline" />
    <Badge :label="12" />
  </Stack>
</template>
