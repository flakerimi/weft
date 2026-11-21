<script setup vapor lang="ts">
// The inside of Dialog, Sheet and Drawer: a header with the heading (title
// and description, or the header slot), actions and the close button, then
// the body, which scrolls, and the footer. The heading keeps `title-id` on
// whatever it renders, the title or the header slot, so the overlay's
// aria-labelledby always names something.
import CloseButton from './CloseButton.vue'
import '../styles/base.css'
import '../styles/close-button.css'
import '../styles/overlay-panel.css'

defineProps<{
  /** The heading. */
  title?: string
  /** A sentence under the title. */
  description?: string
  /** The id the heading carries, for the overlay's aria-labelledby. */
  titleId?: string
  /** The id the description carries, for the overlay's aria-describedby. */
  descriptionId?: string
  /** Show the close button. */
  close?: boolean
  /** The close button's name for assistive technology. */
  closeLabel?: string
}>()

const emit = defineEmits<{
  /** The close button was pressed, or a slot called `close()`. */
  close: []
}>()

defineSlots<{
  /** The heading, in place of `title` and `description`. */
  header?: () => unknown
  /** Controls in the header, before the close button. */
  actions?: () => unknown
  /** The content, which scrolls when long; `close()` closes the overlay. */
  body?: (props: { close: () => void }) => unknown
  /** Below the body, for buttons; `close()` closes the overlay. */
  footer?: (props: { close: () => void }) => unknown
}>()

const dismiss = () => emit('close')
</script>

<template>
  <div v-if="title || description || close || $slots.header || $slots.actions" class="weft-overlay__header">
    <div class="weft-overlay__heading" :id="$slots.header ? titleId : undefined">
      <slot name="header">
        <h2 v-if="title" :id="titleId" class="weft-overlay__title">{{ title }}</h2>
        <p v-if="description" :id="descriptionId" class="weft-overlay__description">{{ description }}</p>
      </slot>
    </div>
    <div v-if="$slots.actions" class="weft-overlay__actions"><slot name="actions" /></div>
    <CloseButton v-if="close" class="weft-overlay__close" :label="closeLabel" @click="emit('close')" />
  </div>
  <div v-if="$slots.body" class="weft-overlay__body"><slot name="body" :close="dismiss" /></div>
  <div v-if="$slots.footer" class="weft-overlay__footer"><slot name="footer" :close="dismiss" /></div>
</template>
