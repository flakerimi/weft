<script setup vapor lang="ts">
import { ref } from 'vue'
import { Button, Checkbox, Popover, Stack } from '@weft/ui'

const open = ref(false)
const archived = ref(false)
</script>

<template>
  <!-- On touch the same content opens as a bottom sheet, titled "Filters". -->
  <Popover v-model:open="open" title="Filters">
    <Button variant="outline" tone="neutral" label="Filters" trailing-icon="chevron-down" />
    <template #content="{ close }">
      <Stack :gap="3">
        <Checkbox v-model="archived" label="Show archived" />
        <Button size="sm" label="Apply" @click="close()" />
      </Stack>
    </template>
  </Popover>
</template>
