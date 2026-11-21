<script setup vapor lang="ts">
import { Stack, Toast } from '@weft/ui'
</script>

<template>
  <!-- Toasts normally come from useToast() through a Toaster; these stay (duration 0). -->
  <Stack :gap="2" role="status">
    <Toast title="Saved" description="Your draft is safe." tone="success" icon="circle-check" :duration="0" />
    <Toast title="Message deleted" :actions="[{ label: 'Undo' }]" :duration="0" />
  </Stack>
</template>
