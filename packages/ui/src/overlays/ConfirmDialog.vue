<script setup vapor lang="ts">
// Asks before an action goes ahead: an alert dialog (WAI-ARIA APG, Alert
// and Message Dialogs) with a title, a description, Cancel and a confirm
// button, focus starting on Cancel, the safe answer. Confirm emits `confirm`
// and closes; a confirm handler that returns a promise keeps it open with
// the confirm button loading (and nothing else closing it) until the
// promise settles: it closes when the work succeeds and stays, ready again,
// when it fails. Cancel, Escape and the backdrop emit `cancel` and close.
import { ref } from 'vue'
import Button from '../basics/Button.vue'
import { useControllable } from '../headless/useControllable.ts'
import type { Tone } from '../types.ts'
import Dialog from './Dialog.vue'
import '../styles/base.css'

const props = withDefaults(
  defineProps<{
    /** Open or closed, bound with `v-model:open`. */
    open?: boolean
    /** Whether it starts open when `v-model:open` is not bound. */
    defaultOpen?: boolean
    /** The question, which names the dialog. */
    title: string
    /** What happens if the action goes ahead. */
    description?: string
    /** The confirm button's text: say the action (`Delete`). */
    confirmLabel?: string
    /** The cancel button's text. */
    cancelLabel?: string
    /** The confirm button's colour: `danger` for an action that destroys. */
    tone?: Tone
    /** Run on confirm; a promise it returns keeps the dialog open, loading, until it settles. Given by `@confirm`. */
    onConfirm?: () => unknown
  }>(),
  { open: undefined, defaultOpen: false, confirmLabel: 'Confirm', cancelLabel: 'Cancel', tone: 'primary' },
)

const emit = defineEmits<{
  /** Opened or closed. */
  'update:open': [open: boolean]
  /** Answered no: Cancel, Escape or the backdrop. */
  cancel: []
}>()

defineSlots<{
  /** The trigger: one element (a Button), which opens it. */
  default?: (props: { open: boolean }) => unknown
  /** More content under the description. */
  body?: () => unknown
}>()

const open = useControllable(props, 'open', emit, () => props.defaultOpen)
const busy = ref(false)

function onOpen(next: boolean) {
  if (next) open.value = true
  else if (!busy.value) cancel()
}
function cancel() {
  open.value = false
  emit('cancel')
}
async function confirm() {
  if (busy.value) return
  const work = props.onConfirm?.()
  if (!(work instanceof Promise)) {
    open.value = false
    return
  }
  busy.value = true
  try {
    await work
    open.value = false
  } catch {
    // It failed: stay open, so the person can try again or cancel.
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <Dialog
    :open="open"
    role="alertdialog"
    :title="title"
    :description="description"
    :close="false"
    :dismissible="!busy"
    @update:open="onOpen"
  >
    <template v-if="$slots.default" #default="{ open: shown }"><slot :open="shown" /></template>
    <template v-if="$slots.body" #body><slot name="body" /></template>
    <template #footer>
      <Button variant="outline" tone="neutral" :label="cancelLabel" :disabled="busy" @click="cancel" />
      <Button :tone="tone" :label="confirmLabel" :loading="busy" @click="confirm" />
    </template>
  </Dialog>
</template>
