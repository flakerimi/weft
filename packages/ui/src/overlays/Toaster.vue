<script setup vapor lang="ts">
// Where useToast()'s toasts appear: a labelled region in a corner (or the
// middle of an edge). Each new toast is announced without moving focus:
// background toasts politely, foreground ones (danger by default) at once.
// At most `max` stay; a new one beyond that drops the oldest. While the
// pointer is over it or focus is inside, every toast's time stands still.
// F8 moves focus to it, to reach a toast's actions, even past a modal's
// focus trap; closing the focused toast leaves focus in the region. On
// touch, a toast swipes away towards the region's side.
// Put one Toaster in the app's layout; it renders into the overlay root
// once mounted, so the server renders none.
import { computed, onScopeDispose, ref, useTemplateRef, watch } from 'vue'
import { overlayRoot, useMounted } from './overlay.ts'
import { useToast, type ToastMessage } from './toast.ts'
import Toast from './Toast.vue'
import '../styles/base.css'
import '../styles/toaster.css'

const props = withDefaults(
  defineProps<{
    /** Where toasts stack: a corner, or the middle of the top or bottom edge. */
    position?: 'top-start' | 'top-center' | 'top-end' | 'bottom-start' | 'bottom-center' | 'bottom-end'
    /** How many stay at once; a new toast beyond that drops the oldest. */
    max?: number
    /** The region's name for assistive technology. */
    label?: string
  }>(),
  { position: 'bottom-end', max: 5, label: 'Notifications' },
)

const queue = useToast()
const shown = computed(() => queue.toasts.value.slice(-Math.max(1, props.max)))
// Drop the oldest beyond max, as Nuxt UI does, rather than keep them waiting.
watch(
  [queue.toasts, () => props.max],
  ([list, max]) => {
    for (const t of list.slice(0, Math.max(0, list.length - Math.max(1, max)))) queue.remove(t.id)
  },
  { immediate: true },
)
// What the announcers say: each toast's text once, at its own politeness.
const polite = ref('')
const assertive = ref('')
const said = new Set<string>()
const textOf = (t: ToastMessage) => [t.title, t.description].filter(Boolean).join(' ')
watch(
  queue.toasts,
  (list) => {
    for (const t of list) {
      if (said.has(t.id)) continue
      said.add(t.id)
      const loud = (t.type ?? (t.tone === 'danger' ? 'foreground' : 'background')) === 'foreground'
      if (loud) assertive.value = textOf(t)
      else polite.value = textOf(t)
    }
    for (const id of said) if (!list.some((t) => t.id === id)) said.delete(id)
  },
  { immediate: true },
)
const swipe = computed(() => (props.position.endsWith('start') ? 'start' : 'end'))
const mounted = useMounted()
const region = useTemplateRef<HTMLElement>('region')
const hovered = ref(false)
const focusInside = ref(false)
const paused = computed(() => hovered.value || focusInside.value)

function onFocusout(event: FocusEvent) {
  const to = event.relatedTarget as Node | null
  if (!to || !region.value?.contains(to)) focusInside.value = false
}
function remove(id: string) {
  const el = region.value
  const item = el?.querySelector(`[data-toast="${CSS.escape(id)}"]`)
  // Focus in the toast that goes would fall to the body; keep it in the region.
  if (el && item?.contains(document.activeElement)) el.focus()
  queue.remove(id)
}

function onHotkey(event: KeyboardEvent) {
  if (event.key === 'F8' && !event.defaultPrevented && region.value) {
    event.preventDefault()
    region.value.focus()
  }
}
if (typeof document !== 'undefined') {
  document.addEventListener('keydown', onHotkey)
  onScopeDispose(() => document.removeEventListener('keydown', onHotkey))
}
</script>

<template>
  <Teleport v-if="mounted" :to="overlayRoot()">
    <section
      ref="region"
      class="weft-toaster"
      role="region"
      :aria-label="label"
      tabindex="-1"
      data-weft-trap-exempt
      :data-position="position"
      @pointerenter="hovered = true"
      @pointerleave="hovered = false"
      @focusin="focusInside = true"
      @focusout="onFocusout"
    >
      <div class="weft-visually-hidden" aria-live="polite" aria-atomic="true">{{ polite }}</div>
      <div class="weft-visually-hidden" aria-live="assertive" aria-atomic="true">{{ assertive }}</div>
      <ol class="weft-toaster__list">
        <li v-for="t in shown" :key="t.id" class="weft-toaster__item" :data-toast="t.id">
          <Toast
            :title="t.title"
            :description="t.description"
            :tone="t.tone"
            :icon="t.icon"
            :duration="t.duration"
            :close="t.close ?? true"
            :actions="t.actions"
            :progress="t.progress ?? true"
            :swipe="swipe"
            :paused="paused"
            @close="remove(t.id)"
          />
        </li>
      </ol>
    </section>
  </Teleport>
</template>
