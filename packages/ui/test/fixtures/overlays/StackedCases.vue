<script setup vapor lang="ts">
// A dialog opening a menu whose item has a tooltip: three overlays stacked,
// the real components. Plus a popover and a tooltip outside, for the live
// switch between pointer and touch.
import { ref } from 'vue'
import Button from '../../../src/basics/Button.vue'
import Dialog from '../../../src/overlays/Dialog.vue'
import DropdownMenu from '../../../src/overlays/DropdownMenu.vue'
import Popover from '../../../src/overlays/Popover.vue'
import Tooltip from '../../../src/overlays/Tooltip.vue'
import type { MenuItem } from '../../../src/overlays/menu.ts'

const log = ref<string[]>([])
const details = ref(false)
const actions: MenuItem[] = [{ label: 'Details', onSelect: () => (details.value = true) }]
const items: MenuItem[] = [
  { label: 'Archive', onSelect: () => log.value.push('archive') },
  { label: 'Export', onSelect: () => log.value.push('export') },
]
defineExpose({ log })
</script>

<template>
  <Dialog title="Project">
    <Button class="open-dialog" label="Project" />
    <template #body>
      <DropdownMenu :items="items">
        <button class="open-menu">More</button>
        <template #item="{ item }">
          <Tooltip v-if="item.label === 'Export'" text="Download as CSV" :delay="0"><span class="tip-trigger">{{ item.label }}</span></Tooltip>
          <template v-else>{{ item.label }}</template>
        </template>
      </DropdownMenu>
    </template>
  </Dialog>
  <Popover title="Filters">
    <button class="open-popover">Filters</button>
    <template #content><input class="field" /></template>
  </Popover>
  <Tooltip text="Help" :delay="0"><button class="help">?</button></Tooltip>
  <DropdownMenu :items="actions"><button class="open-actions">Actions</button></DropdownMenu>
  <Dialog v-model:open="details" title="Details"><template #body><button class="inside">ok</button></template></Dialog>
  <button class="elsewhere">elsewhere</button>
</template>
