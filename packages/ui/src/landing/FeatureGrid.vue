<script setup vapor lang="ts">
// A titled section of capabilities or benefits in a multi-column Grid: an
// optional eyebrow headline, section heading (h2 by default, or `h${level}`),
// description and actions, then items with an icon in a tinted disc, a title
// one heading level deeper, a description and an optional link. `card`
// (default) boxes each item on a surface card; `plain` leaves them unboxed.
// Grid adapts from its own container width in CSS first, reflowing from 1
// column (narrow) to 2 (regular) to 3 (wide) unless `cols` says otherwise.
import { computed } from 'vue'
import Icon from '../basics/Icon.vue'
import Link from '../basics/Link.vue'
import Grid from '../layout/Grid.vue'
import type { Tone } from '../types.ts'
import type { FeatureItem } from './types.ts'
import '../styles/base.css'
import '../styles/tone.css'
import '../styles/feature-grid.css'

type Cols = number | { narrow?: number; regular?: number; wide?: number }

const props = withDefaults(
  defineProps<{
    /** Short eyebrow text above the section title; the headline slot replaces it. */
    headline?: string
    /** The section's heading, rendered as an h2 by default (or `h${level}`). */
    title?: string
    /** A lead sentence under the title. */
    description?: string
    /** The features to show: `{ title, description, icon, to, linkLabel, slot }`. */
    items?: FeatureItem[]
    /** Columns: one number for every width, or `{ narrow, regular, wide }` of the grid's own width. */
    cols?: Cols
    /** `card` (each item in a bordered box, default) or `plain` (unboxed items). */
    variant?: 'card' | 'plain'
    /** The colour of the icon discs. */
    tone?: Tone
    /** Alignment of the section header: `start` (default) or `center`. */
    align?: 'start' | 'center'
    /** The section's heading level, 2 to 6; item titles render one level deeper. */
    level?: 2 | 3 | 4 | 5 | 6
  }>(),
  {
    items: () => [],
    cols: () => ({ narrow: 1, regular: 2, wide: 3 }),
    variant: 'card',
    tone: 'primary',
    align: 'start',
    level: 2,
  },
)

defineSlots<{
  /** Above the section title, in place of `headline`. */
  headline?: () => unknown
  /** The section's heading block, in place of `title` and `description`. */
  header?: () => unknown
  /** Actions at the end of the section header. */
  actions?: () => unknown
  /** Custom content for an item, in place of its default icon, title and description. */
  item?: (props: { item: FeatureItem; index: number }) => unknown
  /** Additional items or cards inside the grid. */
  default?: () => unknown
  /** Custom body for the item whose `slot` names it. */
  [name: string]: ((props: { item: FeatureItem; index: number }) => unknown) | undefined
}>()

const itemHeading = computed(() => `h${Math.min(props.level + 1, 6)}`)
</script>

<template>
  <section class="weft-feature-grid" :data-variant="variant" :data-tone="tone" :data-align="align">
    <div class="weft-feature-grid__inner">
      <header v-if="headline || title || description || $slots.headline || $slots.header || $slots.actions" class="weft-feature-grid__header">
        <div class="weft-feature-grid__heading">
          <slot name="header">
            <div v-if="headline || $slots.headline" class="weft-feature-grid__headline"><slot name="headline">{{ headline }}</slot></div>
            <component :is="`h${level}`" v-if="title" class="weft-feature-grid__title">{{ title }}</component>
            <p v-if="description" class="weft-feature-grid__description">{{ description }}</p>
          </slot>
        </div>
        <div v-if="$slots.actions" class="weft-feature-grid__actions"><slot name="actions" /></div>
      </header>
      <Grid class="weft-feature-grid__grid" :cols="cols" :gap="4">
        <div
          v-for="(item, i) in items"
          :key="item.title || i"
          class="weft-feature-grid__item"
        >
          <slot v-if="item.slot && $slots[item.slot]" :name="item.slot" :item="item" :index="i" />
          <slot v-else name="item" :item="item" :index="i">
            <span v-if="item.icon" class="weft-feature-grid__icon" aria-hidden="true">
              <Icon :name="item.icon" />
            </span>
            <component :is="itemHeading" class="weft-feature-grid__item-title">
              <Link v-if="item.to && !item.linkLabel" :to="item.to" standalone>{{ item.title }}</Link>
              <template v-else>{{ item.title }}</template>
            </component>
            <p v-if="item.description" class="weft-feature-grid__item-description">{{ item.description }}</p>
            <Link v-if="item.to && item.linkLabel" class="weft-feature-grid__item-link" :to="item.to" standalone>{{ item.linkLabel }}</Link>
          </slot>
        </div>
        <slot />
      </Grid>
    </div>
  </section>
</template>
