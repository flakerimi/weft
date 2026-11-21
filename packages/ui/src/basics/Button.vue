<script setup vapor lang="ts">
// A button, or a link drawn as one when it has `to`. Loading keeps its width
// (the content stays, hidden, under a spinner), stays focusable and blocks
// activation by mouse, keyboard and form submit (and, as a link, a middle
// click); disabled is the native attribute on a button and no address on a
// link. On touch an md button is at least --weft-hit-min (44 px) tall; a
// small or square one keeps its size and grows its hit area to
// --weft-hit-min through a pseudo-element.
import { useAttrs, useSlots } from 'vue'
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
    /** `sm`, `md` or `lg`. On touch an md button grows to 44 px; sm keeps its height and grows its hit area. */
    size?: Size
    /** A Lucide icon before the label; alone, the button is square and needs an `aria-label`. The leading slot replaces it. */
    icon?: string
    /** A Lucide icon after the label. The trailing slot replaces it. */
    trailingIcon?: string
    /** Square, its width equal to its height, whatever it holds. */
    square?: boolean
    /** Work in progress: shows a spinner, sets `aria-busy` and blocks activation, keeping width and focus. */
    loading?: boolean
    /** A Lucide icon to turn in place of the spinner while loading. */
    loadingIcon?: string
    /** Unavailable: a button is natively disabled; a link loses its address. */
    disabled?: boolean
    /** Drawn as on: the current one of a set. With `to` it is the current page (aria-current="page"). */
    active?: boolean
    /** Fill the width of the container. */
    block?: boolean
    /** An address: the button renders as an `<a>` that goes there (rel="noopener" when it is another site). */
    to?: string
    /** With `to`: the native rel; `noopener` is added for another site. */
    rel?: string
    /** With `to`: where to open it, as the native attribute: `_blank` for a new tab. */
    target?: string
    /** The native button type; `submit` submits its form. Ignored with `to`. */
    type?: 'button' | 'submit' | 'reset'
  }>(),
  { variant: 'solid', tone: 'primary', size: 'md', type: 'button' },
)

const emit = defineEmits<{
  /** Pressed by mouse, touch or keyboard; never while loading or disabled. */
  click: [event: MouseEvent]
}>()

defineSlots<{
  /** The label, in place of `label`. */
  default?: () => unknown
  /** Before the label, in place of `icon`. */
  leading?: () => unknown
  /** After the label, in place of `trailing-icon`. */
  trailing?: () => unknown
}>()

const slots = useSlots()
if (import.meta.env.DEV && !slots.default && !props.label && (props.icon || props.trailingIcon)) {
  const attrs = useAttrs()
  if (!attrs['aria-label'] && !attrs['aria-labelledby'] && !attrs.title) warnUnnamed()
}

function blocked(event: MouseEvent): boolean {
  if (!props.loading && !props.disabled) return false
  // As a disabled button would: no navigation, no form submit, no handlers.
  event.preventDefault()
  event.stopPropagation()
  return true
}

function onClick(event: MouseEvent) {
  if (!blocked(event)) emit('click', event)
}
</script>

<script lang="ts">
let warned = false
/** Once per page: an icon alone names nothing for assistive technology. */
function warnUnnamed() {
  if (warned) return
  warned = true
  console.warn(
    'Button: a button with only an icon needs a name; give it a label or an aria-label, which assistive technology reads in place of the icon.',
  )
}
</script>

<template>
  <component
    :is="to ? 'a' : 'button'"
    class="weft-button"
    :type="to ? undefined : type"
    :href="to && !disabled ? to : undefined"
    :rel="to && !disabled ? relFor(rel, isExternal(to)) : undefined"
    :target="to && !disabled ? target : undefined"
    :role="to && disabled ? 'link' : undefined"
    :disabled="!to && disabled ? true : undefined"
    :aria-disabled="loading || (to && disabled) ? 'true' : undefined"
    :aria-busy="loading ? 'true' : undefined"
    :aria-current="to && active ? 'page' : undefined"
    :data-variant="variant"
    :data-tone="tone"
    :data-size="size"
    :data-active="active ? 'true' : undefined"
    :data-block="block ? 'true' : undefined"
    :data-square="square || (!$slots.default && !label && (icon || trailingIcon)) ? 'true' : undefined"
    :data-loading="loading ? 'true' : undefined"
    @click="onClick"
    @auxclick="blocked"
  >
    <span class="weft-button__content">
      <slot name="leading"><Icon v-if="icon" :name="icon" /></slot>
      <slot><span v-if="label" class="weft-button__label">{{ label }}</span></slot>
      <slot name="trailing"><Icon v-if="trailingIcon" :name="trailingIcon" /></slot>
    </span>
    <span v-if="loading" class="weft-button__spinner">
      <Icon v-if="loadingIcon" class="weft-button__loading-icon" :name="loadingIcon" />
      <Spinner v-else />
    </span>
  </component>
</template>
