<script setup vapor lang="ts">
// A menu of actions opened from a button (WAI-ARIA APG, Menu Button and
// Menu): items, groups with headings, separators, checkbox and radio items,
// links and submenus. The trigger opens it on a press, Enter, Space or
// ArrowDown (first item focused) or ArrowUp (last). Inside: the arrows move
// and loop, Home and End go to the ends, typing jumps to an item by its
// label, Enter and Space choose. Choosing closes every level and returns
// focus to the trigger; Escape closes one level; Tab closes the menu (as a
// sheet too). A submenu opens with ArrowRight (ArrowLeft right to left),
// Enter, a press or a hover, and closes with the opposite arrow. Disabled items are reached
// but not chosen, and so are loading ones. While open, the page does not
// scroll. On pointer it floats beside the trigger; on touch the same menu
// is a bottom sheet with large items, modal (focus trapped, the page behind
// inert), and a submenu is a sheet above it with a back item. It fades out
// when it closes, unless motion is reduced. With `:portal="false"` it is
// drawn where it stands instead of in the overlay layer at the end of the
// body, and an open one is drawn by the server too: in place, unpositioned
// until mounted, so a page can show it open with no script (a picture of an
// app, made inert, which the page positions with its own CSS).
import { computed, inject, onScopeDispose, provide, ref, shallowRef, useTemplateRef, watch } from 'vue'
import { useWeft } from '../adaptive/context.ts'
import Icon from '../basics/Icon.vue'
import Kbd from '../basics/Kbd.vue'
import Spinner from '../basics/Spinner.vue'
import { focus, isRtl } from '../headless/dom.ts'
import { useControllable } from '../headless/useControllable.ts'
import { useDismiss } from '../headless/useDismiss.ts'
import { useFloating, type Placement } from '../headless/useFloating.ts'
import { useFocusTrap } from '../headless/useFocusTrap.ts'
import { useId } from '../headless/useId.ts'
import { useRovingFocus } from '../headless/useRovingFocus.ts'
import { useInert } from '../headless/useInert.ts'
import { useScrollLock } from '../headless/useScrollLock.ts'
import { useTypeahead } from '../headless/useTypeahead.ts'
import { groupsOf, isChoice, menuLevel, roleOf, type MenuItem } from './menu.ts'
import { elementOf, isTouch, overlayRoot, useMounted, useOverlayContext, usePresence, useTrigger, type Anchor } from './overlay.ts'
import DropdownMenu from './DropdownMenu.vue'
import '../styles/base.css'
import '../styles/tone.css'
import '../styles/overlay.css'
import '../styles/sheet.css'
import '../styles/dropdown-menu.css'

defineOptions({ inheritAttrs: false })

const props = withDefaults(
  defineProps<{
    /** The items, or groups of them (an array of arrays), drawn with separators between. */
    items?: MenuItem[] | MenuItem[][]
    /** Open or closed, bound with `v-model:open`. */
    open?: boolean
    /** Whether it starts open when `v-model:open` is not bound. */
    defaultOpen?: boolean
    /** Where it floats against the trigger on pointer (start and end follow the text direction). */
    placement?: Placement
    /** Float against this element or point instead of the trigger (ContextMenu gives the pointer's). */
    anchor?: Anchor
    /** Draw it in the overlay layer at the end of the body; false draws it in place, and an open one on the server too. */
    portal?: boolean
  }>(),
  { items: () => [], open: undefined, defaultOpen: false, placement: 'bottom-start', portal: true },
)

const emit = defineEmits<{
  /** Opened or closed. */
  'update:open': [open: boolean]
}>()

defineSlots<{
  /** The trigger: one element (a Button). */
  default?: (props: { open: boolean }) => unknown
  /** An item's label, in place of its `label`. */
  item?: (props: { item: MenuItem }) => unknown
}>()

const parent = inject(menuLevel, null)
const open = useControllable(props, 'open', emit, () => props.defaultOpen)
const weft = useWeft()
const sheet = computed(() => weft.input.value === 'touch')
const mounted = useMounted()
const id = useId('menu')
const triggerId = `${id}-trigger`
const wrap = useTemplateRef<HTMLElement>('wrap')
const panel = useTemplateRef<HTMLElement>('panel')
const backdrop = useTemplateRef<HTMLElement>('backdrop')
const groups = computed(() => groupsOf(props.items))
const presence = usePresence(open, () => panel.value)
/** The panel while it is open, not while it leaves. */
const live = computed(() => (presence.closing.value ? null : panel.value))
/** As a sheet it is modal. */
const modal = computed(() => open.value && sheet.value)

