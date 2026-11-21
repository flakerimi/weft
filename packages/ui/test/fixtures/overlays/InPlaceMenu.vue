<script setup vapor lang="ts">
import { ref } from 'vue'
import { Button, DropdownMenu, WeftProvider, type MenuItem } from '../../../src/index.ts'
import type { Input } from '../../../src/adaptive/detect.ts'

const props = defineProps<{ defaultOpen?: boolean; input?: Input }>()
const chosen = ref('')
const items: MenuItem[] = [{ label: 'Open device', onSelect: () => (chosen.value = 'open') }, { label: 'Reboot', onSelect: () => (chosen.value = 'reboot') }]
defineExpose({ chosen })
</script>

<template>
  <WeftProvider :input="props.input">
    <div class="before"><button class="elsewhere">Elsewhere</button></div>
    <div class="place">
      <DropdownMenu :items="items" :portal="false" :default-open="defaultOpen"><Button class="trigger">Actions</Button></DropdownMenu>
    </div>
  </WeftProvider>
</template>
