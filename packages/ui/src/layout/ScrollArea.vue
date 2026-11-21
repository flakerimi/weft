<script setup vapor lang="ts">
// A box whose content scrolls inside it, natively: the browser's own
// scrolling (momentum, wheel, touch, keyboard, find-in-page, scroll
// anchoring), with thin scrollbars in the theme's colours. While its content
// overflows it is a tab stop, so a keyboard alone can scroll it (arrows,
// Page Up and Down, Home, End, Space are the browser's); with a `label` it
// is a named region. With `shadow` the edges with more content beyond them
// fade. On touch the browser draws its own overlay scrollbars.
import { onMounted, onScopeDispose, ref, useTemplateRef } from 'vue'
import '../styles/base.css'
import '../styles/scroll-area.css'

const props = withDefaults(
  defineProps<{
    /** Which way it scrolls: `vertical` (default), `horizontal` or `both`. */
    orientation?: 'vertical' | 'horizontal' | 'both'
    /** Its name, for assistive technology: it becomes a region. Give one when it is a tab stop. */
    label?: string
    /** The most it may be tall before it scrolls, any CSS length; or set its height from outside. */
    maxHeight?: string
    /** Fade the edges that have more content beyond them. */
    shadow?: boolean
  }>(),
  { orientation: 'vertical' },
)

defineSlots<{
  /** What scrolls. */
  default?: () => unknown
}>()

const root = useTemplateRef<HTMLElement>('root')
const content = useTemplateRef<HTMLElement>('content')
/** Whether it overflows; true until measured, so the server's markup is a tab stop. */
const overflows = ref(true)
const fadeStart = ref(false)
const fadeEnd = ref(false)

function measure() {
  const el = root.value
  if (!el) return
  const v = props.orientation !== 'horizontal'
  const h = props.orientation !== 'vertical'
  const overY = el.scrollHeight - el.clientHeight > 1
  const overX = el.scrollWidth - el.clientWidth > 1
  overflows.value = (v && overY) || (h && overX)
  if (!props.shadow) return
  // Right to left, scrollLeft runs from 0 down to minus the overflow.
  const x = Math.abs(el.scrollLeft)
  const atStart = (!v || el.scrollTop <= 0) && (!h || x <= 0)
  const atEnd = (!v || el.scrollTop + el.clientHeight >= el.scrollHeight - 1) && (!h || x + el.clientWidth >= el.scrollWidth - 1)
  fadeStart.value = overflows.value && !atStart
  fadeEnd.value = overflows.value && !atEnd
}

let observer: ResizeObserver | undefined
onMounted(() => {
  measure()
  if (typeof ResizeObserver === 'undefined') return
  observer = new ResizeObserver(measure)
  if (root.value) observer.observe(root.value)
  if (content.value) observer.observe(content.value)
})
onScopeDispose(() => observer?.disconnect())
</script>

<template>
  <div
    ref="root"
    class="weft-scroll-area"
    :data-orientation="orientation"
    :data-shadow="shadow ? 'true' : undefined"
    :data-fade-start="fadeStart ? 'true' : undefined"
    :data-fade-end="fadeEnd ? 'true' : undefined"
    :role="label ? 'region' : undefined"
    :aria-label="label"
    :tabindex="overflows ? 0 : undefined"
    :style="maxHeight ? { maxBlockSize: maxHeight } : undefined"
    @scroll.passive="measure"
  >
    <div ref="content" class="weft-scroll-area__content"><slot /></div>
  </div>
</template>