const trigger = useTrigger(wrap, () => {
  const el = trigger.el.value
  return {
    'aria-haspopup': 'menu',
    'aria-expanded': open.value ? 'true' : 'false',
    'aria-controls': open.value ? id : undefined,
    id: el && (!el.id || trigger.owns('id')) ? triggerId : undefined,
  }
})
const anchorEl = () => elementOf(props.anchor)
const context = useOverlayContext(() => trigger.el.value ?? anchorEl() ?? props.anchor?.contextElement ?? wrap.value)
const reference = computed(() => props.anchor ?? trigger.el.value)
const floating = useFloating(reference, computed(() => (sheet.value ? null : panel.value)), {
  placement: () => props.placement,
  offset: parent ? 2 : 6,
  flip: true,
  shift: true,
  size: true,
})

/** Which item to focus when the menu appears. */
let start: 'first' | 'last' = 'first'
/** Closed by a press outside a level above: return no focus, as after any press outside. */
let pressedAway = false
const close = () => (open.value = false)
watch(open, (on) => on && (pressedAway = false), { flush: 'sync' })
function closeAll(returnFocus: boolean) {
  if (parent) return parent.closeAll(returnFocus)
  if (!returnFocus) pressedAway = true
  close()
  // At once, before an item's onSelect has its effect: a dialog it opens
  // then finds focus on the trigger, its opener to return to.
  if (returnFocus) focus(trigger.el.value)
}
function show(at: 'first' | 'last') {
  start = at
  open.value = true
}
function onTriggerClick(event: MouseEvent) {
  if (event.defaultPrevented) return
  if (open.value) close()
  else show('first')
}
function onTriggerKey(event: KeyboardEvent) {
  if (event.key !== 'ArrowDown' && event.key !== 'ArrowUp') return
  event.preventDefault()
  show(event.key === 'ArrowDown' ? 'first' : 'last')
}

useDismiss(live, {
  onDismiss(reason, event) {
    if (!parent || reason === 'escape' || sheet.value) return close()
    // A press in a level above closes this one; a press outside them all, every level.
    const target = event.target as Node
    if (parent.panels().some((p) => p?.contains(target))) close()
    else closeAll(false)
  },
  ignore: () => [wrap.value, backdrop.value, anchorEl()],
  returnTo: () => trigger.el.value ?? anchorEl(),
  returnFocus: () => !pressedAway,
})
useFocusTrap(live, {
  active: modal,
  // A sheet that becomes a floating menu (touch to pointer) keeps its focus.
  returnFocus: () => !open.value,
  returnTo: () => trigger.el.value ?? anchorEl(),
  initial: () => edge(start),
})
useScrollLock(computed(() => open.value && !parent))
useInert(computed(() => modal.value && !parent))
watch(open, (on) => on && context.refresh(), { immediate: true, flush: 'sync' })

const roving = useRovingFocus(panel, { orientation: 'vertical' })
const itemEls = () => [...(panel.value?.querySelectorAll<HTMLElement>('[data-weft-item]') ?? [])]
const enabled = (el: HTMLElement) => el.getAttribute('aria-disabled') !== 'true'
const typeahead = useTypeahead(
  () => itemEls().map((el) => (enabled(el) ? (el.querySelector('.weft-menu__text')?.textContent ?? '') : '')),
  (i) => roving.focus(i),
)
function edge(at: 'first' | 'last') {
  const list = itemEls().filter(enabled)
  return (at === 'first' ? list[0] : list[list.length - 1]) ?? null
}
const focusEdge = (at: 'first' | 'last') => focus(edge(at))
watch(
  panel,
  (el) => {
    if (!el) return
    if (parent) {
      if (parent.focusFirst()) focusEdge('first')
    } else focusEdge(start)
  },
  { flush: 'post' },
)
if (parent) watch(parent.refocus, () => focusEdge('first'))

