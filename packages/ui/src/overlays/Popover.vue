<script setup vapor lang="ts">
// Floating content opened from a trigger (the default slot), or against an
// anchor. On pointer it floats beside the trigger and is not modal: focus
// moves in, Tab can leave, and focus leaving closes it. On touch the same
// content element becomes a bottom sheet over a backdrop, modal: focus is
// trapped and the page does not scroll. The input can change while it is
// open; only the element's form changes, so what is inside keeps its state
// and focus. Escape and a press outside close it (unless not dismissible);
// focus returns to the trigger, except after a press outside. In `hover`
// mode it opens while the pointer is over the trigger or the content (on
// touch it is pressed, as ever). Modal, the page behind is inert. It fades
// out when it closes, unless motion is reduced.
import { computed, onScopeDispose, useTemplateRef, watch } from 'vue'
import { useWeft } from '../adaptive/context.ts'
import { focus, tabbables } from '../headless/dom.ts'
import { overlayStack } from '../headless/overlayStack.ts'
import { useControllable } from '../headless/useControllable.ts'
import { useDismiss } from '../headless/useDismiss.ts'
import { arrow as arrowMiddleware, useFloating, type Placement } from '../headless/useFloating.ts'
import { useFocusTrap } from '../headless/useFocusTrap.ts'
import { useId } from '../headless/useId.ts'
import { useInert } from '../headless/useInert.ts'
import { useScrollLock } from '../headless/useScrollLock.ts'
import { elementOf, isTouch, overlayRoot, useMounted, useOverlayContext, usePresence, useTrigger, type Anchor } from './overlay.ts'
import '../styles/base.css'
import '../styles/overlay.css'
import '../styles/sheet.css'
import '../styles/popover.css'

defineOptions({ inheritAttrs: false })

const props = withDefaults(
  defineProps<{
    /** Open or closed, bound with `v-model:open`. */
    open?: boolean
    /** Whether it starts open when `v-model:open` is not bound. */
    defaultOpen?: boolean
    /** Where it floats against the trigger on pointer: `bottom`, `top-start`, … (start and end follow the text direction). */
    placement?: Placement
    /** The gap to the trigger, in px. */
    offset?: number
    /** Names the content for assistive technology; on touch it is the sheet's heading too. */
    title?: string
    /** Escape and a press outside close it. */
    dismissible?: boolean
    /** Modal on pointer too: focus is trapped and the page does not scroll. On touch it is always modal. */
    modal?: boolean
    /** Move focus into the content when it opens (on pointer); false keeps focus where it is, as a combobox's input needs. */
    autoFocus?: boolean
    /** Float against this element or point instead of the trigger. */
    anchor?: Anchor
    /** `click`: a press on the trigger toggles it. `hover`: the pointer over the trigger or the content opens it too, without moving focus. */
    mode?: 'click' | 'hover'
    /** In hover mode, ms of hover before it opens. */
    openDelay?: number
    /** In hover mode, ms after the pointer leaves before it closes. */
    closeDelay?: number
    /** Draw an arrow pointing at the trigger (floating form only). */
    arrow?: boolean
  }>(),
  {
    open: undefined,
    defaultOpen: false,
    placement: 'bottom',
    offset: 8,
    dismissible: true,
    modal: false,
    autoFocus: true,
    mode: 'click',
    openDelay: 100,
    closeDelay: 200,
  },
)

const emit = defineEmits<{
  /** Opened or closed, by the trigger, Escape, a press outside or focus leaving. */
  'update:open': [open: boolean]
}>()

defineSlots<{
  /** The trigger: one element (a Button), which a press toggles. */
  default?: (props: { open: boolean }) => unknown
  /** What it shows; `close()` closes it. */
  content?: (props: { close: () => void }) => unknown
}>()

const open = useControllable(props, 'open', emit, () => props.defaultOpen)
const weft = useWeft()
const sheet = computed(() => weft.input.value === 'touch')
const trapping = computed(() => open.value && (sheet.value || props.modal))
const mounted = useMounted()
const id = useId('popover')
const wrap = useTemplateRef<HTMLElement>('wrap')
const panel = useTemplateRef<HTMLElement>('panel')
const backdrop = useTemplateRef<HTMLElement>('backdrop')
const arrowEl = useTemplateRef<HTMLElement>('arrowEl')
const presence = usePresence(open, () => panel.value)
/** The content while it is open, not while it leaves. */
const live = computed(() => (presence.closing.value ? null : panel.value))

const trigger = useTrigger(wrap, () => ({
  'aria-haspopup': 'dialog',
  'aria-expanded': open.value ? 'true' : 'false',
  'aria-controls': open.value ? id : undefined,
}))
const context = useOverlayContext(() => trigger.el.value ?? wrap.value)
const reference = computed(() => props.anchor ?? trigger.el.value)
const floating = useFloating(reference, computed(() => (sheet.value ? null : panel.value)), {
  placement: () => props.placement,
  offset: () => props.offset,
  flip: true,
  shift: true,
  size: true,
  middleware: () => (props.arrow ? [arrowMiddleware(() => ({ element: arrowEl.value as Element, padding: 8 }))] : []),
})
const arrowStyle = computed(() => {
  const at = floating.middlewareData.value.arrow
  return { left: at?.x != null ? `${at.x}px` : '', top: at?.y != null ? `${at.y}px` : '' }
})

