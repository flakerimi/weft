<script setup vapor lang="ts">
import { Button, DropdownMenu, type MenuItem } from '../../../src/index.ts'

defineProps<{ defaultOpen?: boolean }>()
const items: MenuItem[] = [{ label: 'Open device' }, { label: 'Reboot' }]
</script>

<template>
  <div class="place">
    <DropdownMenu :items="items" :portal="false" :default-open="defaultOpen"><Button class="trigger">Actions</Button></DropdownMenu>
  </div>
</template>
