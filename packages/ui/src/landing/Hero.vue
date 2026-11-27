<script setup vapor lang="ts">
// The opening band of a landing page: an optional eyebrow headline, the page's
// h1 (an h2 inside a WeftProvider marked `inset`, or `h${level}` when `level`
// is set), a lead paragraph, primary and secondary action buttons, a quiet
// caption, and an optional visual (an app preview, picture or code block)
// below the copy (`vertical`, the default) or beside it (`horizontal` when its
// own container is wide, stacking above when narrow or regular, in CSS first
// through @container). On touch its action buttons are at least --weft-hit-min
// tall (Button's rule).
import { computed, useTemplateRef } from 'vue'
import { useInset } from '../adaptive/context.ts'
import { useWidth } from '../adaptive/useWidth.ts'
import Button from '../basics/Button.vue'
import type { ActionItem } from '../feedback/actions.ts'
import '../styles/base.css'
import '../styles/hero.css'

const props = withDefaults(
  defineProps<{
    /** Short eyebrow text above the title; the headline slot replaces it. */
    headline?: string
    /** The main heading: an h1 by default (h2 inside an inset WeftProvider, or `h${level}`). */
    title?: string
    /** A lead paragraph under the title. */
    description?: string
    /** Call-to-action buttons: `{ label, onClick, to, icon, tone, variant }`; the first is solid primary, the rest outlined. */
    actions?: ActionItem[]
    /** `vertical` (default: copy above, visual below) or `horizontal` (copy beside the visual when wide, stacked when narrow or regular). */
    orientation?: 'vertical' | 'horizontal'
    /** Alignment of the copy and actions: `center` (default when vertical) or `start` (default when horizontal). */
    align?: 'start' | 'center'
    /** Heading level, 1 to 6; unset uses h1 (or h2 inside an inset WeftProvider). */
    level?: 1 | 2 | 3 | 4 | 5 | 6
  }>(),
  { orientation: 'vertical', align: undefined, level: undefined },
)

defineSlots<{
  /** A decorative background layer behind the container. */
  background?: () => unknown
  /** Above the title, in place of `headline` (e.g. a Badge). */
  headline?: () => unknown
  /** Inside the heading, in place of `title`. */
  title?: () => unknown
  /** Under the title, in place of `description`. */
  description?: () => unknown
  /** The call-to-action controls, in place of `actions`. */
  actions?: () => unknown
  /** A quiet note under the actions ("Already installed? weft update"). */
  caption?: () => unknown
  /** The visual (an app frame, picture or code panel) below or beside the copy. */
  default?: () => unknown
}>()

const inset = useInset()
const root = useTemplateRef<HTMLElement>('root')
const width = useWidth(root)
const widthId = width.id
const headingTag = computed(() => `h${props.level ?? (inset ? 2 : 1)}`)
const resolvedAlign = computed(() => props.align ?? (props.orientation === 'horizontal' ? 'start' : 'center'))
</script>

<template>
  <section
    ref="root"
    class="weft-hero"
    :data-orientation="orientation"
    :data-align="resolvedAlign"
    :data-width="width"
    :data-weft-w="widthId"
  >
    <div v-if="$slots.background" class="weft-hero__background" aria-hidden="true"><slot name="background" /></div>
    <div class="weft-hero__inner">
      <div class="weft-hero__copy">
        <div v-if="headline || $slots.headline" class="weft-hero__headline"><slot name="headline">{{ headline }}</slot></div>
        <component :is="headingTag" v-if="title || $slots.title" class="weft-hero__title"><slot name="title">{{ title }}</slot></component>
        <p v-if="description || $slots.description" class="weft-hero__description"><slot name="description">{{ description }}</slot></p>
        <div v-if="$slots.actions || actions?.length" class="weft-hero__actions">
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
              :tone="action.tone ?? (i === 0 ? 'primary' : 'neutral')"
              :variant="action.variant ?? (i === 0 ? 'solid' : 'outline')"
              @click="action.onClick?.($event)"
            />
          </slot>
        </div>
        <div v-if="$slots.caption" class="weft-hero__caption"><slot name="caption" /></div>
      </div>
      <div v-if="$slots.default" class="weft-hero__media"><slot /></div>
    </div>
  </section>
</template>
