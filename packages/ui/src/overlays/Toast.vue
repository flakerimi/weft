<script setup vapor lang="ts">
// One toast: a short message with an optional icon, actions and close
// button. It leaves on its own after `duration` ms (by default the theme's
// --weft-duration-toast), a bar showing the time left, by emitting `close`
// once its leave transition has played; its time stops while `paused` (a
// Toaster pauses every toast while it is hovered or has focus), resuming
// with what it had left, and starts again when `duration` changes. Escape
// on it closes it; on touch a swipe towards `swipe` (its Toaster's edge)
// throws it away. Toaster renders these from useToast(); render one
// yourself only inside a live region.
import { computed, onMounted, onScopeDispose, ref, useTemplateRef, watch } from 'vue'
import Button from '../basics/Button.vue'
import Icon from '../basics/Icon.vue'
import { isRtl } from '../headless/dom.ts'
import { createSwipe } from '../headless/swipe.ts'
import type { Tone } from '../types.ts'
import CloseButton from './CloseButton.vue'
import { leaveTime } from './overlay.ts'
import { cssTime, type ToastAction } from './toast.ts'
import '../styles/base.css'
import '../styles/tone.css'
import '../styles/close-button.css'
import '../styles/toast.css'

const props = withDefaults(
  defineProps<{
    /** The message, in bold. */
    title?: string
    /** More about it, under the title. */
    description?: string
    /** The colour and what it says: `success`, `danger`, … */
    tone?: Tone
    /** A Lucide icon before the text. */
    icon?: string
    /** ms before it leaves on its own; default `--weft-duration-toast`; 0 stays until closed. Changing it starts the time again. */
    duration?: number
    /** Its time stands still: hovered, or focus inside. */
    paused?: boolean
    /** Show the close button. */
    close?: boolean
    /** The close button's name for assistive technology. */
    closeLabel?: string
    /** Buttons after the text; each closes the toast after it runs. */
    actions?: ToastAction[]
    /** Show the time left as a bar. */
    progress?: boolean
    /** The edge a swipe on touch throws it towards: its Toaster's side. */
    swipe?: 'start' | 'end'
  }>(),
  { tone: 'neutral', close: true, closeLabel: 'Close', progress: true, swipe: 'end' },
)

const emit = defineEmits<{
  /** It has gone: its time ran out, or it was closed, acted on or swiped away. */
  close: []
}>()

defineSlots<{
  /** The text, in place of `title` and `description`. */
  default?: () => unknown
  /** Before the text, in place of `icon`. */
  leading?: () => unknown
  /** After the text, in place of `actions`. */
  actions?: () => unknown
}>()

const root = useTemplateRef<HTMLElement>('root')
/** ms of the whole run, Infinity for none; and of what is left. */
const total = ref(Infinity)
const run = ref(0)
let left = Infinity
let started = 0
let timer: ReturnType<typeof setTimeout> | undefined
const leaving = ref(false)

function stop() {
  if (timer === undefined) return
  clearTimeout(timer)
  timer = undefined
  left -= Date.now() - started
}
function go() {
  if (timer !== undefined || props.paused || leaving.value || !Number.isFinite(left)) return
  started = Date.now()
  timer = setTimeout(() => {
    timer = undefined
    leave()
  }, Math.max(0, left))
}
/** The theme's --weft-duration-toast as it reaches this toast (read on the root too, for engines that do not inherit custom properties into getComputedStyle). */
function themed(): number | undefined {
  for (const el of [root.value, document.documentElement]) {
    const ms = el ? cssTime(getComputedStyle(el).getPropertyValue('--weft-duration-toast')) : undefined
    if (ms !== undefined) return ms
  }
  return undefined
}
function restart() {
  stop()
  const ms = props.duration ?? themed() ?? 4000
  total.value = ms > 0 ? ms : Infinity
  left = total.value
  run.value++
  go()
}
onMounted(restart)
watch(() => props.duration, restart)
watch(() => props.paused, (on) => (on ? stop() : go()))

/** Play the leave transition, then emit close. */
let leaveTimer: ReturnType<typeof setTimeout> | undefined
function leave() {
  if (leaving.value) return
  stop()
  leaving.value = true
  const ms = root.value ? leaveTime(root.value) : 0
  if (ms <= 0) emit('close')
  else leaveTimer = setTimeout(() => emit('close'), ms)
}
onScopeDispose(() => {
  if (timer !== undefined) clearTimeout(timer)
  if (leaveTimer !== undefined) clearTimeout(leaveTimer)
})

function act(action: ToastAction, event: MouseEvent) {
  action.onClick?.(event)
  leave()
}
function onKeydown(event: KeyboardEvent) {
  if (event.key !== 'Escape' || event.defaultPrevented) return
  event.preventDefault()
  leave()
}

// Swiped away on touch: past a third of its width, or a flick.
const swipe = createSwipe(() => {
  const rtl = isRtl(root.value)
  const towardsEnd = props.swipe === 'end'
  return { axis: 'x', sign: towardsEnd !== rtl ? 1 : -1 }
})
let width = 0
function onSwipeStart(event: PointerEvent) {
  if (event.pointerType !== 'touch' || !root.value || leaving.value) return
  width = root.value.offsetWidth
  swipe.start(event)
}
function onSwipeMove(event: PointerEvent) {
  const at = swipe.move(event)
  if (at !== null) root.value?.style.setProperty('--weft-toast-swipe', `${at}px`)
}
function onSwipeEnd(event: PointerEvent) {
  const end = swipe.end(event)
  if (!end) return
  if (end.distance > width / 3 || (end.distance > 8 && end.speed > FLICK)) return leave()
  root.value?.style.removeProperty('--weft-toast-swipe')
}

const bar = computed(() => (props.progress && Number.isFinite(total.value) ? [run.value] : []))
</script>

<script lang="ts">
/** px per ms over its last moves: a swipe at least this fast at its end throws the toast away. */
const FLICK = 0.5
</script>

<template>
  <div
    ref="root"
    class="weft-toast"
    :data-tone="tone"
    :data-swipe="swipe"
    :data-paused="paused ? 'true' : undefined"
    :data-state="leaving ? 'closed' : 'open'"
    @keydown="onKeydown"
    @pointerdown="onSwipeStart"
    @pointermove="onSwipeMove"
    @pointerup="onSwipeEnd"
    @pointercancel="onSwipeEnd"
  >
    <span v-if="$slots.leading || icon" class="weft-toast__leading"><slot name="leading"><Icon v-if="icon" :name="icon" /></slot></span>
    <div class="weft-toast__text">
      <slot>
        <p v-if="title" class="weft-toast__title">{{ title }}</p>
        <p v-if="description" class="weft-toast__description">{{ description }}</p>
      </slot>
    </div>
    <div v-if="$slots.actions || actions?.length" class="weft-toast__actions">
      <slot name="actions">
        <Button v-for="action in actions" :key="action.label" size="sm" variant="outline" tone="neutral" :label="action.label" @click="act(action, $event)" />
      </slot>
    </div>
    <CloseButton v-if="close" class="weft-toast__close" size="sm" :label="closeLabel" @click="leave" />
    <div v-for="n in bar" :key="n" class="weft-toast__progress" aria-hidden="true" :style="{ '--weft-toast-time': `${total}ms` }"></div>
  </div>
</template>
