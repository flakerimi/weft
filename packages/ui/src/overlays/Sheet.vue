<script setup vapor lang="ts">
// A panel that slides over the page from one side (a slideover), or up from
// the bottom (a bottom sheet): a modal dialog (WAI-ARIA APG, Dialog
// (Modal)) like Dialog, opened from its trigger or with v-model:open, focus
// trapped inside, the page behind still and inert. Escape, the backdrop and
// the close button close it, and focus returns to the trigger; it slides
// back out unless motion is reduced. `start` and `end` follow the text
// direction. Popover and DropdownMenu take this form on touch.
import { overlayRoot } from './overlay.ts'
import { useModal } from './modal.ts'
import OverlayPanel from './OverlayPanel.vue'
import '../styles/base.css'
import '../styles/overlay.css'
import '../styles/sheet.css'

defineOptions({ inheritAttrs: false })

const props = withDefaults(
  defineProps<{
    /** Open or closed, bound with `v-model:open`. */
    open?: boolean
    /** Whether it starts open when `v-model:open` is not bound. */
    defaultOpen?: boolean
    /** The heading, which names the sheet. Without one or a header slot, give the sheet an `aria-label`. */
    title?: string
    /** A sentence under the title, which describes the sheet. */
    description?: string
    /** Show the close button (×) in the header. */
    close?: boolean
    /** The close button's name for assistive technology. */
    closeLabel?: string
    /** Escape and a press on the backdrop close it. */
    dismissible?: boolean
    /** Dim the page with a backdrop; without one it is still modal. */
    overlay?: boolean
    /** The side it comes from: `start` and `end` (the inline sides, mirrored right to left), `top` or `bottom`. */
    side?: 'start' | 'end' | 'top' | 'bottom'
  }>(),
  { open: undefined, defaultOpen: false, close: true, closeLabel: 'Close', dismissible: true, overlay: true, side: 'end' },
)

const emit = defineEmits<{
  /** Opened or closed, by the trigger, Escape, the backdrop or a close. */
  'update:open': [open: boolean]
}>()

defineSlots<{
  /** The trigger: one element (a Button), which opens it. */
  default?: (props: { open: boolean }) => unknown
  /** The header's heading, in place of `title` and `description`; it names the sheet. */
  header?: () => unknown
  /** Controls in the header, before the close button. */
  actions?: () => unknown
  /** The sheet's content, which scrolls when long; `close()` closes it. */
  body?: (props: { close: () => void }) => unknown
  /** Below the body, for the sheet's buttons; `close()` closes it. */
  footer?: (props: { close: () => void }) => unknown
}>()

const m = useModal(props, emit, { prefix: 'sheet' })
const open = m.open
</script>

<template>
  <span v-if="$slots.default" ref="wrap" class="weft-overlay-trigger" @click="m.onTrigger"><slot :open="open" /></span>
  <Teleport v-if="m.mounted.value && m.present.value" :to="overlayRoot()">
    <div
      class="weft-overlay"
      v-bind="m.context.attrs.value"
      :data-state="m.closing.value ? 'closed' : 'open'"
    >
      <div v-if="overlay" class="weft-overlay-backdrop" @click="m.onBackdrop"></div>
      <div
        ref="panel"
        v-bind="$attrs"
        :id="m.ids.panel"
        class="weft-sheet"
        role="dialog"
        aria-modal="true"
        :aria-labelledby="title || $slots.header ? m.ids.title : undefined"
        :aria-describedby="description ? m.ids.description : undefined"
        :data-side="side"
      >
        <OverlayPanel
          :title="title"
          :description="description"
          :title-id="m.ids.title"
          :description-id="m.ids.description"
          :close="close"
          :close-label="closeLabel"
          @close="m.close"
        >
          <template v-if="$slots.header" #header><slot name="header" /></template>
          <template v-if="$slots.actions" #actions><slot name="actions" /></template>
          <template v-if="$slots.body" #body="{ close: hide }"><slot name="body" :close="hide" /></template>
          <template v-if="$slots.footer" #footer="{ close: hide }"><slot name="footer" :close="hide" /></template>
        </OverlayPanel>
      </div>
    </div>
  </Teleport>
</template>
