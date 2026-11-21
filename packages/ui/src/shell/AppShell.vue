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
// component sets the layout from Weft's input and the viewport's width.
// Current pages say aria-current="page" (Link's rule).
import { computed, onMounted, shallowRef, watch } from 'vue'
import Button from '../basics/Button.vue'
import Icon from '../basics/Icon.vue'
import Link from '../basics/Link.vue'
import { useWeft } from '../adaptive/context.ts'
import { useWidth } from '../adaptive/useWidth.ts'
import { useControllable } from '../headless/useControllable.ts'
import { useId } from '../headless/useId.ts'
import NavigationMenu from '../navigation/NavigationMenu.vue'
import { navGroups, type NavItem } from '../navigation/nav.ts'
import Sheet from '../overlays/Sheet.vue'
import { APP_SHELL_LABELS, readCollapsed, splitBar, writeCollapsed, type AppShellLabels } from './appShell.ts'
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

const drawerOpen = shallowRef(false)
const moreOpen = shallowRef(false)
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
  <div class="weft-app-shell" :data-layout="layout" :data-collapsed="collapsed ? 'true' : undefined">
    <aside :id="sidebarId" class="weft-app-shell__sidebar">
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
        class="weft-app-shell__collapse"
        variant="ghost"
        tone="neutral"
        square
        :aria-label="collapsed ? t.expand : t.collapse"
        :aria-expanded="collapsed ? 'false' : 'true'"
        :aria-controls="sidebarId"
        @click="collapsed = !collapsed"
      >
        <template #leading><svg class="weft-app-shell__glyph" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><rect width="18" height="18" x="3" y="3" rx="2" /><path d="M9 3v18" /></svg></template>
      </Button>
      <span class="weft-app-shell__menu">
        <Sheet v-model:open="drawerOpen" side="start" :title="title || t.menu">
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
      <span v-if="title || $slots.brand" class="weft-app-shell__heading">
        <slot name="brand" :collapsed="false"><Link class="weft-app-shell__title" standalone :to="to">{{ title }}</Link></slot>
      </span>
      <div class="weft-app-shell__header-main"><slot name="header" /></div>
      <div v-if="$slots.actions" class="weft-app-shell__actions"><slot name="actions" /></div>
    </header>
    <main class="weft-app-shell__main"><slot /></main>
    <nav class="weft-app-shell__bar" :aria-label="label">
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
          <Sheet v-model:open="moreOpen" side="bottom" :title="t.more">
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
