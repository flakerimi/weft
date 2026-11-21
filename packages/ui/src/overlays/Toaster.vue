<script setup vapor lang="ts">
// Where useToast()'s toasts appear: a labelled region in a corner (or the
// middle of an edge), its list a polite live region, so each new toast is
// announced without moving focus. While the pointer is over it or focus is
// inside, every toast's time stands still. F8 moves focus to it, to reach a
// toast's actions; closing the focused toast leaves focus in the region.
// Put one Toaster in the app's layout; it renders into the overlay root
// once mounted, so the server renders none.
import { computed, onScopeDispose, ref, useTemplateRef } from 'vue'
import { overlayRoot, useMounted } from './overlay.ts'
import { useToast } from './toast.ts'
import Toast from './Toast.vue'
import '../styles/base.css'
import '../styles/toaster.css'

const props = withDefaults(
  defineProps<{
    /** Where toasts stack: a corner, or the middle of the top or bottom edge. */
    position?: 'top-start' | 'top-center' | 'top-end' | 'bottom-start' | 'bottom-center' | 'bottom-end'
    /** How many show at once, the newest; the rest wait their turn. */
    max?: number
    /** The region's name for assistive technology. */
    label?: string
  }>(),
  { position: 'bottom-end', max: 5, label: 'Notifications' },
)

const queue = useToast()
const shown = computed(() => queue.toasts.value.slice(-Math.max(1, props.max)))
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
      :data-position="position"
      @pointerenter="hovered = true"
      @pointerleave="hovered = false"
      @focusin="focusInside = true"
      @focusout="onFocusout"
    >
      <ol class="weft-toaster__list" aria-live="polite" aria-relevant="additions text">
        <li v-for="t in shown" :key="t.id" class="weft-toaster__item" :data-toast="t.id">
          <Toast
            :title="t.title"
            :description="t.description"
            :tone="t.tone"
            :icon="t.icon"
            :duration="t.duration"
            :close="t.close ?? true"
            :actions="t.actions"
            :paused="paused"
            @close="remove(t.id)"
          />
        </li>
      </ol>
    </section>
  </Teleport>
</template>
