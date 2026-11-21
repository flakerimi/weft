<script setup vapor lang="ts">
// A box whose content scrolls inside it, natively: the browser's own
// scrolling (momentum, wheel, touch, keyboard, find-in-page, scroll
// anchoring), with thin scrollbars in the theme's colours. While its content
// overflows its viewport is a tab stop, so a keyboard alone can scroll it
// (arrows, Page Up and Down, Home, End, Space are the browser's); with a
// `label` it is a named region. With `shadow` the edges with more content
// beyond them fade: overlays over the content, kept off the scrollbars
// (whose thickness is measured) and inside the focus ring, never a mask.
// On touch the browser draws its own overlay scrollbars.
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
    /** Fade the edges that have more content beyond them (block edges, inline edges, or all four for `both`). */
    shadow?: boolean
  }>(),
  { orientation: 'vertical' },
)

defineSlots<{
  /** What scrolls. */
  default?: () => unknown
}>()

const view = useTemplateRef<HTMLElement>('view')
const content = useTemplateRef<HTMLElement>('content')
/** Whether it overflows; true until measured, so the server's markup is a tab stop. */
const overflows = ref(true)
const fade = ref({ blockStart: false, blockEnd: false, inlineStart: false, inlineEnd: false })
/** The scrollbars' thickness, px, for the fades to stay off them. */
const bars = ref({ inline: 0, block: 0 })

function measure() {
  const el = view.value
  if (!el) return
  const v = props.orientation !== 'horizontal'
  const h = props.orientation !== 'vertical'
  const overY = v && el.scrollHeight - el.clientHeight > 1
  const overX = h && el.scrollWidth - el.clientWidth > 1
  overflows.value = overY || overX
  if (!props.shadow) return
  bars.value = { inline: Math.max(0, el.offsetWidth - el.clientWidth), block: Math.max(0, el.offsetHeight - el.clientHeight) }
  // Right to left, scrollLeft runs from 0 down to minus the overflow.
  const x = Math.abs(el.scrollLeft)
  fade.value = {
    blockStart: overY && el.scrollTop > 0,
    blockEnd: overY && el.scrollTop + el.clientHeight < el.scrollHeight - 1,
    inlineStart: overX && x > 0,
    inlineEnd: overX && x + el.clientWidth < el.scrollWidth - 1,
  }
}

let observer: ResizeObserver | undefined
onMounted(() => {
  measure()
  if (typeof ResizeObserver === 'undefined') return
  observer = new ResizeObserver(measure)
  if (view.value) observer.observe(view.value)
  if (content.value) observer.observe(content.value)
})
onScopeDispose(() => observer?.disconnect())

const on = (v: boolean) => (v ? 'true' : undefined)
const EDGES = ['block-start', 'block-end', 'inline-start', 'inline-end'] as const
const edges = () => (props.orientation === 'vertical' ? EDGES.slice(0, 2) : props.orientation === 'horizontal' ? EDGES.slice(2) : EDGES)
</script>

<template>
  <div
    class="weft-scroll-area"
    :data-orientation="orientation"
    :data-shadow="shadow ? 'true' : undefined"
    :data-fade-block-start="on(fade.blockStart)"
    :data-fade-block-end="on(fade.blockEnd)"
    :data-fade-inline-start="on(fade.inlineStart)"
    :data-fade-inline-end="on(fade.inlineEnd)"
    :style="{
      maxBlockSize: maxHeight,
      '--weft-scroll-bar-inline': shadow ? `${bars.inline}px` : undefined,
      '--weft-scroll-bar-block': shadow ? `${bars.block}px` : undefined,
    }"
  >
    <div
      ref="view"
      class="weft-scroll-area__viewport"
      :role="label ? 'region' : undefined"
      :aria-label="label"
      :tabindex="overflows ? 0 : undefined"
      @scroll.passive="measure"
    >
      <div ref="content" class="weft-scroll-area__content"><slot /></div>
    </div>
    <template v-if="shadow">
      <span v-for="edge in edges()" :key="edge" class="weft-scroll-area__fade" :data-edge="edge" aria-hidden="true"></span>
    </template>
  </div>
</template>
