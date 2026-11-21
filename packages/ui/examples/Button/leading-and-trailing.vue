<script setup vapor lang="ts">
import { ref } from 'vue'
import { Badge, Button, Stack } from '@weft/ui'

const view = ref<'list' | 'grid'>('list')
</script>

<template>
  <Stack direction="row" :gap="2" align="center">
    <Button label="Inbox" variant="outline" tone="neutral" icon="inbox">
      <template #trailing><Badge :label="3" size="sm" /></template>
    </Button>
    <!-- active draws the current one of a set. -->
    <Button label="List" variant="ghost" tone="neutral" :active="view === 'list'" @click="view = 'list'" />
    <Button label="Grid" variant="ghost" tone="neutral" :active="view === 'grid'" @click="view = 'grid'" />
  </Stack>
</template>
