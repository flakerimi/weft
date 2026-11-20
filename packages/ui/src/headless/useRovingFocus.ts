/**
 * useRovingFocus(container, { orientation, loop?, rtl?, items?, columns? })
 *
 * One tab stop for a group of items, moved with the arrow keys: the roving
 * tabindex of the WAI-ARIA APG (toolbars, tabs, radio groups, menus,
 * listboxes, grids). The current item has `tabindex="0"`, every other item
 * `tabindex="-1"`, so Tab enters and leaves the group in one step.
 *
 * Items are the container's descendants matching `items` (default
 * `[data-weft-item]`), in document order, that belong to this group: while
 * it runs, the container carries `data-weft-roving`, and an item whose
 * nearest `[data-weft-roving]` ancestor is another container belongs to that
 * nested group (a submenu, a tree's child group, a radio group in a toolbar).
 *
 * Indices (`active`, `focus(i)`, `setActive(i)`) count every item of the
 * group, disabled ones included, so they match the component's own list. An
 * item with `disabled` or `data-disabled` cannot take the tab stop and is
 * skipped by the arrow keys; one with only `aria-disabled="true"` stays
 * reachable, as APG asks for menu items. The tab stop starts on the first
 * item that is `aria-selected`, `aria-checked`, `aria-pressed` or
 * `aria-current`, else the first enabled item, and follows focus (a click on
 * an item moves it too). Items added or removed later are picked up.
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
 * Returns `{ active, focus, setActive, sync }`: the index of the current item
 * (reactive), a function to focus an item by index, one to move the tab stop
 * without moving focus (for a selection the parent sets), and one to
 * recompute the tab stops at once. Nothing happens on the server.
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
  setActive: (index: number) => void
  sync: () => void
}

const GROUP = 'data-weft-roving'
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

  /** This group's items, disabled ones included. */
  const all = () =>
    root
      ? [...root.querySelectorAll<HTMLElement>(selector)].filter((el) => el.parentElement?.closest(`[${GROUP}]`) === root)
      : []
  const enabled = (el: HTMLElement | undefined): el is HTMLElement =>
    !!el && !el.hasAttribute('disabled') && !el.hasAttribute('data-disabled')

  function sync() {
    const items = all()
    if (!current || !items.includes(current) || !enabled(current)) {
      current = items.find((el) => enabled(el) && el.matches(SELECTED)) ?? items.find(enabled) ?? null
    }
    for (const el of items) {
      const want = el === current ? '0' : '-1'
      if (el.getAttribute('tabindex') !== want) el.setAttribute('tabindex', want)
    }
    active.value = current ? items.indexOf(current) : -1
  }

  /** Make `el` the tab stop; focus it too unless `quiet`. Disabled items are refused. */
  function moveTo(el: HTMLElement | undefined, quiet = false) {
    if (!enabled(el)) return
    current = el
    sync()
    if (!quiet) focusEl(el)
  }

  function columns(items: HTMLElement[]): number {
    if (opts.columns) return Math.max(1, opts.columns())
    const top = items[0]?.getBoundingClientRect().top
    const n = items.filter((el) => el.getBoundingClientRect().top === top).length
    return Math.max(1, n)
  }

  /** The index to move to for the key, over all items, skipping disabled ones; null when the key is not ours. */
  function target(event: KeyboardEvent, at: number, items: HTMLElement[]): number | null {
    const count = items.length
    const loop = opts.loop !== false
    const ok = (i: number) => enabled(items[i])
    /** From `at`, the next enabled item `d` steps at a time; `wrap` wraps round, else it stops at the edge. */
    const seek = (from: number, d: number, wrap: boolean) => {
      for (let i = from + d, n = 0; n < count; i += d, n++) {
        if (i < 0 || i >= count) {
          if (!wrap) return at
          i = (i + count) % count
        }
        if (ok(i)) return i
      }
      return at
    }
    /** The first enabled item from `from` towards `d`, within [lo, hi]. */
    const firstIn = (from: number, d: number, lo: number, hi: number) => {
      for (let i = from; i >= lo && i <= hi; i += d) if (ok(i)) return i
      return at
    }
    const rtl = opts.rtl ? opts.rtl.value : isRtl(root)
    const forward = rtl ? 'ArrowLeft' : 'ArrowRight'
    const back = rtl ? 'ArrowRight' : 'ArrowLeft'
    const o = opts.orientation
    if (o === 'grid') {
      const cols = columns(items)
      const rowStart = at - (at % cols)
      const rowEnd = Math.min(rowStart + cols, count) - 1
      switch (event.key) {
        case forward: return seek(at, 1, loop)
        case back: return seek(at, -1, loop)
        case 'ArrowDown': return seek(at, cols, false)
        case 'ArrowUp': return seek(at, -cols, false)
        case 'Home': return event.ctrlKey ? firstIn(0, 1, 0, count - 1) : firstIn(rowStart, 1, rowStart, rowEnd)
        case 'End': return event.ctrlKey ? firstIn(count - 1, -1, 0, count - 1) : firstIn(rowEnd, -1, rowStart, rowEnd)
      }
      return null
    }
    const next = o === 'horizontal' ? forward : 'ArrowDown'
    const prev = o === 'horizontal' ? back : 'ArrowUp'
    switch (event.key) {
      case next: return seek(at, 1, loop)
      case prev: return seek(at, -1, loop)
      case 'Home': return firstIn(0, 1, 0, count - 1)
      case 'End': return firstIn(count - 1, -1, 0, count - 1)
    }
    return null
  }

  function onKeydown(event: KeyboardEvent) {
    if (event.defaultPrevented || event.altKey || event.metaKey) return
    if (event.ctrlKey && opts.orientation !== 'grid') return
    const items = all()
    const item = (event.target as Element | null)?.closest<HTMLElement>(selector)
    if (!item || !items.includes(item)) return
    const index = target(event, items.indexOf(item), items)
    if (index === null) return
    event.preventDefault()
    moveTo(items[index])
  }

  function onFocusin(event: FocusEvent) {
    const item = (event.target as Element | null)?.closest<HTMLElement>(selector)
    if (item && item !== current && enabled(item) && all().includes(item)) {
      current = item
      sync()
    }
  }

  function detach() {
    root?.removeEventListener('keydown', onKeydown)
    root?.removeEventListener('focusin', onFocusin)
    root?.removeAttribute(GROUP)
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
      node.setAttribute(GROUP, '')
      node.addEventListener('keydown', onKeydown)
      node.addEventListener('focusin', onFocusin)
      if (typeof MutationObserver !== 'undefined') {
        observer = new MutationObserver(sync)
        observer.observe(node, { childList: true, subtree: true, attributeFilter: ['disabled', 'data-disabled', GROUP] })
      }
      sync()
    },
    { immediate: true, flush: 'post' },
  )
  onScopeDispose(detach)

  return {
    active: readonly(active),
    focus: (index) => moveTo(all()[index]),
    setActive: (index) => moveTo(all()[index], true),
    sync,
  }
}
