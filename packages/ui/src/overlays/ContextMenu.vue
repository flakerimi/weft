<script setup vapor lang="ts">
// A menu for an area (the default slot), opened in place of the browser's
// own: by a right press at the pointer, by the context menu key or
// Shift+F10 on something focused inside (under it), and on touch by a long
// press, as a bottom sheet. The menu is a DropdownMenu, with all of its
// keyboard and its items; it closes back to where focus was. Attributes go
// to the menu: give it an `aria-label`.
import { onScopeDispose, ref, shallowRef, useTemplateRef } from 'vue'
import type { MenuItem } from './menu.ts'
import type { Anchor } from './overlay.ts'
import DropdownMenu from './DropdownMenu.vue'
import '../styles/base.css'
import '../styles/context-menu.css'

defineOptions({ inheritAttrs: false })

const props = withDefaults(
  defineProps<{
    /** The menu's items, or groups of them. */
    items?: MenuItem[] | MenuItem[][]
    /** Leave the browser's own menu in place. */
    disabled?: boolean
  }>(),
  { items: () => [] },
)

const emit = defineEmits<{
  /** Opened or closed. */
  'update:open': [open: boolean]
}>()

defineSlots<{
  /** The area the menu belongs to. */
  default?: () => unknown
  /** An item's label, in place of its `label`. */
  item?: (props: { item: MenuItem }) => unknown
}>()

const open = ref(false)
const area = useTemplateRef<HTMLElement>('area')
const point = shallowRef<Anchor | undefined>()

function setOpen(on: boolean) {
  open.value = on
  emit('update:open', on)
}
function openAt(x: number, y: number) {
  const rect = new DOMRect(x, y, 0, 0)
  point.value = { getBoundingClientRect: () => rect, contextElement: area.value ?? undefined }
  if (!open.value) setOpen(true)
}

function onContextmenu(event: MouseEvent) {
  if (props.disabled) return
  event.preventDefault()
  if (longPressed) return // the browser's own long-press menu, after ours opened
  if (event.clientX === 0 && event.clientY === 0) {
    // From the keyboard: under the focused element.
    const r = (event.target as Element).getBoundingClientRect()
    openAt(r.left, r.bottom)
  } else openAt(event.clientX, event.clientY)
}

// A long press on touch: held still for LONG_PRESS ms.
let timer: ReturnType<typeof setTimeout> | undefined
let from: { x: number; y: number } | null = null
let longPressed = false
const cancel = () => {
  if (timer !== undefined) clearTimeout(timer)
  timer = undefined
  from = null
}
onScopeDispose(cancel)
function onPointerdown(event: PointerEvent) {
  longPressed = false
  if (props.disabled || event.pointerType !== 'touch') return
  cancel()
  from = { x: event.clientX, y: event.clientY }
  const at = from
  timer = setTimeout(() => {
    timer = undefined
    longPressed = true
    openAt(at.x, at.y)
  }, LONG_PRESS)
}
function onPointermove(event: PointerEvent) {
  if (from && Math.hypot(event.clientX - from.x, event.clientY - from.y) > SLOP) cancel()
}
function onPointerup() {
  cancel()
  // The finger lifts over the sheet's backdrop: the click that follows is the press's, not a dismissal.
  if (longPressed) swallowClick()
}
function swallowClick() {
  const swallow = (event: MouseEvent) => {
    event.stopPropagation()
    event.preventDefault()
  }
  document.addEventListener('click', swallow, { capture: true, once: true })
  setTimeout(() => document.removeEventListener('click', swallow, { capture: true }), CLICK_WAIT)
}
</script>

<script lang="ts">
/** ms a finger is held still before the menu opens. */
const LONG_PRESS = 600
/** px a finger may move and still be a long press. */
const SLOP = 10
/** ms to wait for the click that ends a long press. */
const CLICK_WAIT = 600
</script>

<template>
  <div
    ref="area"
    class="weft-context-menu"
    @contextmenu="onContextmenu"
    @pointerdown="onPointerdown"
    @pointermove="onPointermove"
    @pointerup="onPointerup"
    @pointercancel="cancel"
  ><slot /></div>
  <DropdownMenu v-bind="$attrs" :open="open" :items="items" :anchor="point" placement="bottom-start" @update:open="setOpen">
    <template v-if="$slots.item" #item="{ item }"><slot name="item" :item="item" /></template>
  </DropdownMenu>
</template>
