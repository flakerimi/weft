<script setup vapor lang="ts">
// What a list or a page shows when loading it failed: an EmptyState in the
// danger tone, as an alert (role="alert", announced when it appears), with
// the error's message (an Error, a string, or the wire format's `{ error: {
// code, message } }`, its code shown small) and, with `retry`, a button
// that emits `retry`. While `loading` (retrying) that button spins and
// takes no presses. On touch its buttons are at least --weft-hit-min tall.
import { computed } from 'vue'
import Button from '../basics/Button.vue'
import Icon from '../basics/Icon.vue'
import EmptyState from './EmptyState.vue'
import { errorText, type ActionItem } from './actions.ts'
import type { Size } from '../types.ts'
import '../styles/base.css'
import '../styles/error-state.css'

const props = withDefaults(
  defineProps<{
    /** What failed, in bold. */
    title?: string
    /** What happened; default the error's message. */
    description?: string
    /** The error: an Error, a string, or a wire-format error body. */
    error?: unknown
    /** A Lucide icon in place of the warning sign. */
    icon?: string
    /** The title's heading level; without it the title is bold text. */
    level?: 2 | 3 | 4 | 5 | 6
    /** Show a button that emits `retry`. */
    retry?: boolean
    /** The retry button's text. */
    retryLabel?: string
    /** Retrying: the retry button spins and takes no presses. */
    loading?: boolean
    /** More buttons after the retry button. */
    actions?: ActionItem[]
    /** `outline`, `soft` or `ghost`, as EmptyState. */
    variant?: 'outline' | 'soft' | 'ghost'
    /** `sm`, `md` or `lg`. */
    size?: Size
  }>(),
  { title: 'Something went wrong', retryLabel: 'Try again', variant: 'outline', size: 'md' },
)

const emit = defineEmits<{
  /** The retry button was pressed. */
  retry: []
}>()

defineSlots<{
  /** More content under the message. */
  default?: () => unknown
  /** The buttons, in place of the retry button and `actions`. */
  actions?: () => unknown
  /** Under everything: a support link. */
  footer?: () => unknown
}>()

const text = computed(() => errorText(props.error))
const message = computed(() => props.description ?? text.value.message)
</script>

<template>
  <EmptyState class="weft-error-state" role="alert" :title="title" :level="level" :variant="variant" :size="size">
    <template #leading>
      <Icon v-if="icon" :name="icon" />
      <svg v-else class="weft-error-state__glyph" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="m21.73 18-8-14a2 2 0 0 0-3.48 0l-8 14A2 2 0 0 0 4 21h16a2 2 0 0 0 1.73-3" /><path d="M12 9v4" /><path d="M12 17h.01" /></svg>
    </template>
    <p v-if="message" class="weft-empty-state__description weft-error-state__description">{{ message }}</p>
    <p v-if="text.code" class="weft-error-state__code">{{ text.code }}</p>
    <slot />
    <template v-if="$slots.actions || retry || actions?.length" #actions>
      <slot name="actions">
        <Button v-if="retry" :label="retryLabel" :loading="loading" @click="emit('retry')" />
        <Button
          v-for="action in actions"
          :key="action.label"
          :label="action.label"
          :icon="action.icon"
          :to="action.to"
          :disabled="action.disabled"
          :loading="action.loading"
          :tone="action.tone ?? 'neutral'"
          :variant="action.variant ?? 'outline'"
          @click="action.onClick?.($event)"
        />
      </slot>
    </template>
    <template v-if="$slots.footer" #footer><slot name="footer" /></template>
  </EmptyState>
</template>
