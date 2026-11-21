<script setup vapor lang="ts">
// The bar at the top of a page: the site's name (a link home) or a brand at
// the start, navigation in the middle, actions at the end, in a <header>
// (the banner landmark at the top of a page). When the viewport is narrow
// and a `menu` slot is given, the middle gives way to a menu button that
// opens the menu in a Sheet from the start side; `close()` in the slot shuts
// it (after a link is chosen). Without a `menu` slot the middle stays, and a
// NavigationMenu there takes its own narrow form. The menu button is at
// least --weft-hit-min square on touch (Button's rule).
import { computed, onMounted, shallowRef } from 'vue'
import Button from '../basics/Button.vue'
import Link from '../basics/Link.vue'
import { useWidth } from '../adaptive/useWidth.ts'
import { useControllable } from '../headless/useControllable.ts'
import Sheet from '../overlays/Sheet.vue'
import '../styles/base.css'
import '../styles/header.css'

const props = withDefaults(
  defineProps<{
    /** The site's or app's name, shown at the start as a link to `to`. The leading slot replaces it. */
    title?: string
    /** Where the title links. */
    to?: string
    /** The menu button's name, and the sheet's title. */
    menuLabel?: string
    /** Stick to the top of the viewport while the page scrolls. */
    sticky?: boolean
    /** The narrow menu open or shut, bound with `v-model:open`. */
    open?: boolean
  }>(),
  { to: '/', menuLabel: 'Menu', sticky: true, open: undefined },
)

const emit = defineEmits<{
  /** The narrow menu opened or shut. */
  'update:open': [open: boolean]
}>()

const slots = defineSlots<{
  /** The start: a logo or brand, in place of the title. */
  leading?: () => unknown
  /** The middle: the main navigation. */
  default?: () => unknown
  /** The end: search, account, theme switch. */
  actions?: () => unknown
  /** What the narrow menu sheet holds; `close()` shuts it. Its presence turns the middle into a menu button when narrow. */
  menu?: (props: { close: () => void }) => unknown
}>()

// Narrow is the viewport's width: the root element's.
const viewport = shallowRef<HTMLElement | null>(null)
onMounted(() => (viewport.value = document.documentElement))
const width = useWidth(viewport)
const compact = computed(() => width.value === 'narrow' && !!slots.menu)
const open = useControllable(props, 'open', emit, false)
</script>

<template>
  <header class="weft-header" :data-sticky="sticky ? 'true' : undefined" :data-compact="compact ? 'true' : undefined">
    <div class="weft-header__inner">
      <span v-if="compact" class="weft-header__menu">
        <Sheet v-model:open="open" side="start" :title="menuLabel">
          <Button variant="ghost" tone="neutral" square :aria-label="menuLabel">
            <template #leading><svg class="weft-header__glyph" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M4 6h16" /><path d="M4 12h16" /><path d="M4 18h16" /></svg></template>
          </Button>
          <template #body="{ close }"><slot name="menu" :close="close" /></template>
        </Sheet>
      </span>
      <div class="weft-header__leading">
        <slot name="leading"><Link v-if="title" class="weft-header__title" standalone :to="to">{{ title }}</Link></slot>
      </div>
      <div v-if="!compact && $slots.default" class="weft-header__center"><slot /></div>
      <div v-if="$slots.actions" class="weft-header__actions"><slot name="actions" /></div>
    </div>
  </header>
</template>
