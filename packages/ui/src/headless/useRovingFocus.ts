/**
 * useRovingFocus(container, { orientation, loop?, rtl?, items?, columns? })
 *
 * One tab stop for a group of items, moved with the arrow keys: the roving
 * tabindex of the WAI-ARIA APG (toolbars, tabs, radio groups, menus,
 * listboxes, grids). The current item has `tabindex="0"`, every other item
 * `tabindex="-1"`, so Tab enters and leaves the group in one step.
 *
 * Items are the container's descendants matching `items` (default
 * `[data-weft-item]`), in document order. An item with `disabled` or
 * `data-disabled` is skipped; one with only `aria-disabled="true"` stays
 * reachable, as APG asks for menu items. The tab stop starts on the first item
 * that is `aria-selected`, `aria-checked`, `aria-pressed` or `aria-current`,
 * else the first item, and follows focus (a click on an item moves it too).
 * Items added or removed later are picked up.
 *
 * Keys (handled ones have their default prevented):
 * - `horizontal`: ArrowRight next, ArrowLeft previous; swapped when `rtl`.
 * - `vertical`: ArrowDown next, ArrowUp previous.
 * - `grid`: Left and Right move by one (mirrored when `rtl`), Up and Down by a
 *   row of `columns()` items (default: as many as share the first item's top
 *   edge), Home and End to the row's ends, Ctrl+Home and Ctrl+End to the first
 *   and last item. Up and Down stop at the edges.
 * - Home and End go to the first and last item (horizontal and vertical).
 * - `loop` (default true) wraps at the ends; false stops there.
 *
 * `rtl` (a ref) overrides the direction; without it, the direction is read
 * from the container (`dir="rtl"` on any ancestor) at each key press.
 *
 * Returns `{ active, focus, sync }`: the index of the current item
 * (reactive), a function to focus an item by index, and one to recompute the
 * tab stops at once. Nothing happens on the server.
 */
import { onScopeDispose, readonly, ref, watch, type Ref } from 'vue'
import { focus as focusEl, isRtl } from './dom.ts'

export interface RovingFocusOptions {
  orientation: 'horizontal' | 'vertical' | 'grid'
  loop?: boolean
  rtl?: Readonly<Ref<boolean>>
  /** CSS selector for the items; default `[data-weft-item]`. */
  items?: string
  /** Items per row in a grid. */
  columns?: () => number
}

export interface RovingFocus {
  active: Readonly<Ref<number>>
  focus: (index: number) => void
  sync: () => void
}

const SELECTED = '[aria-selected="true"],[aria-checked="true"],[aria-pressed="true"],[aria-current]:not([aria-current="false"])'

export function useRovingFocus(
  container: Readonly<Ref<HTMLElement | null | undefined>>,
  opts: RovingFocusOptions,
): RovingFocus {
  const selector = opts.items ?? '[data-weft-item]'
  const active = ref(-1)
  let current: HTMLElement | null = null
  let root: HTMLElement | null = null
  let observer: MutationObserver | undefined

  const all = () => (root ? [...root.querySelectorAll<HTMLElement>(selector)] : [])
  const enabled = (el: HTMLElement) => !el.hasAttribute('disabled') && !el.hasAttribute('data-disabled')
  const list = () => all().filter(enabled)

  function sync() {
    const items = list()
    if (!current || !items.includes(current)) {
      current = items.find((el) => el.matches(SELECTED)) ?? items[0] ?? null
    }
    for (const el of all()) {
      const want = el === current ? '0' : '-1'
      if (el.getAttribute('tabindex') !== want) el.setAttribute('tabindex', want)
    }
    active.value = current ? items.indexOf(current) : -1
  }

  function moveTo(el: HTMLElement | undefined) {
    if (!el) return
    current = el
    sync()
    focusEl(el)
  }

  function columns(items: HTMLElement[]): number {
    if (opts.columns) return Math.max(1, opts.columns())
    const top = items[0]?.getBoundingClientRect().top
    const n = items.filter((el) => el.getBoundingClientRect().top === top).length
    return Math.max(1, n)
  }

  /** The index to move to for `key`, or null when the key is not ours. */
  function target(event: KeyboardEvent, at: number, count: number, items: HTMLElement[]): number | null {
    const loop = opts.loop !== false
    const step = (d: number) => {
      const next = at + d
      if (next >= 0 && next < count) return next
      return loop ? (next + count) % count : at
    }
    const rtl = opts.rtl ? opts.rtl.value : isRtl(root)
    const forward = rtl ? 'ArrowLeft' : 'ArrowRight'
    const back = rtl ? 'ArrowRight' : 'ArrowLeft'
    const o = opts.orientation
    if (o === 'grid') {
      const cols = columns(items)
      const rowStart = at - (at % cols)
      switch (event.key) {
        case forward: return step(1)
        case back: return step(-1)
        case 'ArrowDown': return at + cols < count ? at + cols : at
        case 'ArrowUp': return at - cols >= 0 ? at - cols : at
        case 'Home': return event.ctrlKey ? 0 : rowStart
        case 'End': return event.ctrlKey ? count - 1 : Math.min(rowStart + cols, count) - 1
      }
      return null
    }
    const next = o === 'horizontal' ? forward : 'ArrowDown'
    const prev = o === 'horizontal' ? back : 'ArrowUp'
    switch (event.key) {
      case next: return step(1)
      case prev: return step(-1)
      case 'Home': return 0
      case 'End': return count - 1
    }
    return null
  }

  function onKeydown(event: KeyboardEvent) {
    if (event.defaultPrevented || event.altKey || event.metaKey) return
    if (event.ctrlKey && opts.orientation !== 'grid') return
    const items = list()
    const item = (event.target as Element | null)?.closest<HTMLElement>(selector)
    if (!item || !items.includes(item)) return
    const index = target(event, items.indexOf(item), items.length, items)
    if (index === null) return
    event.preventDefault()
    moveTo(items[index])
  }

  function onFocusin(event: FocusEvent) {
    const item = (event.target as Element | null)?.closest<HTMLElement>(selector)
    if (item && item !== current && list().includes(item)) {
      current = item
      sync()
    }
  }

  function detach() {
    root?.removeEventListener('keydown', onKeydown)
    root?.removeEventListener('focusin', onFocusin)
    observer?.disconnect()
    observer = undefined
    root = null
  }

  watch(
    container,
    (node) => {
      detach()
      if (!node) return
      root = node
      node.addEventListener('keydown', onKeydown)
      node.addEventListener('focusin', onFocusin)
      if (typeof MutationObserver !== 'undefined') {
        observer = new MutationObserver(sync)
        observer.observe(node, { childList: true, subtree: true, attributeFilter: ['disabled', 'data-disabled'] })
      }
      sync()
    },
    { immediate: true, flush: 'post' },
  )
  onScopeDispose(detach)

  return {
    active: readonly(active),
    focus: (index) => moveTo(list()[index]),
    sync,
  }
}
