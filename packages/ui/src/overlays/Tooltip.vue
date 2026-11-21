<script setup vapor lang="ts">
// A short text about its trigger (the default slot), shown on hover after a
// delay and at once on keyboard focus, never taking focus (WAI-ARIA APG,
// Tooltip). While shown, the trigger is described by it (aria-describedby,
// after any description it has). It hides on Escape (as the top overlay
// only), a press, blur, or the pointer leaving both the trigger and the
// tooltip, which it can be moved onto. A press outside it also goes on to
// the overlay below (a menu it shows over). A trigger with no name of its
// own (an icon alone) is named by the text, as aria-label, and then not
// described by it too. On touch, where nothing hovers, it renders nothing:
// that name is what remains.
import { computed, onScopeDispose, provide, useTemplateRef, watch } from 'vue'
import { useWeft } from '../adaptive/context.ts'
import { useControllable } from '../headless/useControllable.ts'
import { useDismiss } from '../headless/useDismiss.ts'
import { useFloating, type Placement } from '../headless/useFloating.ts'
import { useId } from '../headless/useId.ts'
import { isTouch, overlayRoot, useMounted, useOverlayContext, usePresence, useTrigger } from './overlay.ts'
import '../styles/base.css'
import '../styles/overlay.css'
import '../styles/tooltip.css'

defineOptions({ inheritAttrs: false })

const props = withDefaults(
  defineProps<{
    /** What it says. The content slot replaces it; give text anyway, as the name of a trigger that has none. */
    text?: string
    /** Where it shows against the trigger: `top`, `bottom-start`, … */
    placement?: Placement
    /** ms of hover before it shows; focus shows it at once, and so does a hover just after another tooltip closed. */
    delay?: number
    /** Never shown. */
    disabled?: boolean
    /** Shown or hidden, bound with `v-model:open`. */
    open?: boolean
    /** Whether it starts shown when `v-model:open` is not bound. */
    defaultOpen?: boolean
  }>(),
  { placement: 'top', delay: 500, open: undefined, defaultOpen: false },
)

const emit = defineEmits<{
  /** Shown or hidden. */
  'update:open': [open: boolean]
}>()

defineSlots<{
  /** The trigger: one element. */
  default?: () => unknown
  /** The tooltip's content, in place of `text`. */
  content?: () => unknown
}>()

// The trigger is named by the text when it has no name of its own, so an icon-only Button in it need not warn (its NAMES_TRIGGER).
provide('weft:names-trigger', true)

const open = useControllable(props, 'open', emit, () => props.defaultOpen)
const weft = useWeft()
const touch = computed(() => weft.input.value === 'touch')
const shown = computed(() => open.value && !touch.value && !props.disabled)
const mounted = useMounted()
const id = useId('tooltip')
const wrap = useTemplateRef<HTMLElement>('wrap')
const tip = useTemplateRef<HTMLElement>('tip')

/** Whether the trigger has a name not given by this tooltip. */
function named(el: HTMLElement): boolean {
  if (el.hasAttribute('aria-labelledby') || el.hasAttribute('title')) return true
  if (el.hasAttribute('aria-label') && !trigger.owns('aria-label')) return true
  return !!el.textContent?.trim()
}
const trigger = useTrigger(wrap, () => {
  const el = trigger.el.value
  return { 'aria-label': el && props.text && !named(el) ? props.text : undefined }
})
// Added to the trigger's own description, never replacing it, and not when
// the text names the trigger already (it would be read twice).
watch(
  [trigger.el, shown],
  ([el, on], [was]) => {
    if (was && was !== el) describe(was, false)
    if (el) describe(el, on && !trigger.owns('aria-label'))
  },
  { flush: 'post' },
)
function describe(el: HTMLElement, on: boolean) {
  const ids = (el.getAttribute('aria-describedby') ?? '').split(/\s+/).filter((t) => t && t !== id)
  if (on) ids.push(id)
  if (ids.length) el.setAttribute('aria-describedby', ids.join(' '))
  else el.removeAttribute('aria-describedby')
}

const context = useOverlayContext(() => trigger.el.value ?? wrap.value)
const presence = usePresence(shown, () => tip.value)
const live = computed(() => (presence.closing.value ? null : tip.value))
const floating = useFloating(trigger.el, tip, { placement: () => props.placement, offset: 6, flip: true, shift: true })

let timer: ReturnType<typeof setTimeout> | undefined
const clear = () => {
  if (timer !== undefined) clearTimeout(timer)
  timer = undefined
}
onScopeDispose(clear)
function show(wait: number) {
  clear()
  if (touch.value || props.disabled) return
  if (wait <= 0 || Date.now() - lastClosed < SKIP) open.value = true
  else timer = setTimeout(() => (open.value = true), wait)
}
function hide(wait = 0) {
  clear()
  if (wait > 0) timer = setTimeout(() => (open.value = false), wait)
  else open.value = false
}
// On touch it is not shown, and it does not come back when the input turns back to pointer.
watch(touch, (on) => on && hide())
watch(shown, (on, was) => {
  if (on) context.refresh()
  else if (was) lastClosed = Date.now()
}, { flush: 'sync' })

/** Pressed: focus that follows a press does not show it. */
let pressing = false
function onPointerdown() {
  pressing = true
  hide()
}
function onFocusin() {
  if (!pressing) show(0)
  pressing = false
}
function onEnter(event: PointerEvent) {
  if (!isTouch(event)) show(props.delay)
}
function onLeave(event: PointerEvent) {
  if (!isTouch(event)) hide(GRACE)
}

useDismiss(live, { onDismiss: () => hide(), ignore: () => [wrap.value], returnFocus: false, passThrough: true })
</script>

<script lang="ts">
/** ms the pointer has to go from the trigger onto the tooltip. */
const GRACE = 120
/** ms after a tooltip hides in which the next one shows without a delay. */
const SKIP = 300
let lastClosed = 0
</script>

<template>
  <span
    ref="wrap"
    class="weft-overlay-trigger"
    @pointerenter="onEnter"
    @pointerleave="onLeave"
    @pointerdown="onPointerdown"
    @focusin="onFocusin"
    @focusout="hide()"
  ><slot /></span>
  <Teleport v-if="mounted && presence.present.value" :to="overlayRoot()">
    <div class="weft-overlay" v-bind="context.attrs.value" :data-state="presence.closing.value ? 'closed' : 'open'">
      <div
        ref="tip"
        v-bind="$attrs"
        :id="id"
        class="weft-tooltip"
        role="tooltip"
        :data-side="floating.placement.value.split('-')[0]"
        :data-positioned="floating.isPositioned.value ? 'true' : undefined"
        :style="floating.styles.value"
        @pointerenter="clear"
        @pointerleave="hide(GRACE)"
      ><slot name="content">{{ text }}</slot></div>
    </div>
  </Teleport>
</template>
