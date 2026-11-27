<script setup vapor lang="ts">
// A modal dialog (WAI-ARIA APG, Dialog (Modal)): opened from its trigger
// (the default slot) or with v-model:open, centred over a backdrop, focus
// trapped inside, the page behind still and inert. Escape, the backdrop and
// the close button close it, and focus returns to the trigger. Full-screen
// when the viewport is narrow (under 640 px), or always with `fullscreen`.
// It fades out when it closes, unless motion is reduced.
import { computed, onMounted, shallowRef } from 'vue'
import { useWidth } from '../adaptive/useWidth.ts'
import { overlayRoot } from './overlay.ts'
import { useModal } from './modal.ts'
import OverlayPanel from './OverlayPanel.vue'
import '../styles/base.css'
import '../styles/overlay.css'
import '../styles/dialog.css'

defineOptions({ inheritAttrs: false })

const props = withDefaults(
  defineProps<{
    /** Open or closed, bound with `v-model:open`. */
    open?: boolean
    /** Whether it starts open when `v-model:open` is not bound. */
    defaultOpen?: boolean
    /** The heading, which names the dialog. Without one or a header slot, give the dialog an `aria-label`. */
    title?: string
    /** A sentence under the title, which describes the dialog. */
    description?: string
    /** Show the close button (×) in the header. */
    close?: boolean
    /** The close button's name for assistive technology. */
    closeLabel?: string
    /** Escape and a press on the backdrop close it. */
    dismissible?: boolean
    /** Full-screen at every width; it is full-screen when narrow anyway. */
    fullscreen?: boolean
    /** Dim the page with a backdrop; without one it is still modal. */
    overlay?: boolean
    /** The dialog scrolls with its layer, as one long page, instead of within the viewport's height. */
    scrollable?: boolean
    /** `alertdialog` for a dialog that needs an answer before going on (ConfirmDialog uses it). */
    role?: 'dialog' | 'alertdialog'
  }>(),
  { open: undefined, defaultOpen: false, close: true, closeLabel: 'Close', dismissible: true, overlay: true, role: 'dialog' },
)

const emit = defineEmits<{
  /** Opened or closed, by the trigger, Escape, the backdrop or a close. */
  'update:open': [open: boolean]
}>()

defineSlots<{
  /** The trigger: one element (a Button), which opens it. */
  default?: (props: { open: boolean }) => unknown
  /** The header's heading, in place of `title` and `description`; it names the dialog. */
  header?: () => unknown
  /** Controls in the header, before the close button. */
  actions?: () => unknown
  /** The dialog's content; `close()` closes it. */
  body?: (props: { close: () => void }) => unknown
  /** Below the body, for the dialog's buttons; `close()` closes it. */
  footer?: (props: { close: () => void }) => unknown
}>()

const m = useModal(props, emit, { prefix: 'dialog' })
const open = m.open
// Narrow is the viewport's width: the root element's.
const viewport = shallowRef<HTMLElement | null>(null)
onMounted(() => (viewport.value = document.documentElement))
const width = useWidth(viewport, { prepaint: false })
const full = computed(() => props.fullscreen || width.value === 'narrow')
</script>

<template>
  <span v-if="$slots.default" ref="wrap" class="weft-overlay-trigger" @click="m.onTrigger"><slot :open="open" /></span>
  <Teleport v-if="m.mounted.value && m.present.value" :to="overlayRoot()">
    <div
      class="weft-overlay"
      v-bind="m.context.attrs.value"
      :data-state="m.closing.value ? 'closed' : 'open'"
      :data-scrollable="scrollable ? 'true' : undefined"
    >
      <div v-if="overlay" class="weft-overlay-backdrop" @click="m.onBackdrop"></div>
      <div
        ref="panel"
        v-bind="$attrs"
        :id="m.ids.panel"
        class="weft-dialog"
        :role="role"
        aria-modal="true"
        :aria-labelledby="title || $slots.header ? m.ids.title : undefined"
        :aria-describedby="description && !$slots.header ? m.ids.description : undefined"
        :inert="m.closing.value || undefined"
        :data-fullscreen="full ? 'true' : undefined"
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
