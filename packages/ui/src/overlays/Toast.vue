<script setup vapor lang="ts">
// One toast: a short message with an optional icon, actions and close
// button. It leaves on its own after `duration` ms (by default the theme's
// --weft-duration-toast) by emitting `close`, and its time stops while
// `paused` (a Toaster pauses every toast while it is hovered or has focus),
// resuming with what it had left. Escape on it closes it. Toaster renders
// these from useToast(); render one yourself only inside a live region.
import { onMounted, onScopeDispose, useTemplateRef, watch } from 'vue'
import Button from '../basics/Button.vue'
import Icon from '../basics/Icon.vue'
import type { Tone } from '../types.ts'
import { cssTime, type ToastAction } from './toast.ts'
import '../styles/base.css'
import '../styles/tone.css'
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
    /** ms before it leaves on its own; default `--weft-duration-toast`; 0 stays until closed. */
    duration?: number
    /** Its time stands still: hovered, or focus inside. */
    paused?: boolean
    /** Show the close button. */
    close?: boolean
    /** The close button's name for assistive technology. */
    closeLabel?: string
    /** Buttons after the text; each closes the toast after it runs. */
    actions?: ToastAction[]
  }>(),
  { tone: 'neutral', close: true, closeLabel: 'Close' },
)

const emit = defineEmits<{
  /** It should go: its time ran out, or it was closed or acted on. */
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
let left = Infinity
let started = 0
let timer: ReturnType<typeof setTimeout> | undefined

function stop() {
  if (timer === undefined) return
  clearTimeout(timer)
  timer = undefined
  left -= Date.now() - started
}
function run() {
  if (timer !== undefined || props.paused || !Number.isFinite(left)) return
  started = Date.now()
  timer = setTimeout(() => emit('close'), Math.max(0, left))
}
/** The theme's --weft-duration-toast as it reaches this toast (read on the root too, for engines that do not inherit custom properties into getComputedStyle). */
function themed(): number | undefined {
  for (const el of [root.value, document.documentElement]) {
    const ms = el ? cssTime(getComputedStyle(el).getPropertyValue('--weft-duration-toast')) : undefined
    if (ms !== undefined) return ms
  }
  return undefined
}
onMounted(() => {
  const ms = props.duration ?? themed() ?? 4000
  left = ms > 0 ? ms : Infinity
  run()
})
watch(() => props.paused, (on) => (on ? stop() : run()))
onScopeDispose(() => timer !== undefined && clearTimeout(timer))

function act(action: ToastAction, event: MouseEvent) {
  action.onClick?.(event)
  emit('close')
}
function onKeydown(event: KeyboardEvent) {
  if (event.key !== 'Escape' || event.defaultPrevented) return
  event.preventDefault()
  emit('close')
}
</script>

<template>
  <div ref="root" class="weft-toast" :data-tone="tone" @keydown="onKeydown">
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
    <button v-if="close" type="button" class="weft-toast__close" :aria-label="closeLabel" @click="emit('close')">
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M18 6 6 18" /><path d="m6 6 12 12" /></svg>
    </button>
  </div>
</template>
