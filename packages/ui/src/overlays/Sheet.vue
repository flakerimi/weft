<script setup vapor lang="ts">
// A panel that slides over the page from one side (a slideover), or up from
// the bottom (a bottom sheet): a modal dialog (WAI-ARIA APG, Dialog
// (Modal)) like Dialog, opened from its trigger or with v-model:open, focus
// trapped inside, the page still. Escape, the backdrop and the close button
// close it, and focus returns to the trigger. `start` and `end` follow the
// text direction. Popover and DropdownMenu take this form on touch.
import { overlayRoot } from './overlay.ts'
import { useModal } from './modal.ts'
import '../styles/base.css'
import '../styles/overlay.css'
import '../styles/sheet.css'

defineOptions({ inheritAttrs: false })

const props = withDefaults(
  defineProps<{
    /** Open or closed, bound with `v-model:open`. */
    open?: boolean
    /** Whether it starts open when `v-model:open` is not bound. */
    defaultOpen?: boolean
    /** The heading, which names the sheet. Without one, give the sheet an `aria-label`. */
    title?: string
    /** A sentence under the title, which describes the sheet. */
    description?: string
    /** Show the close button (×) in the header. */
    close?: boolean
    /** The close button's name for assistive technology. */
    closeLabel?: string
    /** Escape and a press on the backdrop close it. */
    dismissible?: boolean
    /** The side it comes from: `start` and `end` (the inline sides, mirrored right to left), `top` or `bottom`. */
    side?: 'start' | 'end' | 'top' | 'bottom'
  }>(),
  { open: undefined, defaultOpen: false, close: true, closeLabel: 'Close', dismissible: true, side: 'end' },
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
  /** The sheet's content, which scrolls when long; `close()` closes it. */
  body?: (props: { close: () => void }) => unknown
  /** Below the body, for the sheet's buttons; `close()` closes it. */
  footer?: (props: { close: () => void }) => unknown
}>()

const m = useModal(props, emit, { prefix: 'sheet' })
const open = m.open
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
      <div class="weft-overlay-backdrop" @click="m.onBackdrop"></div>
      <div
        ref="panel"
        v-bind="$attrs"
        :id="m.ids.panel"
        class="weft-sheet"
        role="dialog"
        aria-modal="true"
        :aria-labelledby="title ? m.ids.title : undefined"
        :aria-describedby="description ? m.ids.description : undefined"
        :data-side="side"
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
