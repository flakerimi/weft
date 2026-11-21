<script setup vapor lang="ts">
// Where the page sits in the site: a navigation landmark holding an ordered
// list of links from the top down (WAI-ARIA APG, Breadcrumb). The last item
// is the current page, aria-current="page", whether it is a link or text; an
// item without `to` is text. Separators are drawn, hidden from assistive
// technology, and mirror right to left. Links are Weft Links, standalone, so
// each has a --weft-hit-min hit area on touch. A path too long for its width
// scrolls sideways.
import { computed } from 'vue'
import Icon from '../basics/Icon.vue'
import Link from '../basics/Link.vue'
import type { BreadcrumbItem } from './breadcrumb.ts'
import '../styles/base.css'
import '../styles/breadcrumb.css'

const props = withDefaults(
  defineProps<{
    /** The path, from the top: `{ label, to, icon, target, disabled }`; the last is the current page. */
    items?: BreadcrumbItem[]
    /** A Lucide icon between items, in place of the chevron. */
    separatorIcon?: string
    /** The landmark's name, for assistive technology. */
    label?: string
  }>(),
  { items: () => [], label: 'Breadcrumb' },
)

defineSlots<{
  /** An item's content, in place of its icon and label. */
  item?: (props: { item: BreadcrumbItem; index: number; current: boolean }) => unknown
  /** The separator, in place of the chevron. */
  separator?: () => unknown
}>()

const last = computed(() => props.items.length - 1)
</script>

<template>
  <nav class="weft-breadcrumb" :aria-label="label">
    <ol class="weft-breadcrumb__list">
      <li v-for="(item, i) in items" :key="i" class="weft-breadcrumb__item">
        <Link
          v-if="item.to"
          class="weft-breadcrumb__link"
          standalone
          :to="item.to"
          :target="item.target"
          :disabled="item.disabled"
          :active="i === last ? true : item.active"
        >
          <slot name="item" :item="item" :index="i" :current="i === last"><Icon v-if="item.icon" :name="item.icon" class="weft-breadcrumb__icon" />{{ item.label }}</slot>
        </Link>
        <span v-else class="weft-breadcrumb__text" :aria-current="i === last ? 'page' : undefined">
          <slot name="item" :item="item" :index="i" :current="i === last"><Icon v-if="item.icon" :name="item.icon" class="weft-breadcrumb__icon" />{{ item.label }}</slot>
        </span>
        <span v-if="i < last" class="weft-breadcrumb__separator" aria-hidden="true">
          <slot name="separator">
            <Icon v-if="separatorIcon" :name="separatorIcon" />
            <svg v-else viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m9 18 6-6-6-6" /></svg>
          </slot>
        </span>
      </li>
    </ol>
  </nav>
</template>
