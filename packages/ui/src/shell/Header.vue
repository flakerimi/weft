<script setup vapor lang="ts">
// The bar at the top of a page: the site's name (a link home) or a brand at
// the start, navigation in the middle, actions at the end, in a <header>
// (the banner landmark at the top of a page). When the viewport is narrow
// and a `menu` slot is given, the middle gives way to a menu button that
// opens the menu in a Sheet from the start side; `close()` in the slot shuts
// it (after a link is chosen). Without a `menu` slot the middle stays, and a
// NavigationMenu there takes its own narrow form. Before mounting both
// render and CSS picks from the viewport; once mounted (`data-layout`
// compact or full) only the one shown does. When the layout changes the
// sheet closes, and focus in the part that goes moves to the menu button,
// or to the first link of the middle. The menu button is at least
// --weft-hit-min square on touch (Button's rule).
import { computed, nextTick, onMounted, shallowRef, useTemplateRef, watch } from 'vue'
import Button from '../basics/Button.vue'
import Link from '../basics/Link.vue'
import { useWidth } from '../adaptive/useWidth.ts'
import { focus, focused, tabbables } from '../headless/dom.ts'
import { useControllable } from '../headless/useControllable.ts'
import { useId } from '../headless/useId.ts'
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
const mounted = shallowRef(false)
onMounted(() => {
  viewport.value = document.documentElement
  mounted.value = true
})
const width = useWidth(viewport, { prepaint: false })
const compact = computed(() => width.value === 'narrow' && !!slots.menu)
const open = useControllable(props, 'open', emit, false)
const id = useId('header')
const root = useTemplateRef<HTMLElement>('root')

// A change of layout closes the sheet, and moves focus out of the part that goes.
watch(compact, async (now) => {
  if (!mounted.value) return
  const el = focused()
  const going = now ? '.weft-header__center' : '.weft-header__menu'
  const moving = !!el && (!!el.closest(`[data-weft-header-sheet="${id}"]`) || (!!root.value?.contains(el) && !!el.closest(going)))
  open.value = false
  if (!moving) return
  await nextTick()
  const el2 = root.value
  if (!el2) return
  const center = el2.querySelector('.weft-header__center')
  focus(now ? el2.querySelector<HTMLElement>('.weft-header__menu button') : ((center && tabbables(center)[0]) ?? el2.querySelector<HTMLElement>('.weft-header__title')))
})
</script>

<template>
  <header
    ref="root"
    class="weft-header"
    :data-sticky="sticky ? 'true' : undefined"
    :data-menu="$slots.menu ? 'true' : undefined"
    :data-layout="mounted ? (compact ? 'compact' : 'full') : undefined"
  >
    <div class="weft-header__inner">
      <span v-if="$slots.menu && (!mounted || compact)" class="weft-header__menu">
        <Sheet v-model:open="open" side="start" :title="menuLabel" :data-weft-header-sheet="id">
          <Button variant="ghost" tone="neutral" square :aria-label="menuLabel">
            <template #leading><svg class="weft-header__glyph" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M4 6h16" /><path d="M4 12h16" /><path d="M4 18h16" /></svg></template>
          </Button>
          <template #body="{ close }"><slot name="menu" :close="close" /></template>
        </Sheet>
      </span>
      <div class="weft-header__leading">
        <slot name="leading"><Link v-if="title" class="weft-header__title" standalone :to="to">{{ title }}</Link></slot>
      </div>
      <div v-if="$slots.default && (!mounted || !compact)" class="weft-header__center"><slot /></div>
      <div v-if="$slots.actions" class="weft-header__actions"><slot name="actions" /></div>
    </div>
  </header>
</template>
