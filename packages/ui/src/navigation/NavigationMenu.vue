<script setup vapor lang="ts">
// A site's or an app's main links: a navigation landmark of lists, in the
// WAI-ARIA APG Disclosure Navigation pattern. Links stay links; an item with
// children is a disclosure button (aria-expanded, aria-controls) that opens
// a list of links. Every link is a Weft Link, so the current page says
// aria-current="page" (its `active`, else its address against the
// browser's, after mounting).
//
// Forms (`data-form`, set once mounted):
// - `bar` (horizontal, on pointer and on wide touch screens): a row of links
//   and disclosure buttons, scrolling sideways when it overflows; an open
//   list floats under its button. Every item is in the tab order; Left and
//   Right also move between them (mirrored right to left), Home and End to
//   the ends. On a button, ArrowDown opens the list and enters it; in the
//   list the arrows, Home and End move, Escape closes it back to the button.
//   A list closes when focus leaves it, on a press outside, and when a link
//   in it is chosen. One list is open at a time.
// - `sheet` (horizontal on a narrow touch screen): one menu button opening
//   the items as a column in a Sheet from the start side, which closes when
//   an item is chosen.
// - `column` (vertical, a sidebar): a list's button opens it below, rendered
//   and hidden until found, so find-in-page opens it; a group starts open
//   with `defaultOpen` or when it holds the current page, and follows new
//   items. Up and Down move over the visible items. `collapsed` draws icons
//   alone (labels kept for assistive technology, in a Tooltip on pointer),
//   and a list floats beside its button, opened with the forward arrow.
// Before mounting there is no `data-form`: CSS shows the bar, or the menu
// button on a narrow touch screen (`max-width` and `pointer: coarse`), so
// the first paint matches. When the form changes (the input or the width
// changes), an open sheet or list closes, and focus, if it was in the part
// that goes, moves to the same item in the new form, or its menu button.
// On touch every item is at least --weft-hit-min tall.
import { computed, nextTick, onMounted, ref, shallowRef, useTemplateRef, watch, watchEffect } from 'vue'
import Badge from '../basics/Badge.vue'
import Button from '../basics/Button.vue'
import Icon from '../basics/Icon.vue'
import Link from '../basics/Link.vue'
import { useWeft } from '../adaptive/context.ts'
import { useViewportWidth } from '../adaptive/useWidth.ts'
import { focus, focused, isRtl } from '../headless/dom.ts'
import { useDismiss } from '../headless/useDismiss.ts'
import { useFloating } from '../headless/useFloating.ts'
import { useId } from '../headless/useId.ts'
import { useRovingFocus } from '../headless/useRovingFocus.ts'
import Sheet from '../overlays/Sheet.vue'
import Tooltip from '../overlays/Tooltip.vue'
import { holdsCurrent, navGroups, type NavItem } from './nav.ts'
import NavigationMenu from './NavigationMenu.vue'
import '../styles/base.css'
import '../styles/navigation-menu.css'

