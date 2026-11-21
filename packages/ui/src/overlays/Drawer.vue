<script setup vapor lang="ts">
// A Sheet with a handle that can be dragged away: from the bottom by
// default, a modal dialog (WAI-ARIA APG, Dialog (Modal)) that a finger can
// also close by dragging its handle or header towards its edge, past a
// quarter of its size or with a flick. Short of that it springs back.
// Escape, the backdrop and the close button close it too, and focus
// returns to the trigger.
import { onScopeDispose } from 'vue'
import { isRtl } from '../headless/dom.ts'
import { overlayRoot } from './overlay.ts'
import { useModal } from './modal.ts'
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
    /** The heading, which names the drawer. Without one, give the drawer an `aria-label`. */
    title?: string
    /** A sentence under the title, which describes the drawer. */
    description?: string
    /** Show the close button (×) in the header; off by default, as the handle and the backdrop close it. */
    close?: boolean
    /** The close button's name for assistive technology. */
    closeLabel?: string
    /** Escape and a press on the backdrop close it. */
    dismissible?: boolean
    /** The side it comes from and is dragged back to: `bottom`, `top`, `start` or `end` (mirrored right to left). */
    side?: 'start' | 'end' | 'top' | 'bottom'
    /** Show the drag handle. */
    handle?: boolean
  }>(),
  { open: undefined, defaultOpen: false, close: false, closeLabel: 'Close', dismissible: true, side: 'bottom', handle: true },
)

const emit = defineEmits<{
  /** Opened or closed, by the trigger, Escape, the backdrop or a close. */
  'update:open': [open: boolean]
}>()

defineSlots<{
  /** The trigger: one element (a Button), which opens it. */
  default?: (props: { open: boolean }) => unknown
  /** The header's heading, in place of `title` and `description`. */
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

/** A drag in progress: where it started and how far it has come towards the edge, in px. */
let drag: { id: number; start: number; at: number; time: number; size: number } | null = null
/** How far a pointer has moved towards the drawer's edge since `from`. */
function towardsEdge(event: PointerEvent, from: number): number {
  const panel = m.panel.value!
  const vertical = props.side === 'top' || props.side === 'bottom'
  const delta = (vertical ? event.clientY : event.clientX) - from
  const rtl = isRtl(panel)
  const sign = { bottom: 1, top: -1, end: rtl ? -1 : 1, start: rtl ? 1 : -1 }[props.side]
  return delta * sign
}
const coordinate = (event: PointerEvent) => (props.side === 'top' || props.side === 'bottom' ? event.clientY : event.clientX)
function onDragStart(event: PointerEvent) {
  const panel = m.panel.value
  if (!panel || !props.dismissible || (event.pointerType === 'mouse' && event.button !== 0)) return
  // A press on a control in the header is that control's.
  const control = (event.target as Element).closest('button, a, input, select, textarea, [tabindex]')
  if (control && (event.currentTarget as Element).contains(control)) return
  const vertical = props.side === 'top' || props.side === 'bottom'
  drag = { id: event.pointerId, start: coordinate(event), at: 0, time: Date.now(), size: vertical ? panel.offsetHeight : panel.offsetWidth }
  try {
    ;(event.currentTarget as Element).setPointerCapture(event.pointerId)
  } catch {
    // No such pointer any more: the moves still reach the handle while over it.
  }
  panel.dataset.dragging = 'true'
}
function onDragMove(event: PointerEvent) {
  if (!drag || event.pointerId !== drag.id) return
  drag.at = Math.max(0, towardsEdge(event, drag.start))
  m.panel.value?.style.setProperty('--weft-drawer-drag', `${drag.at}px`)
}
function onDragEnd(event: PointerEvent) {
  if (!drag || event.pointerId !== drag.id) return
  const { at, size, time } = drag
  const speed = at / Math.max(1, Date.now() - time)
  drag = null
  const panel = m.panel.value
  panel?.style.removeProperty('--weft-drawer-drag')
  if (panel) delete panel.dataset.dragging
  if (at > size / 4 || (at > 8 && speed > FLICK)) m.close()
}
onScopeDispose(() => (drag = null))
</script>

<script lang="ts">
/** px per ms: a drag at least this fast closes the drawer however short it is. */
const FLICK = 0.5
</script>

<template>
  <span v-if="$slots.default" ref="wrap" class="weft-overlay-trigger" @click="m.onTrigger"><slot :open="open" /></span>
  <Teleport v-if="m.mounted.value && open" :to="overlayRoot()">
    <div
      class="weft-overlay"
      :dir="m.context.attrs.value.dir"
      :data-weft-input="m.context.attrs.value['data-weft-input']"
      :data-weft-surface="m.context.attrs.value['data-weft-surface']"
      :data-weft-scheme="m.context.attrs.value['data-weft-scheme']"
    >
      <div class="weft-overlay-backdrop" @click="m.onBackdrop"></div>
      <div
        ref="panel"
        v-bind="$attrs"
        :id="m.ids.panel"
        class="weft-sheet weft-drawer"
        role="dialog"
        aria-modal="true"
        :aria-labelledby="title ? m.ids.title : undefined"
        :aria-describedby="description ? m.ids.description : undefined"
        :data-side="side"
      >
        <div
          v-if="handle"
          class="weft-drawer__handle"
          aria-hidden="true"
          @pointerdown="onDragStart"
          @pointermove="onDragMove"
          @pointerup="onDragEnd"
          @pointercancel="onDragEnd"
        ></div>
        <div
          v-if="title || description || close || $slots.header || $slots.actions"
          class="weft-overlay__header"
          @pointerdown="onDragStart"
          @pointermove="onDragMove"
          @pointerup="onDragEnd"
          @pointercancel="onDragEnd"
        >
          <div class="weft-overlay__heading">
            <slot name="header">
              <h2 v-if="title" :id="m.ids.title" class="weft-overlay__title">{{ title }}</h2>
              <p v-if="description" :id="m.ids.description" class="weft-overlay__description">{{ description }}</p>
            </slot>
          </div>
          <div v-if="$slots.actions" class="weft-overlay__actions"><slot name="actions" /></div>
          <button v-if="close" type="button" class="weft-overlay__close" :aria-label="closeLabel" @click="m.close">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M18 6 6 18" /><path d="m6 6 12 12" /></svg>
          </button>
        </div>
        <div v-if="$slots.body" class="weft-overlay__body"><slot name="body" :close="m.close" /></div>
        <div v-if="$slots.footer" class="weft-overlay__footer"><slot name="footer" :close="m.close" /></div>
      </div>
    </div>
  </Teleport>
</template>
