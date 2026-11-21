<script setup vapor lang="ts">
// Floating content opened from a trigger (the default slot), or against an
// anchor. On pointer it floats beside the trigger and is not modal: focus
// moves in, Tab can leave, and focus leaving closes it. On touch the same
// content element becomes a bottom sheet over a backdrop, modal: focus is
// trapped and the page does not scroll. The input can change while it is
// open; only the element's form changes, so what is inside keeps its state
// and focus. Escape and a press outside close it (unless not dismissible);
// focus returns to the trigger, except after a press outside.
import { computed, useTemplateRef, watch } from 'vue'
import { useWeft } from '../adaptive/context.ts'
import { focus, tabbables } from '../headless/dom.ts'
import { overlayStack } from '../headless/overlayStack.ts'
import { useControllable } from '../headless/useControllable.ts'
import { useDismiss } from '../headless/useDismiss.ts'
import { useFloating, type Placement } from '../headless/useFloating.ts'
import { useFocusTrap } from '../headless/useFocusTrap.ts'
import { useId } from '../headless/useId.ts'
import { useScrollLock } from '../headless/useScrollLock.ts'
import { overlayRoot, useMounted, useOverlayContext, useTrigger, type Anchor } from './overlay.ts'
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
  }>(),
  { open: undefined, defaultOpen: false, placement: 'bottom', offset: 8, dismissible: true, modal: false, autoFocus: true },
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

const trigger = useTrigger(wrap, () => ({
  'aria-haspopup': 'dialog',
  'aria-expanded': open.value ? 'true' : 'false',
  'aria-controls': open.value ? id : undefined,
}))
const context = useOverlayContext(() => trigger.el.value ?? wrap.value)
const reference = computed(() => props.anchor ?? trigger.el.value)
const floating = useFloating(reference, computed(() => (sheet.value ? null : panel.value)), {
  placement: () => props.placement,
  offset: props.offset,
  flip: true,
  shift: true,
  size: true,
})

const close = () => (open.value = false)
function toggle(event: MouseEvent) {
  if (!event.defaultPrevented) open.value = !open.value
}

useDismiss(panel, {
  onDismiss: close,
  escape: () => props.dismissible,
  outside: () => props.dismissible,
  ignore: () => [wrap.value, backdrop.value],
  returnTo: () => trigger.el.value,
})
// A trap that ends because the popover became a floating one (touch to
// pointer) leaves focus where it is; one that ends with the popover returns it.
useFocusTrap(panel, { active: trapping, returnFocus: () => !open.value })
useScrollLock(trapping)

watch(open, (on) => on && context.refresh(), { immediate: true, flush: 'sync' })
// Not trapped: move focus in once, when the content appears.
watch(
  panel,
  (el) => {
    if (el && !trapping.value && props.autoFocus) focus(tabbables(el)[0] ?? el)
  },
  { flush: 'post' },
)

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
  <span v-if="$slots.default" ref="wrap" class="weft-overlay-trigger" @click="toggle"><slot :open="open" /></span>
  <Teleport v-if="mounted && open" :to="overlayRoot()">
    <div
      class="weft-overlay"
      :dir="context.attrs.value.dir"
      :data-weft-input="context.attrs.value['data-weft-input']"
      :data-weft-surface="context.attrs.value['data-weft-surface']"
      :data-weft-scheme="context.attrs.value['data-weft-scheme']"
    >
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
      >
        <h2 v-if="sheet && title" class="weft-overlay__title weft-popover__title">{{ title }}</h2>
        <slot name="content" :close="close" />
      </div>
    </div>
  </Teleport>
</template>