const props = withDefaults(
  defineProps<{
    /** The items, or groups of them (an array of arrays), drawn apart: `{ label, to, icon, active, badge, children, … }`. */
    items?: NavItem[] | NavItem[][]
    /** `horizontal`: a bar of links and disclosure buttons. `vertical`: a column, as in a sidebar. */
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
// The viewport's width class, as the `@media` fallback before mounting sees it.
const width = useViewportWidth()
const mounted = ref(false)
onMounted(() => {
  mounted.value = true
})
const horizontal = computed(() => props.orientation === 'horizontal')
const form = computed<'bar' | 'sheet' | 'column'>(() =>
  !horizontal.value ? 'column' : weft.input.value === 'touch' && width.value === 'narrow' ? 'sheet' : 'bar',
)
/** Before mounting both horizontal forms render, and CSS shows one. */
const showBar = computed(() => !horizontal.value || !mounted.value || form.value === 'bar')
const showSheet = computed(() => horizontal.value && (!mounted.value || form.value === 'sheet'))
const collapsed = computed(() => !horizontal.value && props.collapsed)
/** Lists that float: from the bar, and from a collapsed column. */
const floats = computed(() => horizontal.value || collapsed.value)
const groups = computed(() => navGroups(props.items))

useRovingFocus(root, {
  get orientation() {
    return horizontal.value ? 'horizontal' : 'vertical'
  },
  tabStops: 'all',
})

const keyOf = (g: number, i: number, item: NavItem) => `${g}-${i}-${item.label ?? ''}`
const listId = (g: number, i: number) => `${id}-${g}-${i}`

// ── Choosing ──
const sheetOpen = ref(false)
function chosen(item: NavItem, event: Event) {
  emit('select', item, event)
}
function onItem(item: NavItem, event: MouseEvent) {
  if (item.disabled) return
  item.onSelect?.(event)
  openKey.value = null
  chosen(item, event)
}
function onSheetSelect(item: NavItem, event: Event) {
  sheetOpen.value = false
  chosen(item, event)
}

// ── Lists below their button (column) ──
const toggled = ref(new Map<string, boolean>())
const isOpen = (key: string, item: NavItem) =>
  toggled.value.get(key) ?? (!!item.defaultOpen || holdsCurrent(item, mounted.value))
function setOpen(key: string, on: boolean) {
  const next = new Map(toggled.value)
  next.set(key, on)
  toggled.value = next
}

// ── Lists that float (bar, collapsed column): one open at a time ──
const openKey = ref<string | null>(null)
const panel = shallowRef<HTMLElement | null>(null)
const trigger = shallowRef<HTMLElement | null>(null)
watch(
  [openKey, groups],
  () => {
    const at = openKey.value ? groups.value.flatMap((group, g) => group.map((item, i) => (keyOf(g, i, item) === openKey.value ? listId(g, i) : null))).find(Boolean) : null
    panel.value = at ? document.getElementById(at) : null
    trigger.value = at ? (root.value?.querySelector<HTMLElement>(`[aria-controls="${at}"]`) ?? null) : null
  },
  { flush: 'post' },
)
const floating = useFloating(trigger, panel, {
  placement: () => (horizontal.value ? 'bottom-start' : 'right-start'),
  offset: 6,
  flip: true,
  shift: true,
  strategy: 'fixed',
})
// Set on the open list from script, so the server's markup carries no style.
watchEffect(() => {
  const el = panel.value
  if (!el) return
  const { position, left, top } = floating.styles.value
  Object.assign(el.style, { position, left, top })
})
const close = () => (openKey.value = null)
useDismiss(panel, {
  onDismiss: close,
  ignore: () => [trigger.value],
  returnTo: () => trigger.value,
})
const subitems = (list: Element | null | undefined) =>
  [...(list?.querySelectorAll<HTMLElement>('[data-weft-subitem]') ?? [])].filter((el) => !el.hasAttribute('disabled') && el.getAttribute('aria-disabled') !== 'true')

function toggleFloat(key: string) {
  openKey.value = openKey.value === key ? null : key
}
async function enter(key: string, at: 'first' | 'last') {
  openKey.value = key
  await nextTick()
  const list = subitems(panel.value)
  focus(at === 'first' ? list[0] : list[list.length - 1])
}
function onTriggerKey(event: KeyboardEvent, key: string) {
  if (event.defaultPrevented || event.altKey || event.ctrlKey || event.metaKey) return
  const forward = isRtl(root.value) ? 'ArrowLeft' : 'ArrowRight'
  const opens = horizontal.value ? event.key === 'ArrowDown' : event.key === forward
  if (!opens && !(horizontal.value && event.key === 'ArrowUp')) return
  event.preventDefault()
  void enter(key, event.key === 'ArrowUp' ? 'last' : 'first')
}
function onPanelKey(event: KeyboardEvent) {
  if (event.defaultPrevented) return
  const list = subitems(event.currentTarget as HTMLElement)
  const at = list.indexOf(event.target as HTMLElement)
  if (at < 0) return
  const to =
    event.key === 'ArrowDown' ? (at + 1) % list.length
    : event.key === 'ArrowUp' ? (at - 1 + list.length) % list.length
    : event.key === 'Home' ? 0
    : event.key === 'End' ? list.length - 1
    : -1
  if (to < 0) return
  event.preventDefault()
  focus(list[to])
}
function onEntryFocusout(event: FocusEvent, key: string) {
  const to = event.relatedTarget as Node | null
  if (openKey.value === key && to && !(event.currentTarget as HTMLElement).contains(to)) close()
}

// ── A change of form ──
watch(form, async (now) => {
  if (!mounted.value) return
  const el = focused()
  const was = !!el && (!!root.value?.contains(el) || !!el.closest(`[data-weft-nav-sheet="${id}"]`))
  const label = el?.closest('.weft-nav__item')?.querySelector('.weft-nav__label')?.textContent?.trim()
  sheetOpen.value = false
  openKey.value = null
  if (!was) return
  await nextTick()
  focus(equivalent(now, label))
})
/** The control in the form now drawn that stands for `label`: its item, the button over it, else the first item; the menu button in a sheet form. */
function equivalent(now: 'bar' | 'sheet' | 'column', label: string | undefined): HTMLElement | null {
  const el = root.value
  if (!el) return null
  if (now === 'sheet') return el.querySelector<HTMLElement>('.weft-nav__menu')
  const labelled = [...el.querySelectorAll<HTMLElement>('.weft-nav__item')].find((item) => item.querySelector('.weft-nav__label')?.textContent?.trim() === label)
  if (labelled && !labelled.closest('[hidden]')) return labelled
  const list = labelled?.closest('ul.weft-nav__children')
  const over = list ? el.querySelector<HTMLElement>(`[aria-controls="${list.id}"]`) : null
  return over ?? el.querySelector<HTMLElement>('[data-weft-item]:not([disabled]):not([data-disabled])')
}
</script>

<template>
  <nav
    ref="root"
    class="weft-nav"
    :aria-label="label"
    :data-orientation="orientation"
    :data-form="mounted ? form : undefined"
    :data-collapsed="collapsed ? 'true' : undefined"
  >
    <span v-if="showSheet" class="weft-nav__menu-wrap">
      <Sheet v-model:open="sheetOpen" side="start" :title="label">
        <Button class="weft-nav__menu" variant="ghost" tone="neutral" square :aria-label="menuLabel">
          <template #leading><svg class="weft-nav__glyph" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M4 6h16" /><path d="M4 12h16" /><path d="M4 18h16" /></svg></template>
        </Button>
        <template #body>
          <NavigationMenu :items="items" orientation="vertical" :label="label" :data-weft-nav-sheet="id" @select="onSheetSelect">
            <template v-if="$slots.item" #item="{ item }"><slot name="item" :item="item" /></template>
          </NavigationMenu>
        </template>
      </Sheet>
    </span>
    <template v-if="showBar">
      <ul v-for="(group, g) in groups" :key="g" class="weft-nav__list">
        <li
          v-for="(item, i) in group"
          :key="keyOf(g, i, item)"
          class="weft-nav__entry"
          :data-type="item.type"
          @focusout="onEntryFocusout($event, keyOf(g, i, item))"
        >
          <span v-if="item.type === 'label'" class="weft-nav__heading" :class="{ 'weft-visually-hidden': collapsed }">{{ item.label }}</span>
          <template v-else-if="item.children?.length && floats">
            <button
              type="button"
              class="weft-nav__item weft-nav__trigger"
              data-weft-item
              :aria-expanded="openKey === keyOf(g, i, item) ? 'true' : 'false'"
              :aria-controls="listId(g, i)"
              :disabled="item.disabled || undefined"
              :data-active="holdsCurrent(item, mounted) ? 'true' : undefined"
              @click="toggleFloat(keyOf(g, i, item))"
              @keydown="onTriggerKey($event, keyOf(g, i, item))"
            >
              <Icon v-if="item.icon" :name="item.icon" class="weft-nav__icon" />
              <span class="weft-nav__label" :class="{ 'weft-visually-hidden': collapsed }"><slot name="item" :item="item">{{ item.label }}</slot></span>
              <svg v-if="!collapsed" class="weft-nav__glyph weft-nav__chevron" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="m6 9 6 6 6-6" /></svg>
            </button>
            <ul
              :id="listId(g, i)"
              class="weft-nav__panel"
              data-float="true"
              :hidden="openKey !== keyOf(g, i, item) || undefined"
              :data-positioned="openKey === keyOf(g, i, item) && floating.isPositioned.value ? 'true' : undefined"
              @keydown="onPanelKey"
            >
              <li v-for="(child, c) in item.children" :key="c" class="weft-nav__entry">
                <Link
                  v-if="child.to"
                  class="weft-nav__item weft-nav__child"
                  data-weft-subitem
                  :to="child.to"
                  :target="child.target"
                  :active="child.active"
                  :disabled="child.disabled"
                  @click="onItem(child, $event)"
                >
                  <Icon v-if="child.icon" :name="child.icon" class="weft-nav__icon" />
                  <span class="weft-nav__text">
                    <span class="weft-nav__label"><slot name="item" :item="child">{{ child.label }}</slot></span>
                    <span v-if="child.description" class="weft-nav__description">{{ child.description }}</span>
                  </span>
                </Link>
                <button v-else type="button" class="weft-nav__item weft-nav__child" data-weft-subitem :disabled="child.disabled || undefined" @click="onItem(child, $event)">
                  <Icon v-if="child.icon" :name="child.icon" class="weft-nav__icon" />
                  <span class="weft-nav__label"><slot name="item" :item="child">{{ child.label }}</slot></span>
                </button>
              </li>
            </ul>
          </template>
          <template v-else-if="item.children?.length">
            <button
              type="button"
              class="weft-nav__item weft-nav__toggle"
              data-weft-item
              :aria-expanded="isOpen(keyOf(g, i, item), item) ? 'true' : 'false'"
              :aria-controls="listId(g, i)"
              :disabled="item.disabled || undefined"
              :data-active="holdsCurrent(item, mounted) ? 'true' : undefined"
              @click="setOpen(keyOf(g, i, item), !isOpen(keyOf(g, i, item), item))"
            >
              <Icon v-if="item.icon" :name="item.icon" class="weft-nav__icon" />
              <span class="weft-nav__label"><slot name="item" :item="item">{{ item.label }}</slot></span>
              <Badge v-if="item.badge !== undefined" :label="item.badge" variant="soft" tone="neutral" size="sm" />
              <svg class="weft-nav__glyph weft-nav__chevron" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="m6 9 6 6 6-6" /></svg>
            </button>
            <ul
              :id="listId(g, i)"
              class="weft-nav__children"
              :hidden.attr="isOpen(keyOf(g, i, item), item) ? undefined : 'until-found'"
              @beforematch="setOpen(keyOf(g, i, item), true)"
            >
              <li v-for="(child, c) in item.children" :key="c" class="weft-nav__entry">
                <Link
                  v-if="child.to"
                  class="weft-nav__item weft-nav__child"
                  :data-weft-item="isOpen(keyOf(g, i, item), item) ? '' : undefined"
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
                  :data-weft-item="isOpen(keyOf(g, i, item), item) ? '' : undefined"
                  :disabled="child.disabled || undefined"
                  @click="onItem(child, $event)"
                >
                  <Icon v-if="child.icon" :name="child.icon" class="weft-nav__icon" />
                  <span class="weft-nav__label"><slot name="item" :item="child">{{ child.label }}</slot></span>
                </button>
              </li>
            </ul>
          </template>
          <!-- Collapsed, an item names itself in a Tooltip; open, it needs none, so a page that is not
               collapsed renders no Tooltip and ships none of its CSS. -->
          <Tooltip v-else-if="item.to && collapsed" :text="item.label" placement="right">
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
          <Link
            v-else-if="item.to"
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
          <Tooltip v-else-if="collapsed" :text="item.label" placement="right">
            <button type="button" class="weft-nav__item" data-weft-item :disabled="item.disabled || undefined" @click="onItem(item, $event)">
              <Icon v-if="item.icon" :name="item.icon" class="weft-nav__icon" />
              <span class="weft-nav__label" :class="{ 'weft-visually-hidden': collapsed }"><slot name="item" :item="item">{{ item.label }}</slot></span>
              <Badge v-if="item.badge !== undefined && !collapsed" :label="item.badge" variant="soft" tone="neutral" size="sm" />
            </button>
          </Tooltip>
          <button v-else type="button" class="weft-nav__item" data-weft-item :disabled="item.disabled || undefined" @click="onItem(item, $event)">
            <Icon v-if="item.icon" :name="item.icon" class="weft-nav__icon" />
            <span class="weft-nav__label" :class="{ 'weft-visually-hidden': collapsed }"><slot name="item" :item="item">{{ item.label }}</slot></span>
            <Badge v-if="item.badge !== undefined && !collapsed" :label="item.badge" variant="soft" tone="neutral" size="sm" />
          </button>
        </li>
      </ul>
    </template>
  </nav>
</template>
