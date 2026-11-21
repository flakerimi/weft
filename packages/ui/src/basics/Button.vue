<script setup vapor lang="ts">
// A button, or a link drawn as one when it has `to`. Loading keeps its width
// (the content stays, hidden, under a spinner), stays focusable and blocks
// activation by mouse, keyboard and form submit; disabled is the native
// attribute on a button and no address on a link. On touch its hit area grows
// to --weft-hit-min through a pseudo-element, so its drawn size stays.
import { isExternal, relFor } from './address.ts'
import Icon from './Icon.vue'
import Spinner from './Spinner.vue'
import type { Size, Tone, Variant } from '../types.ts'

const props = withDefaults(
  defineProps<{
    /** The button's text; the default slot replaces it. */
    label?: string
    /** `solid` for the main action, `soft` and `outline` for secondary ones, `ghost` for the quietest. */
    variant?: Variant
    /** The colour and what it says: `danger` for destructive actions. */
    tone?: Tone
    /** `sm`, `md` or `lg`: the drawn height, which touch does not change. */
    size?: Size
    /** A Lucide icon before the label; alone, the button is square and needs an `aria-label`. */
    icon?: string
    /** A Lucide icon after the label. */
    trailingIcon?: string
    /** Work in progress: shows a spinner, sets `aria-busy` and blocks activation, keeping width and focus. */
    loading?: boolean
    /** Unavailable: a button is natively disabled; a link loses its address. */
    disabled?: boolean
    /** Fill the width of the container. */
    block?: boolean
    /** An address: the button renders as an `<a>` that goes there (rel="noopener" when it is another site). */
    to?: string
    /** The native button type; `submit` submits its form. Ignored with `to`. */
    type?: 'button' | 'submit' | 'reset'
  }>(),
  { variant: 'solid', tone: 'primary', size: 'md', type: 'button' },
)

const emit = defineEmits<{
  /** Pressed by mouse, touch or keyboard; never while loading or disabled. */
  click: [event: MouseEvent]
}>()

function onClick(event: MouseEvent) {
  if (props.loading || props.disabled) {
    // As a disabled button would: no navigation, no form submit, no handlers.
    event.preventDefault()
    event.stopPropagation()
    return
  }
  emit('click', event)
}
</script>

<template>
  <component
    :is="to ? 'a' : 'button'"
    class="weft-button"
    :type="to ? undefined : type"
    :href="to && !disabled ? to : undefined"
    :rel="to && !disabled ? relFor(undefined, isExternal(to)) : undefined"
    :role="to && disabled ? 'link' : undefined"
    :disabled="!to && disabled ? true : undefined"
    :aria-disabled="loading || (to && disabled) ? 'true' : undefined"
    :aria-busy="loading ? 'true' : undefined"
    :data-variant="variant"
    :data-tone="tone"
    :data-size="size"
    :data-block="block ? 'true' : undefined"
    :data-square="!$slots.default && !label && (icon || trailingIcon) ? 'true' : undefined"
    :data-loading="loading ? 'true' : undefined"
    @click="onClick"
  >
    <span class="weft-button__content">
      <Icon v-if="icon" :name="icon" />
      <slot><span v-if="label" class="weft-button__label">{{ label }}</span></slot>
      <Icon v-if="trailingIcon" :name="trailingIcon" />
    </span>
    <span v-if="loading" class="weft-button__spinner"><Spinner /></span>
  </component>
</template>