// The open submenu: one at a time, keyed by the item's place.
const sub = shallowRef<{ key: string; item: MenuItem; el: HTMLElement; focus: boolean } | null>(null)
const subs = computed(() => (sub.value ? [sub.value] : []))
const refocus = ref(0)
let hoverTimer: ReturnType<typeof setTimeout> | undefined
const cancelHover = () => {
  if (hoverTimer !== undefined) clearTimeout(hoverTimer)
  hoverTimer = undefined
}
onScopeDispose(cancelHover)
function openSub(item: MenuItem, key: string, el: HTMLElement, focusFirst: boolean) {
  cancelHover()
  if (sub.value?.key === key) {
    if (focusFirst) refocus.value++
    return
  }
  sub.value = { key, item, el, focus: focusFirst }
}
const closeSub = () => (sub.value = null)
watch(open, (on) => !on && closeSub())
const subPlacement = computed<Placement>(() => (isRtl(panel.value) ? 'left-start' : 'right-start'))

provide(menuLevel, {
  closeAll,
  panels: () => [...(parent?.panels() ?? []), panel.value],
  item: () => sub.value?.item,
  focusFirst: () => !!sub.value?.focus,
  refocus,
  keep: cancelHover,
})

/** A link only while it can be followed: disabled or loading, it has no address (no Ctrl-click, no middle click). */
const linked = (item: MenuItem) => !!item.to && !item.disabled && !item.loading
const itemOf = (event: Event) => (event.target as Element).closest<HTMLElement>('[data-weft-item]')

function choose(item: MenuItem, key: string, event: MouseEvent) {
  const el = itemOf(event)
  if (item.disabled || item.loading) {
    event.preventDefault()
    return
  }
  if (item.children) {
    event.preventDefault()
    if (el) openSub(item, key, el, true)
    return
  }
  if (item.type === 'checkbox') item.onUpdateChecked?.(!item.checked)
  else if (item.type === 'radio' && !item.checked) item.onUpdateChecked?.(true)
  const select = new Event('select', { cancelable: true })
  item.onSelect?.(select)
  if (!select.defaultPrevented) closeAll(true)
}

function onHover(item: MenuItem, key: string, event: PointerEvent) {
  if (isTouch(event) || sheet.value) return
  const el = itemOf(event)
  if (!el) return
  if (document.activeElement !== el) focus(el)
  if (item.children && !item.disabled) {
    if (sub.value?.key === key) return cancelHover()
    cancelHover()
    hoverTimer = setTimeout(() => openSub(item, key, el, false), OPEN_DELAY)
  } else if (sub.value && hoverTimer === undefined) {
    hoverTimer = setTimeout(closeSub, CLOSE_DELAY)
  }
}

// Tab closes the menu, every level, in both forms. Heard in the capture
// phase on the panel, before the touch sheet's focus trap could cycle it.
function onTab(event: KeyboardEvent) {
  if (event.key !== 'Tab' || event.defaultPrevented) return
  event.preventDefault()
  closeAll(true)
}
watch(panel, (el, _, cleanup) => {
  if (!el) return
  el.addEventListener('keydown', onTab, true)
  cleanup(() => el.removeEventListener('keydown', onTab, true))
})

function onKeydown(event: KeyboardEvent) {
  if (event.defaultPrevented) return
  if (typeahead.onKeydown(event, roving.active.value)) {
    event.preventDefault()
    return
  }
  const el = itemOf(event)
  if (!el) return
  const rtl = isRtl(el)
  const forward = rtl ? 'ArrowLeft' : 'ArrowRight'
  const back = rtl ? 'ArrowRight' : 'ArrowLeft'
  if (event.key === 'Enter' || event.key === ' ') {
    event.preventDefault()
    el.click()
  } else if (event.key === forward && el.getAttribute('aria-haspopup') === 'menu') {
    event.preventDefault()
    el.click()
  } else if (event.key === back && parent) {
    event.preventDefault()
    close()
  }
}

function onBackdrop() {
  close()
}

const side = computed(() => (sheet.value ? 'bottom' : floating.placement.value.split('-')[0]))
</script>

<script lang="ts">
/** ms of hover on a submenu's item before it opens, and on another item before it closes. */
const OPEN_DELAY = 100
const CLOSE_DELAY = 150
</script>

