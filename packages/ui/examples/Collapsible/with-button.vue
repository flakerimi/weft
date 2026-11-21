<script setup vapor lang="ts">
import { ref } from 'vue'
import { Button, Collapsible, Stack } from '@weft/ui'

const open = ref(false)
</script>

<template>
  <Collapsible v-model:open="open">
    <Button :label="open ? 'Show fewer' : 'Show all 12 repositories'" variant="outline" tone="neutral" trailing-icon="chevron-down" />
    <template #content>
      <Stack :gap="1">
        <span>weft/ui</span>
        <span>weft/themes</span>
        <span>weft/vite</span>
      </Stack>
    </template>
  </Collapsible>
</template>