const close = () => (open.value = false)
watch(open, (on) => {
  if (!on) hovered = pinned = false
})
function toggle(event: MouseEvent) {
  if (event.defaultPrevented) return
  // A click on a trigger the hover has just opened keeps it open, pinned: the next one closes it.
  if (open.value && hovered) {
    hovered = false
    pinned = true
    cancelHover()
    return
  }
  open.value = !open.value
}

useDismiss(live, {
  onDismiss: close,
  escape: () => props.dismissible,
  outside: () => props.dismissible,
  // A press on the element it is anchored to is not outside it, as for the trigger.
  ignore: () => [wrap.value, backdrop.value, elementOf(props.anchor)],
  returnTo: () => trigger.el.value,
})
// A trap that ends because the popover became a floating one (touch to
// pointer) leaves focus where it is; one that ends with the popover returns it.
useFocusTrap(live, { active: trapping, returnFocus: () => !open.value })
useScrollLock(trapping)
useInert(trapping)

watch(open, (on) => on && context.refresh(), { immediate: true, flush: 'sync' })
// Not trapped: move focus in once, when the content appears; not when it
// was open from the start (a page that hydrates keeps its focus), and not
// when the pointer opened it in hover mode.
let quiet = open.value
watch(open, (on) => !on && (quiet = false))
watch(
  live,
  (el) => {
    if (!el) return
    const skip = quiet || hovered
    quiet = false
    if (!skip && !trapping.value && props.autoFocus) focus(tabbables(el)[0] ?? el)
  },
  { flush: 'post' },
)

// Hover mode: open while the pointer is over the trigger or the content.
let hovered = false
/** Clicked open in hover mode: the pointer leaving no longer closes it. */
let pinned = false
let hoverTimer: ReturnType<typeof setTimeout> | undefined
const cancelHover = () => {
  if (hoverTimer !== undefined) clearTimeout(hoverTimer)
  hoverTimer = undefined
}
onScopeDispose(cancelHover)
const later = (ms: number, fn: () => void) => {
  cancelHover()
  if (ms > 0) hoverTimer = setTimeout(fn, ms)
  else fn()
}
function onEnter(event: PointerEvent) {
  if (props.mode !== 'hover' || isTouch(event) || sheet.value) return
  if (open.value) return cancelHover()
  later(props.openDelay, () => {
    hovered = true
    open.value = true
  })
}
function onLeave(event: PointerEvent) {
  if (props.mode !== 'hover' || isTouch(event) || sheet.value || pinned) return
  later(props.closeDelay, () => {
    hovered = false
    close()
  })
}

/** On pointer, focus leaving for somewhere other than the trigger or an overlay above closes it. */
function onFocusout(event: FocusEvent) {
  const to = event.relatedTarget as Node | null
  const el = panel.value
  if (trapping.value || !to || !el || el.contains(to) || wrap.value?.contains(to) || overlayStack.isAbove(to, el)) return
  close()
}

function onBackdrop() {
  if (props.dismissible) close()
}

const side = computed(() => (sheet.value ? 'bottom' : floating.placement.value.split('-')[0]))
</script>

<template>
  <span
    v-if="$slots.default"
    ref="wrap"
    class="weft-overlay-trigger"
    @click="toggle"
    @pointerenter="onEnter"
    @pointerleave="onLeave"
  ><slot :open="open" /></span>
  <Teleport v-if="mounted && presence.present.value" :to="overlayRoot()">
    <div class="weft-overlay" v-bind="context.attrs.value" :data-state="presence.closing.value ? 'closed' : 'open'">
      <div v-if="sheet" ref="backdrop" class="weft-overlay-backdrop" @click="onBackdrop"></div>
      <div
        ref="panel"
        role="dialog"
        tabindex="-1"
        :aria-label="title"
        v-bind="$attrs"
        :id="id"
        class="weft-popover"
        :class="{ 'weft-sheet': sheet }"
        :aria-modal="trapping ? 'true' : undefined"
        :data-form="sheet ? 'sheet' : 'floating'"
        :data-side="side"
        :data-positioned="sheet || floating.isPositioned.value ? 'true' : undefined"
        :style="sheet ? undefined : floating.styles.value"
        @focusout="onFocusout"
        @pointerenter="onEnter"
        @pointerleave="onLeave"
      >
        <div v-if="arrow && !sheet" ref="arrowEl" class="weft-popover__arrow" aria-hidden="true" :style="arrowStyle"></div>
        <h2 v-if="sheet && title" class="weft-popover__title">{{ title }}</h2>
        <slot name="content" :close="close" />
      </div>
    </div>
  </Teleport>
</template>
