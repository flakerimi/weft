<script setup vapor lang="ts">
// The frame of an app: navigation, a header and the page's main content
// (the <main> landmark), in one of three layouts it picks itself.
//
// - `sidebar` (a regular or wide viewport): a sidebar of the items as a
//   vertical NavigationMenu beside a header and the content. The header's
//   toggle collapses the sidebar to icons (labels in tooltips) and back: a
//   button, so the keyboard reaches it, saying aria-expanded and
//   aria-controls. The choice is kept in localStorage under `persist`, read
//   after mounting (never on the server), so a reload keeps it.
// - `bar` (a narrow touch screen): a bottom navigation bar of up to five
//   links, icon over label; when there are more, four and a "More" button
//   that opens the rest in a bottom Sheet.
// - `drawer` (a narrow pointer screen): the header's menu button opens the
//   sidebar's content in a Sheet from the start side.
//
// The server and the first render in the browser draw every part and let
// CSS pick the layout from the viewport (`max-width` and `pointer: coarse`),
// so a page that is never hydrated still lays out right; once mounted, the
// component sets the layout from Weft's input and the viewport's width and
// renders only that layout's parts, so each slot is drawn once (before
// mounting the brand slot is drawn twice, in the sidebar and in the header:
// give it no ids). A pre-paint script (usePrePaint) copies the stored
// collapsed state to the root element before the first paint, so CSS draws
// the rail at once and a reload does not flash the sidebar open. When the
// layout changes, an open drawer or More sheet closes, and focus, if it was
// in a part that goes, moves to the same item in the new layout (or its
// menu button, toggle or bar). Current pages say aria-current="page"
// (Link's rule).
import { computed, nextTick, onMounted, shallowRef, useTemplateRef, watch } from 'vue'
import Button from '../basics/Button.vue'
import Icon from '../basics/Icon.vue'
import Link from '../basics/Link.vue'
import { useWeft } from '../adaptive/context.ts'
import { useWidth } from '../adaptive/useWidth.ts'
import { focus, focused } from '../headless/dom.ts'
import { useControllable } from '../headless/useControllable.ts'
import { useId } from '../headless/useId.ts'
import NavigationMenu from '../navigation/NavigationMenu.vue'
import { navGroups, type NavItem } from '../navigation/nav.ts'
import Sheet from '../overlays/Sheet.vue'
import { usePrePaint } from '../head/usePrePaint.ts'
import { APP_SHELL_LABELS, collapsedSnippet, readCollapsed, splitBar, writeCollapsed, type AppShellLabels } from './appShell.ts'
import '../styles/base.css'
import '../styles/app-shell.css'

const props = withDefaults(
  defineProps<{
    /** The navigation: `{ label, to, icon, active, badge, children, … }`, or groups of them. Give each an `icon` for the collapsed sidebar and the bottom bar. */
    items?: NavItem[] | NavItem[][]
    /** The app's name, at the top of the sidebar (and in the header when there is no sidebar), as a link to `to`. The brand slot replaces it. */
    title?: string
    /** Where the title links. */
    to?: string
    /** The sidebar shows icons alone, bound with `v-model:collapsed`. */
    collapsed?: boolean
    /** Whether the sidebar starts collapsed when `v-model:collapsed` is not bound and nothing is stored. */
    defaultCollapsed?: boolean
    /** The localStorage key the collapsed state is kept under; false keeps nothing. */
    persist?: string | false
    /** The name of the navigation landmarks. */
    label?: string
    /** The words it says, for assistive technology; give some or all to translate them. */
    labels?: Partial<AppShellLabels>
  }>(),
  { items: () => [], to: '/', collapsed: undefined, defaultCollapsed: false, persist: 'weft:app-shell:collapsed', label: 'Main' },
)

const emit = defineEmits<{
  /** The sidebar collapsed or expanded. */
  'update:collapsed': [collapsed: boolean]
  /** A navigation item was chosen, in the sidebar, the bottom bar or a sheet. */
  select: [item: NavItem, event: Event]
}>()

