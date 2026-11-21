<script setup vapor lang="ts">
import { ref } from 'vue'
import { Progress, Stack } from '@weft/ui'

const uploaded = ref(64)
</script>

<template>
  <Stack :gap="4">
    <Progress v-model="uploaded" label="Uploading report.pdf" status />
    <!-- No value: indeterminate. -->
    <Progress label="Preparing the export" tone="neutral" size="sm" />
  </Stack>
</template>
