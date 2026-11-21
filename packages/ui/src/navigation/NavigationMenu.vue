<script setup vapor lang="ts">
// A site's or an app's main links: a navigation landmark of lists. Every
// link is a Weft Link, so the current page says aria-current="page" (its
// `active`, else its address against the browser's, after mounting).
//
// - Horizontal, on pointer and on wide touch screens: a row of links, and of
//   buttons that open their children as a DropdownMenu (a bottom sheet on
//   touch, DropdownMenu's own rule). Every item is in the tab order; Left
//   and Right also move between them (mirrored right to left), Home and End
//   to the ends.
// - Horizontal on a narrow touch screen: one menu button that opens the same
//   items as a vertical list in a Sheet from the start side, which closes
//   when an item is chosen.
// - Vertical (a sidebar): a column of links; an item with children is a
//   disclosure button (aria-expanded, aria-controls) over a list that opens
//   below it, rendered and hidden until found, so find-in-page opens it. Up
//   and Down move over the visible items. `collapsed` draws icons alone,
//   each label in a Tooltip (on pointer) and still read by assistive
//   technology; children open as a DropdownMenu beside it.
// On touch every item is at least --weft-hit-min tall.
import { computed, onMounted, ref, shallowRef, useTemplateRef } from 'vue'
import Badge from '../basics/Badge.vue'
import Button from '../basics/Button.vue'
import Icon from '../basics/Icon.vue'
import Link from '../basics/Link.vue'
import { useWeft } from '../adaptive/context.ts'
import { useWidth } from '../adaptive/useWidth.ts'
import { useId } from '../headless/useId.ts'
import { useRovingFocus } from '../headless/useRovingFocus.ts'
import DropdownMenu from '../overlays/DropdownMenu.vue'
import Sheet from '../overlays/Sheet.vue'
import Tooltip from '../overlays/Tooltip.vue'
import { holdsCurrent, menuOf, navGroups, type NavItem } from './nav.ts'
import NavigationMenu from './NavigationMenu.vue'
import '../styles/base.css'
import '../styles/navigation-menu.css'

const props = withDefaults(
  defineProps<{
    /** The items, or groups of them (an array of arrays), drawn apart: `{ label, to, icon, active, badge, children, … }`. */
    items?: NavItem[] | NavItem[][]
    /** `horizontal`: a bar of links with dropdowns. `vertical`: a column, as in a sidebar. */
    orientation?: 'horizontal' | 'vertical'
    /** Vertical only: icons alone, for a collapsed sidebar; the labels stay for assistive technology and show in tooltips. */
    collapsed?: boolean
    /** The landmark's name, for assistive technology (and the sheet's title on a narrow touch screen). */
    label?: string
    /** The name of the menu button on a narrow touch screen. */
    menuLabel?: string
  }>(),
  { items: () => [], orientation: 'horizontal', label: 'Main', menuLabel: 'Menu' },
)

const emit = defineEmits<{
  /** An item was chosen: a link followed or a button pressed. */
  select: [item: NavItem, event: Event]
}>()

defineSlots<{
  /** An item's label, in place of `label`. */
  item?: (props: { item: NavItem }) => unknown
}>()

const weft = useWeft()
const id = useId('nav')
const root = useTemplateRef<HTMLElement>('root')
// Narrow is the viewport's width: the root element's.
const viewport = shallowRef<HTMLElement | null>(null)
onMounted(() => (viewport.value = document.documentElement))
const width = useWidth(viewport)
/** What it draws: a bar, a column, or (horizontal on a narrow touch screen) a sheet. */
const form = computed(() =>
  props.orientation === 'vertical' ? 'vertical' : weft.input.value === 'touch' && width.value === 'narrow' ? 'sheet' : 'horizontal',
)
const collapsed = computed(() => form.value === 'vertical' && props.collapsed)
const groups = computed(() => navGroups(props.items))

useRovingFocus(root, {
  get orientation() {
    return form.value === 'vertical' ? 'vertical' : 'horizontal'
  },
  tabStops: 'all',
})

const sheetOpen = ref(false)
function chosen(item: NavItem, event: Event) {
  emit('select', item, event)
}
function onItem(item: NavItem, event: MouseEvent) {
  if (item.disabled) return
  item.onSelect?.(event)
  chosen(item, event)
}
function onSheetSelect(item: NavItem, event: Event) {
  sheetOpen.value = false
  chosen(item, event)
}

// Vertical: which items with children are open, by place.
const opened = ref<Set<string>>(new Set())
const initial = new Set<string>()
groups.value.forEach((group, g) => group.forEach((item, i) => (item.defaultOpen || item.children?.some(holdsCurrent)) && initial.add(`${g}-${i}`)))
opened.value = initial
const isOpen = (key: string) => opened.value.has(key)
function setOpen(key: string, on: boolean) {
  const next = new Set(opened.value)
  if (on) next.add(key)
  else next.delete(key)
  opened.value = next
}
const menus = (item: NavItem) => menuOf(item, chosen)
</script>