defineSlots<{
  /** The app's name or logo, in place of the title; `collapsed` says whether the sidebar is a rail. */
  brand?: (props: { collapsed: boolean }) => unknown
  /** The middle of the header (a search field, the page's title). */
  header?: () => unknown
  /** The end of the header (account, notifications). */
  actions?: () => unknown
  /** The sidebar's content, in place of the navigation built from `items`. */
  sidebar?: (props: { collapsed: boolean }) => unknown
  /** The bottom of the sidebar (the signed-in user). */
  footer?: (props: { collapsed: boolean }) => unknown
  /** The page's content, in <main>. */
  default?: () => unknown
}>()

const t = computed<AppShellLabels>(() => ({ ...APP_SHELL_LABELS, ...props.labels }))
const weft = useWeft()
const id = useId('app-shell')
const sidebarId = `${id}-sidebar`
const collapsed = useControllable(props, 'collapsed', emit, () => props.defaultCollapsed)
const root = useTemplateRef<HTMLElement>('root')
if (props.persist) usePrePaint(`app-shell:${props.persist}`, collapsedSnippet(props.persist))

// Narrow is the viewport's width: the root element's.
const viewport = shallowRef<HTMLElement | null>(null)
const mounted = shallowRef(false)
onMounted(() => {
  viewport.value = document.documentElement
  mounted.value = true
  if (props.persist) {
    const stored = readCollapsed(props.persist)
    if (stored !== undefined) collapsed.value = stored
  }
})
watch(collapsed, (on) => props.persist && mounted.value && writeCollapsed(props.persist, on))
const width = useWidth(viewport)
/** Unset until mounted: CSS picks from the viewport. */
const layout = computed(() => {
  if (!mounted.value) return undefined
  if (width.value !== 'narrow') return 'sidebar'
  return weft.input.value === 'touch' ? 'bar' : 'drawer'
})

const all = computed(() => navGroups(props.items).flat().filter((item) => item.type !== 'label'))
const bar = computed(() => splitBar(all.value))

/** Before mounting every part renders (CSS picks); after, only the layout's. */
const shows = (name: 'sidebar' | 'bar' | 'drawer') => !mounted.value || layout.value === name

const drawerOpen = shallowRef(false)
const moreOpen = shallowRef(false)

// A change of layout closes the sheets, and moves focus out of a part that goes.
const GOING = '.weft-app-shell__sidebar, .weft-app-shell__collapse, .weft-app-shell__menu, .weft-app-shell__bar'
watch(layout, async (now, was) => {
  if (!now || !was) return
  const el = focused()
  const moving = !!el && (!!el.closest(`[data-weft-shell-sheet="${id}"]`) || (!!root.value?.contains(el) && !!el.closest(GOING)))
  const label = el?.closest('.weft-nav__item, .weft-app-shell__bar-item')?.querySelector('.weft-nav__label, .weft-app-shell__bar-label')?.textContent?.trim()
  drawerOpen.value = false
  moreOpen.value = false
  if (!moving) return
  await nextTick()
  focus(equivalent(now, label))
})
/** The control in `now` that stands for `label`: the same item, else the layout's own control. */
function equivalent(now: 'sidebar' | 'bar' | 'drawer', label: string | undefined): HTMLElement | null {
  const el = root.value
  if (!el) return null
  const named = (selector: string, text: string) =>
    [...el.querySelectorAll<HTMLElement>(selector)].find((item) => !item.closest('[hidden]') && item.querySelector(text)?.textContent?.trim() === label)
  if (now === 'drawer') return el.querySelector<HTMLElement>('.weft-app-shell__menu button')
  if (now === 'bar') return named('.weft-app-shell__bar-item', '.weft-app-shell__bar-label') ?? el.querySelector<HTMLElement>('.weft-app-shell__bar-item')
  return named('.weft-app-shell__sidebar .weft-nav__item', '.weft-nav__label') ?? el.querySelector<HTMLElement>('.weft-app-shell__collapse')
}
function chosen(item: NavItem, event: Event) {
  drawerOpen.value = false
  moreOpen.value = false
  emit('select', item, event)
}
function onBarItem(item: NavItem, event: MouseEvent) {
  if (item.disabled) return
  item.onSelect?.(event)
  chosen(item, event)
}
</script>

