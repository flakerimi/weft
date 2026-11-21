<script setup vapor lang="ts">
// A modal dialog (WAI-ARIA APG, Dialog (Modal)): opened from its trigger
// (the default slot) or with v-model:open, centred over a backdrop, focus
// trapped inside, the page still. Escape, the backdrop and the close button
// close it, and focus returns to the trigger. Full-screen when the viewport
// is narrow (under 640 px), or always with `fullscreen`.
import { computed, useTemplateRef } from 'vue'
import { useWidth } from '../adaptive/useWidth.ts'
import { overlayRoot } from './overlay.ts'
import { useModal } from './modal.ts'
import '../styles/base.css'
import '../styles/overlay.css'
import '../styles/dialog.css'

defineOptions({ inheritAttrs: false })

const props = withDefaults(
  defineProps<{
    /** Open or closed, bound with `v-model:open`. */
    open?: boolean
    /** Whether it starts open when `v-model:open` is not bound. */
    defaultOpen?: boolean
    /** The heading, which names the dialog. Without one, give the dialog an `aria-label`. */
    title?: string
    /** A sentence under the title, which describes the dialog. */
    description?: string
    /** Show the close button (×) in the header. */
    close?: boolean
    /** The close button's name for assistive technology. */
    closeLabel?: string
    /** Escape and a press on the backdrop close it. */
    dismissible?: boolean
    /** Full-screen at every width; it is full-screen when narrow anyway. */
    fullscreen?: boolean
    /** `alertdialog` for a dialog that needs an answer before going on (ConfirmDialog uses it). */
    role?: 'dialog' | 'alertdialog'
  }>(),
  { open: undefined, defaultOpen: false, close: true, closeLabel: 'Close', dismissible: true, role: 'dialog' },
)

const emit = defineEmits<{
  /** Opened or closed, by the trigger, Escape, the backdrop or a close. */
  'update:open': [open: boolean]
}>()

defineSlots<{
  /** The trigger: one element (a Button), which opens it. */
  default?: (props: { open: boolean }) => unknown
  /** The header's heading, in place of `title` and `description`. */
  header?: () => unknown
  /** Controls in the header, before the close button. */
  actions?: () => unknown
  /** The dialog's content; `close()` closes it. */
  body?: (props: { close: () => void }) => unknown
  /** Below the body, for the dialog's buttons; `close()` closes it. */
  footer?: (props: { close: () => void }) => unknown
}>()

const m = useModal(props, emit, { prefix: 'dialog' })
const open = m.open
const backdrop = useTemplateRef<HTMLElement>('backdrop')
// The backdrop covers the viewport, so its width is the viewport's.
const width = useWidth(backdrop)
const full = computed(() => props.fullscreen || width.value === 'narrow')
</script>

<template>
  <span v-if="$slots.default" ref="wrap" class="weft-overlay-trigger" @click="m.onTrigger"><slot :open="open" /></span>
  <Teleport v-if="m.mounted.value && open" :to="overlayRoot()">
    <div
      class="weft-overlay"
      :dir="m.context.attrs.value.dir"
      :data-weft-input="m.context.attrs.value['data-weft-input']"
      :data-weft-surface="m.context.attrs.value['data-weft-surface']"
      :data-weft-scheme="m.context.attrs.value['data-weft-scheme']"
    >
      <div ref="backdrop" class="weft-overlay-backdrop" @click="m.onBackdrop"></div>
      <div
        ref="panel"
        v-bind="$attrs"
        :id="m.ids.panel"
        class="weft-dialog"
        :role="role"
        aria-modal="true"
        :aria-labelledby="title ? m.ids.title : undefined"
        :aria-describedby="description ? m.ids.description : undefined"
        :data-fullscreen="full ? 'true' : undefined"
      >
        <div v-if="title || description || close || $slots.header || $slots.actions" class="weft-overlay__header">
          <div class="weft-overlay__heading">
            <slot name="header">
              <h2 v-if="title" :id="m.ids.title" class="weft-overlay__title">{{ title }}</h2>
              <p v-if="description" :id="m.ids.description" class="weft-overlay__description">{{ description }}</p>
            </slot>
          </div>
          <div v-if="$slots.actions" class="weft-overlay__actions"><slot name="actions" /></div>
          <button v-if="close" type="button" class="weft-overlay__close" :aria-label="closeLabel" @click="m.close">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M18 6 6 18" /><path d="m6 6 12 12" /></svg>
          </button>
        </div>
        <div v-if="$slots.body" class="weft-overlay__body"><slot name="body" :close="m.close" /></div>
        <div v-if="$slots.footer" class="weft-overlay__footer"><slot name="footer" :close="m.close" /></div>
      </div>
    </div>
  </Teleport>
</template>
