<script setup vapor lang="ts">
// A Sheet with a handle that can be dragged away: from the bottom by
// default, a modal dialog (WAI-ARIA APG, Dialog (Modal)) that a finger can
// also close by dragging its handle or header towards its edge, past a
// quarter of its size or with a flick (fast over its last moves). Short of
// that it springs back. Escape, the backdrop and the close button close it
// too, and focus returns to the trigger.
import { isRtl } from '../headless/dom.ts'
import { createSwipe } from '../headless/swipe.ts'
import { overlayRoot } from './overlay.ts'
import { useModal } from './modal.ts'
import OverlayPanel from './OverlayPanel.vue'
import '../styles/base.css'
import '../styles/overlay.css'
import '../styles/sheet.css'
import '../styles/drawer.css'

defineOptions({ inheritAttrs: false })

const props = withDefaults(
  defineProps<{
    /** Open or closed, bound with `v-model:open`. */
    open?: boolean
    /** Whether it starts open when `v-model:open` is not bound. */
    defaultOpen?: boolean
    /** The heading, which names the drawer. Without one or a header slot, give the drawer an `aria-label`. */
    title?: string
    /** A sentence under the title, which describes the drawer. */
    description?: string
    /** Show the close button (×) in the header; off by default, as the handle and the backdrop close it. */
    close?: boolean
    /** The close button's name for assistive technology. */
    closeLabel?: string
    /** Escape and a press on the backdrop close it. */
    dismissible?: boolean
    /** Dim the page with a backdrop; without one it is still modal. */
    overlay?: boolean
    /** The side it comes from and is dragged back to: `bottom`, `top`, `start` or `end` (mirrored right to left). */
    side?: 'start' | 'end' | 'top' | 'bottom'
    /** Show the drag handle. */
    handle?: boolean
  }>(),
  { open: undefined, defaultOpen: false, close: false, closeLabel: 'Close', dismissible: true, overlay: true, side: 'bottom', handle: true },
)

const emit = defineEmits<{
  /** Opened or closed, by the trigger, Escape, the backdrop or a close. */
  'update:open': [open: boolean]
}>()

defineSlots<{
  /** The trigger: one element (a Button), which opens it. */
  default?: (props: { open: boolean }) => unknown
  /** The header's heading, in place of `title` and `description`; it names the drawer. */
  header?: () => unknown
  /** Controls in the header, before the close button. */
  actions?: () => unknown
  /** The drawer's content, which scrolls when long; `close()` closes it. */
  body?: (props: { close: () => void }) => unknown
  /** Below the body, for the drawer's buttons; `close()` closes it. */
  footer?: (props: { close: () => void }) => unknown
}>()

const m = useModal(props, emit, { prefix: 'drawer' })
const open = m.open

const vertical = () => props.side === 'top' || props.side === 'bottom'
const swipe = createSwipe(() => {
  const rtl = isRtl(m.panel.value)
  const sign = { bottom: 1, top: -1, end: rtl ? -1 : 1, start: rtl ? 1 : -1 }[props.side] as 1 | -1
  return { axis: vertical() ? 'y' : 'x', sign }
})
let size = 0
function onDragStart(event: PointerEvent) {
  const panel = m.panel.value
  if (!panel || !props.dismissible || (event.pointerType === 'mouse' && event.button !== 0)) return
  // Dragged by its handle or header; a press on a control in the header is that control's.
  const target = event.target as Element
  const grip = target.closest('.weft-drawer__handle, .weft-overlay__header')
  if (!grip || !panel.contains(grip)) return
  const control = target.closest('button, a, input, select, textarea, [tabindex]')
  if (control && grip.contains(control)) return
  size = vertical() ? panel.offsetHeight : panel.offsetWidth
  swipe.start(event)
  try {
    ;(event.currentTarget as Element).setPointerCapture(event.pointerId)
  } catch {
    // No such pointer any more: the moves still reach the handle while over it.
  }
  panel.dataset.dragging = 'true'
}
function onDragMove(event: PointerEvent) {
  const at = swipe.move(event)
  if (at !== null) m.panel.value?.style.setProperty('--weft-drawer-drag', `${at}px`)
}
function onDragEnd(event: PointerEvent) {
  const end = swipe.end(event)
  if (!end) return
  const panel = m.panel.value
  panel?.style.removeProperty('--weft-drawer-drag')
  if (panel) delete panel.dataset.dragging
  if (end.distance > size / 4 || (end.distance > 8 && end.speed > FLICK)) m.close()
}
</script>

<script lang="ts">
/** px per ms over the last moves: a drag at least this fast at its end closes the drawer however short it is. */
const FLICK = 0.5
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
        class="weft-sheet weft-drawer"
        role="dialog"
        aria-modal="true"
        :aria-labelledby="title || $slots.header ? m.ids.title : undefined"
        :aria-describedby="description && !$slots.header ? m.ids.description : undefined"
        :inert="m.closing.value || undefined"
        :data-side="side"
        @pointerdown="onDragStart"
        @pointermove="onDragMove"
        @pointerup="onDragEnd"
        @pointercancel="onDragEnd"
      >
        <div v-if="handle" class="weft-drawer__handle" aria-hidden="true"></div>
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