<template>
  <div ref="root" class="weft-app-shell" :data-layout="layout" :data-collapsed="collapsed ? 'true' : undefined">
    <aside v-if="shows('sidebar')" :id="sidebarId" class="weft-app-shell__sidebar">
      <div class="weft-app-shell__brand">
        <slot name="brand" :collapsed="collapsed">
          <Link v-if="title" class="weft-app-shell__title" standalone :to="to" :aria-label="collapsed ? title : undefined">
            <span :class="{ 'weft-visually-hidden': collapsed }">{{ title }}</span>
            <span v-if="collapsed" class="weft-app-shell__initial" aria-hidden="true">{{ title.slice(0, 1) }}</span>
          </Link>
        </slot>
      </div>
      <div class="weft-app-shell__nav">
        <slot name="sidebar" :collapsed="collapsed">
          <NavigationMenu :items="items" orientation="vertical" :collapsed="collapsed" :label="label" @select="chosen" />
        </slot>
      </div>
      <div v-if="$slots.footer" class="weft-app-shell__sidebar-footer"><slot name="footer" :collapsed="collapsed" /></div>
    </aside>
    <header class="weft-app-shell__header">
      <Button
        v-if="shows('sidebar')"
        class="weft-app-shell__collapse"
        variant="ghost"
        tone="neutral"
        square
        :aria-label="t.sidebar"
        :aria-expanded="collapsed ? 'false' : 'true'"
        :aria-controls="sidebarId"
        @click="collapsed = !collapsed"
      >
        <template #leading><svg class="weft-app-shell__glyph" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><rect width="18" height="18" x="3" y="3" rx="2" /><path d="M9 3v18" /></svg></template>
      </Button>
      <span v-if="shows('drawer')" class="weft-app-shell__menu">
        <Sheet v-model:open="drawerOpen" side="start" :title="title || t.menu" :data-weft-shell-sheet="id">
          <Button variant="ghost" tone="neutral" square :aria-label="t.menu">
            <template #leading><svg class="weft-app-shell__glyph" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M4 6h16" /><path d="M4 12h16" /><path d="M4 18h16" /></svg></template>
          </Button>
          <template #body>
            <slot name="sidebar" :collapsed="false">
              <NavigationMenu :items="items" orientation="vertical" :label="label" @select="chosen" />
            </slot>
          </template>
          <template v-if="$slots.footer" #footer><slot name="footer" :collapsed="false" /></template>
        </Sheet>
      </span>
      <span v-if="(title || $slots.brand) && (!mounted || layout !== 'sidebar')" class="weft-app-shell__heading">
        <slot name="brand" :collapsed="false"><Link class="weft-app-shell__title" standalone :to="to">{{ title }}</Link></slot>
      </span>
      <div class="weft-app-shell__header-main"><slot name="header" /></div>
      <div v-if="$slots.actions" class="weft-app-shell__actions"><slot name="actions" /></div>
    </header>
    <main class="weft-app-shell__main"><slot /></main>
    <nav v-if="shows('bar')" class="weft-app-shell__bar" :aria-label="label">
      <ul class="weft-app-shell__bar-list">
        <li v-for="(item, i) in bar.shown" :key="i" class="weft-app-shell__bar-entry">
          <Link
            class="weft-app-shell__bar-item"
            :to="item.to"
            :target="item.target"
            :active="item.active"
            :disabled="item.disabled"
            @click="onBarItem(item, $event)"
          >
            <Icon v-if="item.icon" :name="item.icon" class="weft-app-shell__bar-icon" />
            <span class="weft-app-shell__bar-label">{{ item.label }}</span>
          </Link>
        </li>
        <li v-if="bar.more.length" class="weft-app-shell__bar-entry">
          <Sheet v-model:open="moreOpen" side="bottom" :title="t.more" :data-weft-shell-sheet="id">
            <button type="button" class="weft-app-shell__bar-item">
              <svg class="weft-app-shell__bar-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><circle cx="12" cy="12" r="1" /><circle cx="19" cy="12" r="1" /><circle cx="5" cy="12" r="1" /></svg>
              <span class="weft-app-shell__bar-label">{{ t.more }}</span>
            </button>
            <template #body>
              <NavigationMenu :items="bar.more" orientation="vertical" :label="t.more" @select="chosen" />
            </template>
          </Sheet>
        </li>
      </ul>
    </nav>
  </div>
</template>
