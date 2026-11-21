<script setup vapor lang="ts">
// Every overlay component in one island: the server renders their triggers
// and none of their content (not even the popover that starts open); the
// browser hydrates that markup in place, then opens overlays into
// #weft-overlays (overlays-island.test.ts).
import { ref } from 'vue'
import Button from '@weft/ui/basics/Button.vue'
import ConfirmDialog from '@weft/ui/overlays/ConfirmDialog.vue'
import ContextMenu from '@weft/ui/overlays/ContextMenu.vue'
import Dialog from '@weft/ui/overlays/Dialog.vue'
import Drawer from '@weft/ui/overlays/Drawer.vue'
import DropdownMenu from '@weft/ui/overlays/DropdownMenu.vue'
import Popover from '@weft/ui/overlays/Popover.vue'
import Sheet from '@weft/ui/overlays/Sheet.vue'
import Toaster from '@weft/ui/overlays/Toaster.vue'
import Tooltip from '@weft/ui/overlays/Tooltip.vue'
import type { MenuItem } from '@weft/ui'

const chosen = ref('')
const items: MenuItem[] = [{ label: 'Archive', onSelect: () => (chosen.value = 'archive') }]
</script>

<template>
  <div class="overlays">
    <Dialog title="Rename">
      <Button class="open-dialog" label="Rename" />
      <template #footer="{ close }"><Button class="done" label="Done" @click="close()" /></template>
    </Dialog>
    <Sheet title="Settings"><button class="open-sheet">Settings</button></Sheet>
    <Drawer title="Share"><button class="open-drawer">Share</button></Drawer>
    <ConfirmDialog title="Delete?"><button class="open-confirm">Delete</button></ConfirmDialog>
    <Popover title="Filters" default-open>
      <button class="open-popover">Filters</button>
      <template #content><input class="field" /></template>
    </Popover>
    <Tooltip text="Help"><button class="help">?</button></Tooltip>
    <DropdownMenu :items="items"><button class="open-menu">More</button></DropdownMenu>
    <ContextMenu :items="items" aria-label="Actions"><p class="area">Area</p></ContextMenu>
    <Toaster />
    <output>{{ chosen }}</output>
  </div>
</template>