<template>
  <span v-if="!parent && $slots.default" ref="wrap" class="weft-overlay-trigger" @click="onTriggerClick" @keydown="onTriggerKey"><slot :open="open" /></span>
  <Teleport v-if="(mounted || !portal) && presence.present.value" :to="portal ? overlayRoot() : 'body'" :disabled="!portal">
    <div class="weft-overlay" v-bind="context.attrs.value" :data-state="presence.closing.value ? 'closed' : 'open'">
      <div v-if="sheet" ref="backdrop" class="weft-overlay-backdrop" @click="onBackdrop"></div>
      <div
        ref="panel"
        v-bind="$attrs"
        :id="id"
        class="weft-menu"
        :class="{ 'weft-sheet': sheet }"
        role="menu"
        tabindex="-1"
        :aria-labelledby="!parent && trigger.el.value?.id ? trigger.el.value.id : undefined"
        :aria-label="parent ? parent.item()?.label : ($attrs['aria-label'] as string | undefined)"
        :aria-modal="sheet ? 'true' : undefined"
        :data-form="sheet ? 'sheet' : 'floating'"
        :data-side="side"
        :data-positioned="sheet || !mounted || floating.isPositioned.value ? 'true' : undefined"
        :style="sheet || !mounted ? undefined : floating.styles.value"
        @keydown="onKeydown"
        @pointerenter="parent?.keep()"
      >
        <div v-if="sheet && parent" class="weft-menu__item weft-menu__back" role="menuitem" data-weft-item @click="close()">
          <svg class="weft-menu__glyph weft-menu__chevron" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="m15 18-6-6 6-6" /></svg>
          <span class="weft-menu__text">{{ parent.item()?.label }}</span>
        </div>
        <template v-for="(group, g) in groups" :key="g">
          <div v-if="g > 0" class="weft-menu__separator" role="separator"></div>
          <div class="weft-menu__group" role="group" :aria-labelledby="group[0]?.type === 'label' ? `${id}-g${g}` : undefined">
            <template v-for="(item, i) in group" :key="i">
              <div v-if="item.type === 'label'" :id="i === 0 ? `${id}-g${g}` : undefined" class="weft-menu__heading">{{ item.label }}</div>
              <div v-else-if="item.type === 'separator'" class="weft-menu__separator" role="separator"></div>
              <component
                :is="linked(item) ? 'a' : 'div'"
                v-else
                class="weft-menu__item"
                data-weft-item
                :role="roleOf(item)"
                :href="linked(item) ? item.to : undefined"
                :target="linked(item) ? item.target : undefined"
                :aria-disabled="item.disabled || item.loading ? 'true' : undefined"
                :aria-busy="item.loading ? 'true' : undefined"
                :aria-describedby="item.description ? `${id}-d${g}-${i}` : undefined"
                :aria-checked="isChoice(item) ? (item.checked ? 'true' : 'false') : undefined"
                :aria-haspopup="item.children ? 'menu' : undefined"
                :aria-expanded="item.children ? (sub?.key === `${g}-${i}` ? 'true' : 'false') : undefined"
                :data-tone="item.color ?? item.tone"
                @click="choose(item, `${g}-${i}`, $event)"
                @pointermove="onHover(item, `${g}-${i}`, $event)"
              >
                <span v-if="item.loading" class="weft-menu__glyph"><Spinner /></span>
                <span v-else-if="isChoice(item)" class="weft-menu__glyph weft-menu__indicator" aria-hidden="true">
                  <svg v-if="item.checked && item.type === 'checkbox'" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><path d="M20 6 9 17l-5-5" /></svg>
                  <svg v-else-if="item.checked" viewBox="0 0 24 24" fill="currentColor"><circle cx="12" cy="12" r="4.5" /></svg>
                </span>
                <span v-else-if="item.icon" class="weft-menu__glyph"><Icon :name="item.icon" /></span>
                <span class="weft-menu__label">
                  <span class="weft-menu__text"><slot name="item" :item="item">{{ item.label }}</slot></span>
                  <span v-if="item.description" :id="`${id}-d${g}-${i}`" class="weft-menu__description">{{ item.description }}</span>
                </span>
                <span v-if="item.kbds?.length" class="weft-menu__kbds"><Kbd v-for="k in item.kbds" :key="k" :value="k" size="sm" /></span>
                <svg v-if="item.children" class="weft-menu__glyph weft-menu__chevron" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="m9 18 6-6-6-6" /></svg>
              </component>
            </template>
          </div>
        </template>
        <DropdownMenu
          v-for="s in subs"
          :key="s.key"
          :items="s.item.children"
          :anchor="s.el"
          :open="true"
          :placement="subPlacement"
          :portal="portal"
          @update:open="(on: boolean) => !on && closeSub()"
        >
          <template v-if="$slots.item" #item="{ item }"><slot name="item" :item="item" /></template>
        </DropdownMenu>
      </div>
    </div>
  </Teleport>
</template>
