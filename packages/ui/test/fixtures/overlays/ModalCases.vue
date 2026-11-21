<script setup vapor lang="ts">
import { ref } from 'vue'
import Button from '../../../src/basics/Button.vue'
import ConfirmDialog from '../../../src/overlays/ConfirmDialog.vue'
import Dialog from '../../../src/overlays/Dialog.vue'
import Drawer from '../../../src/overlays/Drawer.vue'
import Sheet from '../../../src/overlays/Sheet.vue'

const props = withDefaults(
  defineProps<{
    case: 'dialog' | 'sheet' | 'drawer' | 'confirm'
    dismissible?: boolean
    fullscreen?: boolean
    side?: 'start' | 'end' | 'top' | 'bottom'
    work?: 'none' | 'resolve' | 'reject'
    overlay?: boolean
    scrollable?: boolean
    header?: boolean
  }>(),
  { dismissible: true, work: 'none', overlay: true },
)
const open = ref(false)
const confirms = ref(0)
const cancels = ref(0)
const errors = ref<unknown[]>([])
let settleWork: (() => void) | undefined
function onConfirm() {
  confirms.value++
  if (props.work === 'none') return
  return new Promise<void>((resolve, reject) => {
    settleWork = props.work === 'resolve' ? resolve : () => reject(new Error('no'))
  })
}
defineExpose({ open, confirms, cancels, errors, finish: () => settleWork?.() })
</script>

<template>
  <button class="before">before</button>
  <Dialog
    v-if="$props.case === 'dialog'"
    v-model:open="open"
    title="Rename file"
    description="Pick a new name."
    :dismissible="dismissible"
    :fullscreen="fullscreen"
    :overlay="overlay"
    :scrollable="scrollable"
  >
    <Button class="trigger" label="Rename" />
    <template v-if="header" #header><h2 class="mine">My heading</h2></template>
    <template #body><input class="field" /></template>
    <template #footer="{ close }"><button class="cancel" @click="close()">Cancel</button><button class="save">Save</button></template>
  </Dialog>
  <Sheet v-if="$props.case === 'sheet'" v-model:open="open" title="Settings" :side="side">
    <button class="trigger">Settings</button>
    <template #body><input class="field" /></template>
  </Sheet>
  <Drawer v-if="$props.case === 'drawer'" v-model:open="open" title="Share">
    <button class="trigger">Share</button>
    <template #body><input class="field" /></template>
  </Drawer>
  <ConfirmDialog
    v-if="$props.case === 'confirm'"
    v-model:open="open"
    title="Delete the file?"
    description="It cannot be undone."
    confirm-label="Delete"
    tone="danger"
    @confirm="onConfirm"
    @cancel="cancels++"
    @error="(e: unknown) => errors.push(e)"
  >
    <button class="trigger">Delete</button>
  </ConfirmDialog>
  <button class="after">after</button>
</template>
