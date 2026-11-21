<script setup vapor lang="ts">
// Two menus drawn open in place (:portal="false"), one on pointer and one on
// touch, in one island: the server draws them open where they stand; the
// browser hydrates that markup and they stay usable (in-place-island.test.ts).
import { ref } from 'vue'
import Button from '@weft/ui/basics/Button.vue'
import DropdownMenu from '@weft/ui/overlays/DropdownMenu.vue'
import WeftProvider from '@weft/ui/adaptive/WeftProvider.vue'
import type { MenuItem } from '@weft/ui'

const chosen = ref('')
const items = (where: string): MenuItem[] => [{ label: 'Open', onSelect: () => (chosen.value = `${where} open`) }, { label: 'Reboot', onSelect: () => (chosen.value = `${where} reboot`) }]
</script>

<template>
  <div class="in-place">
    <p class="chosen">{{ chosen }}</p>
    <WeftProvider input="pointer">
      <div class="pointer"><DropdownMenu :items="items('pointer')" :portal="false" default-open><Button class="open-pointer" label="Pointer" /></DropdownMenu></div>
    </WeftProvider>
    <WeftProvider input="touch">
      <div class="touch"><DropdownMenu :items="items('touch')" :portal="false" default-open><Button class="open-touch" label="Touch" /></DropdownMenu></div>
    </WeftProvider>
  </div>
</template>