<template>
  <nav ref="root" class="weft-nav" :aria-label="label" :data-orientation="form" :data-collapsed="collapsed ? 'true' : undefined">
    <Sheet v-if="form === 'sheet'" v-model:open="sheetOpen" side="start" :title="label">
      <Button class="weft-nav__menu" variant="ghost" tone="neutral" square :aria-label="menuLabel">
        <template #leading><svg class="weft-nav__glyph" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M4 6h16" /><path d="M4 12h16" /><path d="M4 18h16" /></svg></template>
      </Button>
      <template #body>
        <NavigationMenu :items="items" orientation="vertical" :label="label" @select="onSheetSelect">
          <template v-if="$slots.item" #item="{ item }"><slot name="item" :item="item" /></template>
        </NavigationMenu>
      </template>
    </Sheet>
    <template v-else>
      <ul v-for="(group, g) in groups" :key="g" class="weft-nav__list">
        <li v-for="(item, i) in group" :key="i" class="weft-nav__entry" :data-type="item.type">
          <span v-if="item.type === 'label'" class="weft-nav__heading" :class="{ 'weft-visually-hidden': collapsed }">{{ item.label }}</span>
          <DropdownMenu
            v-else-if="item.children?.length && form !== 'vertical'"
            :items="menus(item)"
            placement="bottom-start"
          >
            <button type="button" class="weft-nav__item weft-nav__trigger" data-weft-item :disabled="item.disabled || undefined" :data-active="holdsCurrent(item) ? 'true' : undefined">
              <Icon v-if="item.icon" :name="item.icon" class="weft-nav__icon" />
              <span class="weft-nav__label"><slot name="item" :item="item">{{ item.label }}</slot></span>
              <svg class="weft-nav__glyph weft-nav__chevron" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="m6 9 6 6 6-6" /></svg>
            </button>
          </DropdownMenu>
          <DropdownMenu v-else-if="item.children?.length && collapsed" :items="menus(item)" placement="right-start">
            <button type="button" class="weft-nav__item weft-nav__trigger" data-weft-item :disabled="item.disabled || undefined" :data-active="holdsCurrent(item) ? 'true' : undefined">
              <Icon v-if="item.icon" :name="item.icon" class="weft-nav__icon" />
              <span class="weft-nav__label weft-visually-hidden"><slot name="item" :item="item">{{ item.label }}</slot></span>
            </button>
          </DropdownMenu>
          <template v-else-if="item.children?.length">
            <button
              type="button"
              class="weft-nav__item weft-nav__toggle"
              data-weft-item
              :aria-expanded="isOpen(`${g}-${i}`) ? 'true' : 'false'"
              :aria-controls="`${id}-${g}-${i}`"
              :disabled="item.disabled || undefined"
              :data-active="holdsCurrent(item) ? 'true' : undefined"
              @click="setOpen(`${g}-${i}`, !isOpen(`${g}-${i}`))"
            >
              <Icon v-if="item.icon" :name="item.icon" class="weft-nav__icon" />
              <span class="weft-nav__label"><slot name="item" :item="item">{{ item.label }}</slot></span>
              <Badge v-if="item.badge !== undefined" :label="item.badge" variant="soft" tone="neutral" size="sm" />
              <svg class="weft-nav__glyph weft-nav__chevron" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="m6 9 6 6 6-6" /></svg>
            </button>
            <ul
              :id="`${id}-${g}-${i}`"
              class="weft-nav__children"
              :hidden.attr="isOpen(`${g}-${i}`) ? undefined : 'until-found'"
              @beforematch="setOpen(`${g}-${i}`, true)"
            >
              <li v-for="(child, c) in item.children" :key="c" class="weft-nav__entry">
                <Link
                  v-if="child.to"
                  class="weft-nav__item weft-nav__child"
                  :data-weft-item="isOpen(`${g}-${i}`) ? '' : undefined"
                  :data-disabled="child.disabled ? 'true' : undefined"
                  :to="child.to"
                  :target="child.target"
                  :active="child.active"
                  :disabled="child.disabled"
                  @click="onItem(child, $event)"
                >
                  <Icon v-if="child.icon" :name="child.icon" class="weft-nav__icon" />
                  <span class="weft-nav__label"><slot name="item" :item="child">{{ child.label }}</slot></span>
                  <Badge v-if="child.badge !== undefined" :label="child.badge" variant="soft" tone="neutral" size="sm" />
                </Link>
                <button
                  v-else
                  type="button"
                  class="weft-nav__item weft-nav__child"
                  :data-weft-item="isOpen(`${g}-${i}`) ? '' : undefined"
                  :disabled="child.disabled || undefined"
                  @click="onItem(child, $event)"
                >
                  <Icon v-if="child.icon" :name="child.icon" class="weft-nav__icon" />
                  <span class="weft-nav__label"><slot name="item" :item="child">{{ child.label }}</slot></span>
                </button>
              </li>
            </ul>
          </template>
          <Tooltip v-else-if="item.to" :text="item.label" placement="right" :disabled="!collapsed">
            <Link
              class="weft-nav__item"
              data-weft-item
              :data-disabled="item.disabled ? 'true' : undefined"
              :to="item.to"
              :target="item.target"
              :active="item.active"
              :disabled="item.disabled"
              @click="onItem(item, $event)"
            >
              <Icon v-if="item.icon" :name="item.icon" class="weft-nav__icon" />
              <span class="weft-nav__label" :class="{ 'weft-visually-hidden': collapsed }"><slot name="item" :item="item">{{ item.label }}</slot></span>
              <Badge v-if="item.badge !== undefined && !collapsed" :label="item.badge" variant="soft" tone="neutral" size="sm" />
            </Link>
          </Tooltip>
          <Tooltip v-else :text="item.label" placement="right" :disabled="!collapsed">
            <button type="button" class="weft-nav__item" data-weft-item :disabled="item.disabled || undefined" @click="onItem(item, $event)">
              <Icon v-if="item.icon" :name="item.icon" class="weft-nav__icon" />
              <span class="weft-nav__label" :class="{ 'weft-visually-hidden': collapsed }"><slot name="item" :item="item">{{ item.label }}</slot></span>
              <Badge v-if="item.badge !== undefined && !collapsed" :label="item.badge" variant="soft" tone="neutral" size="sm" />
            </button>
          </Tooltip>
        </li>
      </ul>
    </template>
  </nav>
</template>
