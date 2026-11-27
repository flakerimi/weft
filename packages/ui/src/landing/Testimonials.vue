<script setup vapor lang="ts">
// Customer or reader quotes in a responsive Grid: each quote is a semantic
// <figure> with a <blockquote> and a <figcaption> naming the author, their
// role or company, and their Avatar (which falls back to initials from
// `author` and is decorative beside the visible author name). An item with
// `to` links its author through Link. Grid adapts from its own container width
// in CSS first, reflowing from 1 column (narrow) to 2 (regular) to 3 (wide).
import Avatar from '../basics/Avatar.vue'
import Link from '../basics/Link.vue'
import Grid from '../layout/Grid.vue'
import type { TestimonialItem } from './types.ts'
import '../styles/base.css'
import '../styles/testimonials.css'

type Cols = number | { narrow?: number; regular?: number; wide?: number }

withDefaults(
  defineProps<{
    /** Short eyebrow text above the section title; the headline slot replaces it. */
    headline?: string
    /** The section's heading, rendered as an h2 by default (or `h${level}`). */
    title?: string
    /** A lead sentence under the title. */
    description?: string
    /** The quotes to show: `{ quote, author, role, avatar, to }`. */
    items?: TestimonialItem[]
    /** Columns: one number for every width, or `{ narrow, regular, wide }` of the grid's own width. */
    cols?: Cols
    /** Alignment of the section header: `start` (default) or `center`. */
    align?: 'start' | 'center'
    /** The section's heading level, 2 to 6. */
    level?: 2 | 3 | 4 | 5 | 6
  }>(),
  {
    items: () => [],
    cols: () => ({ narrow: 1, regular: 2, wide: 3 }),
    align: 'start',
    level: 2,
  },
)

defineSlots<{
  /** Above the section title, in place of `headline`. */
  headline?: () => unknown
  /** The section's heading block, in place of `title` and `description`. */
  header?: () => unknown
  /** Custom content for a testimonial card, in place of its quote and author caption. */
  item?: (props: { item: TestimonialItem; index: number }) => unknown
  /** Additional testimonial figures inside the grid. */
  default?: () => unknown
}>()

function initialsOf(author: string): string {
  const words = author.trim().split(/\s+/).filter(Boolean)
  const first = (w: string | undefined) => (w ? Array.from(w)[0]!.toUpperCase() : '')
  return words.length > 1 ? first(words[0]) + first(words[words.length - 1]) : first(words[0])
}
</script>

<template>
  <section class="weft-testimonials" :data-align="align">
    <div class="weft-testimonials__inner">
      <header v-if="headline || title || description || $slots.headline || $slots.header" class="weft-testimonials__header">
        <slot name="header">
          <div v-if="headline || $slots.headline" class="weft-testimonials__headline"><slot name="headline">{{ headline }}</slot></div>
          <component :is="`h${level}`" v-if="title" class="weft-testimonials__title">{{ title }}</component>
          <p v-if="description" class="weft-testimonials__description">{{ description }}</p>
        </slot>
      </header>
      <Grid class="weft-testimonials__grid" :cols="cols" :gap="4">
        <figure
          v-for="(item, i) in items"
          :key="`${item.author}-${i}`"
          class="weft-testimonials__item"
        >
          <slot name="item" :item="item" :index="i">
            <blockquote class="weft-testimonials__quote">
              <p class="weft-testimonials__quote-text">{{ item.quote }}</p>
            </blockquote>
            <figcaption class="weft-testimonials__author">
              <Avatar class="weft-testimonials__avatar" :src="item.avatar" :text="initialsOf(item.author)" size="md" />
              <div class="weft-testimonials__meta">
                <div class="weft-testimonials__name">
                  <Link v-if="item.to" :to="item.to" standalone>{{ item.author }}</Link>
                  <template v-else>{{ item.author }}</template>
                </div>
                <div v-if="item.role" class="weft-testimonials__role">{{ item.role }}</div>
              </div>
            </figcaption>
          </slot>
        </figure>
        <slot />
      </Grid>
    </div>
  </section>
</template>
