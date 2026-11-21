<script setup vapor lang="ts">
// What a list, a search or a page shows when it has nothing yet: an icon,
// a title, a sentence, the actions that would fill it, and a footer (a
// help link). The title is a heading at `level` when given, else bold
// text, so it fits the page's outline. Loading turns the icon into a
// spinner and marks it busy. Centred; on touch its buttons are at least
// --weft-hit-min tall (Button's rule); when narrow the actions stack.
import Button from '../basics/Button.vue'
import Icon from '../basics/Icon.vue'
import Spinner from '../basics/Spinner.vue'
import type { Size } from '../types.ts'
import type { ActionItem } from './actions.ts'
import '../styles/base.css'
import '../styles/empty-state.css'

withDefaults(
  defineProps<{
    /** What is empty: "No customers yet". */
    title?: string
    /** What to do about it, under the title. */
    description?: string
    /** A Lucide icon above the title; the leading slot replaces it. */
    icon?: string
    /** The title's heading level; without it the title is bold text, not a heading. */
    level?: 2 | 3 | 4 | 5 | 6
    /** Buttons under the text: `{ label, onClick, to, icon, tone, variant }`; the first is solid, the rest outlined. */
    actions?: ActionItem[]
    /** `outline` a dashed border, `soft` a tinted box, `ghost` none. */
    variant?: 'outline' | 'soft' | 'ghost'
    /** `sm`, `md` or `lg`: its space and icon size. */
    size?: Size
    /** Still looking: a spinner in place of the icon, and aria-busy. */
    loading?: boolean
  }>(),
  { variant: 'outline', size: 'md' },
)

defineSlots<{
  /** More content under the description. */
  default?: () => unknown
  /** Above the title, in place of `icon`. */
  leading?: () => unknown
  /** The buttons, in place of `actions`. */
  actions?: () => unknown
  /** Under everything: a help link. */
  footer?: () => unknown
}>()
</script>

<template>
  <div class="weft-empty-state" :data-variant="variant" :data-size="size" :aria-busy="loading ? 'true' : undefined">
    <div v-if="loading || $slots.leading || icon" class="weft-empty-state__icon">
      <Spinner v-if="loading" />
      <slot v-else name="leading"><Icon v-if="icon" :name="icon" /></slot>
    </div>
    <component :is="level ? `h${level}` : 'p'" v-if="title" class="weft-empty-state__title">{{ title }}</component>
    <p v-if="description" class="weft-empty-state__description">{{ description }}</p>
    <div v-if="$slots.default" class="weft-empty-state__body"><slot /></div>
    <div v-if="$slots.actions || actions?.length" class="weft-empty-state__actions">
      <slot name="actions">
        <Button
          v-for="(action, i) in actions"
          :key="action.label"
          :label="action.label"
          :icon="action.icon"
          :to="action.to"
          :disabled="action.disabled"
          :loading="action.loading"
          :tone="action.tone ?? (i === 0 ? 'primary' : 'neutral')"
          :variant="action.variant ?? (i === 0 ? 'solid' : 'outline')"
          @click="action.onClick?.($event)"
        />
      </slot>
    </div>
    <div v-if="$slots.footer" class="weft-empty-state__footer"><slot name="footer" /></div>
  </div>
</template>
