<script setup vapor lang="ts">
// A closing call-to-action band or card: an optional eyebrow headline, a
// heading (h2 by default, or `h${level}`), a supporting sentence, action
// buttons, and an optional aside (an illustration or install box). Drawn as a
// raised card (`card`, default), a soft tint in `tone` (`soft`), a solid fill
// in `tone` (`solid`, where default buttons switch to neutral so they contrast
// with the fill), or unboxed (`plain`). `vertical` (the fixed default) stacks
// the copy and actions; `horizontal` puts the copy at the start and the
// actions or `aside` at the end when regular or wide, stacking when narrow, in
// CSS first through @container.
import { computed, useTemplateRef } from 'vue'
import { useWidth } from '../adaptive/useWidth.ts'
import Button from '../basics/Button.vue'
import type { ActionItem } from '../feedback/actions.ts'
import type { Tone } from '../types.ts'
import '../styles/base.css'
import '../styles/tone.css'
import '../styles/cta.css'

const props = withDefaults(
  defineProps<{
    /** Short eyebrow text above the title; the headline slot replaces it. */
    headline?: string
    /** The call-to-action heading, rendered as an h2 by default (or `h${level}`). */
    title?: string
    /** A sentence under the title. */
    description?: string
    /** Buttons: `{ label, onClick, to, icon, tone, variant }`; the first is solid, the rest outlined. */
    actions?: ActionItem[]
    /** `card` (bordered raised surface, default), `soft` (tinted in `tone`), `solid` (filled in `tone`), or `plain` (no box). */
    variant?: 'card' | 'soft' | 'solid' | 'plain'
    /** The colour of `soft` and `solid`, and of the first button otherwise. */
    tone?: Tone
    /** `vertical` (default: copy above actions) or `horizontal` (copy at the start, actions or aside at the end when regular or wide, stacked when narrow). */
    orientation?: 'vertical' | 'horizontal'
    /** Alignment when vertical: `center` (default when vertical) or `start` (default when horizontal). */
    align?: 'start' | 'center'
    /** The heading level, 2 to 6, to fit the page's outline. */
    level?: 2 | 3 | 4 | 5 | 6
  }>(),
  { variant: 'card', tone: 'primary', orientation: 'vertical', align: undefined, level: 2 },
)

defineSlots<{
  /** Above the title, in place of `headline`. */
  headline?: () => unknown
  /** Inside the heading, in place of `title`. */
  title?: () => unknown
  /** Under the title, in place of `description`. */
  description?: () => unknown
  /** Extra content under the description. */
  default?: () => unknown
  /** The call-to-action controls, in place of `actions`. */
  actions?: () => unknown
  /** An illustration, picture or code panel beside the copy (or below it when vertical or narrow). */
  aside?: () => unknown
}>()

const root = useTemplateRef<HTMLElement>('root')
const width = useWidth(root)
const widthId = width.id
const resolvedAlign = computed(() => props.align ?? (props.orientation === 'horizontal' ? 'start' : 'center'))
</script>

<template>
  <section
    ref="root"
    class="weft-cta"
    :data-variant="variant"
    :data-tone="tone"
    :data-orientation="orientation"
    :data-align="resolvedAlign"
    :data-width="width"
    :data-weft-w="widthId"
  >
    <div class="weft-cta__box">
      <div class="weft-cta__copy">
        <div v-if="headline || $slots.headline" class="weft-cta__headline"><slot name="headline">{{ headline }}</slot></div>
        <component :is="`h${level}`" v-if="title || $slots.title" class="weft-cta__title"><slot name="title">{{ title }}</slot></component>
        <p v-if="description || $slots.description" class="weft-cta__description"><slot name="description">{{ description }}</slot></p>
        <div v-if="$slots.default" class="weft-cta__body"><slot /></div>
        <div v-if="($slots.actions || actions?.length) && (orientation === 'vertical' || !!$slots.aside)" class="weft-cta__actions">
          <slot name="actions">
            <Button
              v-for="(action, i) in actions"
              :key="i"
              size="lg"
              :label="action.label"
              :icon="action.icon"
              :to="action.to"
              :disabled="action.disabled"
              :loading="action.loading"
              :tone="action.tone ?? (variant === 'solid' ? 'neutral' : i === 0 ? tone : 'neutral')"
              :variant="action.variant ?? (i === 0 ? 'solid' : 'outline')"
              @click="action.onClick?.($event)"
            />
          </slot>
        </div>
      </div>
      <div v-if="$slots.aside" class="weft-cta__aside"><slot name="aside" /></div>
      <div v-else-if="orientation === 'horizontal' && ($slots.actions || actions?.length)" class="weft-cta__actions weft-cta__actions--end">
        <slot name="actions">
          <Button
            v-for="(action, i) in actions"
            :key="i"
            size="lg"
            :label="action.label"
            :icon="action.icon"
            :to="action.to"
            :disabled="action.disabled"
            :loading="action.loading"
            :tone="action.tone ?? (variant === 'solid' ? 'neutral' : i === 0 ? tone : 'neutral')"
            :variant="action.variant ?? (i === 0 ? 'solid' : 'outline')"
            @click="action.onClick?.($event)"
          />
        </slot>
      </div>
    </div>
  </section>
</template>
