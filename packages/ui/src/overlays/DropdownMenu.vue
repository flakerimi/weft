<script setup vapor lang="ts">
// A menu of actions opened from a button (WAI-ARIA APG, Menu Button and
// Menu): items, groups with headings, separators, checkbox and radio items,
// links and submenus. The trigger opens it on a press, Enter, Space or
// ArrowDown (first item focused) or ArrowUp (last). Inside: the arrows move
// and loop, Home and End go to the ends, typing jumps to an item by its
// label, Enter and Space choose. Choosing closes every level and returns
// focus to the trigger; Escape closes one level; Tab closes the menu. A
// submenu opens with ArrowRight (ArrowLeft right to left), Enter, a press or
// a hover, and closes with the opposite arrow. Disabled items are reached
// but not chosen. While open, the page does not scroll. On pointer it floats
// beside the trigger; on touch the same menu is a bottom sheet with large
// items, and a submenu is a sheet above it with a back item.
import { computed, inject, onScopeDispose, provide, ref, shallowRef, useTemplateRef, watch } from 'vue'
import { useWeft } from '../adaptive/context.ts'
import Icon from '../basics/Icon.vue'
import { focus, isRtl } from '../headless/dom.ts'
import { useControllable } from '../headless/useControllable.ts'
import { useDismiss } from '../headless/useDismiss.ts'
import { useFloating, type Placement } from '../headless/useFloating.ts'
import { useId } from '../headless/useId.ts'
import { useRovingFocus } from '../headless/useRovingFocus.ts'
import { useScrollLock } from '../headless/useScrollLock.ts'
import { useTypeahead } from '../headless/useTypeahead.ts'
import { groupsOf, isChoice, menuLevel, roleOf, type MenuItem } from './menu.ts'
import { elementOf, isTouch, overlayRoot, useMounted, useOverlayContext, useTrigger, type Anchor } from './overlay.ts'
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
  }>(),
  { items: () => [], open: undefined, defaultOpen: false, placement: 'bottom-start' },
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
let start: 'first' | 'last' | 'none' = 'first'
const close = () => (open.value = false)
function closeAll(returnFocus: boolean) {
  if (parent) return parent.closeAll(returnFocus)
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

useDismiss(panel, {
  onDismiss(reason, event) {
    if (!parent || reason === 'escape' || sheet.value) return close()
    // A press in a level above closes this one; a press outside them all, every level.
    const target = event.target as Node
    if (parent.panels().some((p) => p?.contains(target))) close()
    else closeAll(false)
  },
  ignore: () => [wrap.value, backdrop.value, anchorEl()],
  returnTo: () => trigger.el.value ?? anchorEl(),
})
useScrollLock(computed(() => open.value && !parent))
watch(open, (on) => on && context.refresh(), { immediate: true, flush: 'sync' })

const roving = useRovingFocus(panel, { orientation: 'vertical' })
const itemEls = () => [...(panel.value?.querySelectorAll<HTMLElement>('[data-weft-item]') ?? [])]
const enabled = (el: HTMLElement) => el.getAttribute('aria-disabled') !== 'true'
const typeahead = useTypeahead(
  () => itemEls().map((el) => (enabled(el) ? (el.querySelector('.weft-menu__text')?.textContent ?? '') : '')),
  (i) => roving.focus(i),
)
function focusEdge(at: 'first' | 'last') {
  const list = itemEls().filter(enabled)
  focus(at === 'first' ? list[0] : list[list.length - 1])
}
watch(
  panel,
  (el) => {
    if (!el) return
    if (parent) {
      if (parent.focusFirst()) focusEdge('first')
    } else if (start !== 'none') focusEdge(start)
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

const itemOf = (event: Event) => (event.target as Element).closest<HTMLElement>('[data-weft-item]')

function choose(item: MenuItem, key: string, event: MouseEvent) {
  const el = itemOf(event)
  if (item.disabled) {
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

function onKeydown(event: KeyboardEvent) {
  if (event.defaultPrevented) return
  if (event.key === 'Tab') {
    event.preventDefault()
    closeAll(true)
    return
  }
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
  <Teleport v-if="mounted && open" :to="overlayRoot()">
    <div
      class="weft-overlay"
      :dir="context.attrs.value.dir"
      :data-weft-input="context.attrs.value['data-weft-input']"
      :data-weft-surface="context.attrs.value['data-weft-surface']"
      :data-weft-scheme="context.attrs.value['data-weft-scheme']"
    >
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
        :data-form="sheet ? 'sheet' : 'floating'"
        :data-side="side"
        :data-positioned="sheet || floating.isPositioned.value ? 'true' : undefined"
        :style="sheet ? undefined : floating.styles.value"
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
                :is="item.to && !item.disabled ? 'a' : 'div'"
                v-else
                class="weft-menu__item"
                data-weft-item
                :role="roleOf(item)"
                :href="item.to && !item.disabled ? item.to : undefined"
                :target="item.to && !item.disabled ? item.target : undefined"
                :aria-disabled="item.disabled ? 'true' : undefined"
                :aria-checked="isChoice(item) ? (item.checked ? 'true' : 'false') : undefined"
                :aria-haspopup="item.children ? 'menu' : undefined"
                :aria-expanded="item.children ? (sub?.key === `${g}-${i}` ? 'true' : 'false') : undefined"
                :data-tone="item.tone"
                @click="choose(item, `${g}-${i}`, $event)"
                @pointermove="onHover(item, `${g}-${i}`, $event)"
              >
                <span v-if="isChoice(item)" class="weft-menu__glyph weft-menu__indicator" aria-hidden="true">
                  <svg v-if="item.checked && item.type === 'checkbox'" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><path d="M20 6 9 17l-5-5" /></svg>
                  <svg v-else-if="item.checked" viewBox="0 0 24 24" fill="currentColor"><circle cx="12" cy="12" r="4.5" /></svg>
                </span>
                <span v-else-if="item.icon" class="weft-menu__glyph"><Icon :name="item.icon" /></span>
                <span class="weft-menu__text"><slot name="item" :item="item">{{ item.label }}</slot></span>
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
          @update:open="(on: boolean) => !on && closeSub()"
        >
          <template v-if="$slots.item" #item="{ item }"><slot name="item" :item="item" /></template>
        </DropdownMenu>
      </div>
    </div>
  </Teleport>
</template>
