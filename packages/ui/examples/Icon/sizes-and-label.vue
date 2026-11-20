<script setup vapor lang="ts">
import { Icon, Stack } from '@weft/ui'
</script>

<template>
  <Stack direction="row" :gap="3" align="center">
    <Icon name="star" size="sm" />
    <Icon name="star" size="md" />
    <Icon name="star" size="lg" />
    <!-- Standing alone, the icon needs a label. -->
    <Icon name="circle-alert" label="Payment failed" />
  </Stack>
</template>
