<script setup vapor lang="ts">
// A message in the page that asks for attention: a title, a description, an
// icon, actions and a close button, drawn in a tone. It has no live role of
// its own, since a message present when the page loads is not news: give
// `role="alert"` (urgent) or `role="status"` to one that appears later and
// must be announced. Closed with its button, it leaves the page through
// `v-model:open`. Actions sit under the text, or beside it when horizontal;
// when narrow they wrap under it anyway. On touch the close button's hit
// area is at least --weft-hit-min.
import Button from '../basics/Button.vue'
import Icon from '../basics/Icon.vue'
import CloseButton from '../overlays/CloseButton.vue'
import { useControllable } from '../headless/useControllable.ts'
import type { Tone, Variant } from '../types.ts'
import type { ActionItem } from './actions.ts'
import '../styles/base.css'
import '../styles/tone.css'
import '../styles/alert.css'

const props = withDefaults(
  defineProps<{
    /** The message, in bold. */
    title?: string
    /** More about it, under the title; the default slot replaces it. */
    description?: string
    /** A Lucide icon before the text; the leading slot replaces it. */
    icon?: string
    /** The colour and what it says: `danger` for a failure, `success` for done. */
    tone?: Tone
    /** `soft` (default) tinted, `solid` filled, `outline` bordered, `ghost` text in the tone. */
    variant?: Variant
    /** `vertical`: actions under the text; `horizontal`: beside it. */
    orientation?: 'vertical' | 'horizontal'
    /** Buttons: `{ label, onClick, to, icon, tone, variant }`; the actions slot replaces them. */
    actions?: ActionItem[]
    /** Show a close button. */
    close?: boolean
    /** The close button's name for assistive technology. */
    closeLabel?: string
    /** Shown, bound with `v-model:open`; the close button sets it false. */
    open?: boolean
  }>(),
  { tone: 'primary', variant: 'soft', orientation: 'vertical', closeLabel: 'Close', open: undefined },
)

const emit = defineEmits<{
  /** Closed with its button (false). */
  'update:open': [open: boolean]
}>()

defineSlots<{
  /** The description, in place of `description`. */
  default?: () => unknown
  /** The title, in place of `title`. */
  title?: () => unknown
  /** Before the text, in place of `icon`. */
  leading?: () => unknown
  /** The buttons, in place of `actions`. */
  actions?: () => unknown
}>()

const open = useControllable(props, 'open', emit, true)
</script>

<template>
  <div v-if="open" class="weft-alert" :data-tone="tone" :data-variant="variant" :data-orientation="orientation">
    <span v-if="$slots.leading || icon" class="weft-alert__leading"><slot name="leading"><Icon v-if="icon" :name="icon" /></slot></span>
    <div class="weft-alert__body">
      <div class="weft-alert__text">
        <p v-if="$slots.title || title" class="weft-alert__title"><slot name="title">{{ title }}</slot></p>
        <div v-if="$slots.default || description" class="weft-alert__description"><slot>{{ description }}</slot></div>
      </div>
      <div v-if="$slots.actions || actions?.length" class="weft-alert__actions">
        <slot name="actions">
          <Button
            v-for="action in actions"
            :key="action.label"
            size="sm"
            :label="action.label"
            :icon="action.icon"
            :to="action.to"
            :disabled="action.disabled"
            :loading="action.loading"
            :tone="action.tone ?? (variant === 'solid' ? 'neutral' : tone)"
            :variant="action.variant ?? 'outline'"
            @click="action.onClick?.($event)"
          />
        </slot>
      </div>
    </div>
    <CloseButton v-if="close" class="weft-alert__close" :label="closeLabel" @click="open = false" />
  </div>
</template>
