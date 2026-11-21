<script setup vapor lang="ts">
import { ContextMenu, type MenuItem } from '@weft/ui'

const items: MenuItem[] = [
  { label: 'Open', icon: 'folder-open' },
  { label: 'Rename', icon: 'pencil' },
  { type: 'separator' },
  { label: 'Delete', icon: 'trash', tone: 'danger' },
]
</script>

<template>
  <!-- Right press, the context menu key, or a long press on touch. -->
  <ContextMenu :items="items" aria-label="File actions">
    <div tabindex="0" style="padding: 32px; border: 1px dashed; border-radius: 12px">report.pdf</div>
  </ContextMenu>
</template>
