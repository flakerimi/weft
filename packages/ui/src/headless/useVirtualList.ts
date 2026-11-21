/**
 * useVirtualList(scroller, { count, itemHeight, overscan?, enabled?, pinned?, viewport? })
 *
 * Render only the rows of a long list that are in view: a listbox of 10,000
 * options draws a few dozen. Every row has the same height, `itemHeight()`
 * px (a fixed height, so a row's place is `index × height` and nothing is
 * measured). The component draws a spacer `total` px tall inside `scroller`
 * (the element that scrolls) and each rendered row at `offset(index)`.
 *
 * - `indices`: the rows to render, ascending: those in view, `overscan`
 *   more on each side (default 6), and the `pinned()` index wherever it is.
 *   Pin the active option: a listbox's `aria-activedescendant` must name an
 *   element that exists, and keyboard focus must not vanish when the pointer
 *   scrolls it away.
 * - `scrollTo(index)`: scroll the least that shows the row (to its top
 *   when it is above the view, to its bottom when below; not at all when it
 *   shows), and render the new window at once, before the scroll event.
 * - `enabled()` false renders every row (`virtual` says which); then
 *   `scrollTo` scrolls the scroller (and not the page) to show its
 *   `[data-index="<index>"]` element, so rows carry their index. A component turns it on past a count
 *   (Weft's selection components past 200 options).
 * - The view's height is the scroller's `clientHeight`, kept current by its
 *   scroll events and a ResizeObserver; before layout, and on the server,
 *   it is `viewport` px (default 320), so the server's first rows and the
 *   browser's first render agree and hydrate.
 * - When the count shrinks under the scroll position the window moves back
 *   inside the list.
 *
 * Nothing touches the DOM on the server; listeners go when the scope ends.
 */
import { computed, onScopeDispose, ref, watch, type ComputedRef, type Ref } from 'vue'

export interface VirtualListOptions {
  count: () => number
  itemHeight: () => number
  overscan?: number
  enabled?: () => boolean
  pinned?: () => number
  viewport?: number
}

export interface VirtualList {
  /** Whether it renders a window of the rows (false: every row). */
  virtual: ComputedRef<boolean>
  indices: ComputedRef<number[]>
  /** The height of every row together, px. */
  total: ComputedRef<number>
  /** The top of row `index`, px. */
  offset: (index: number) => number
  scrollTo: (index: number) => void
}

export function useVirtualList(scroller: Readonly<Ref<HTMLElement | null | undefined>>, opts: VirtualListOptions): VirtualList {
  const overscan = opts.overscan ?? 6
  const top = ref(0)
  const height = ref(0)
  const virtual = computed(() => opts.enabled?.() !== false)
  const rowHeight = () => Math.max(1, opts.itemHeight())
  const view = () => height.value || opts.viewport || 320
  const total = computed(() => opts.count() * rowHeight())
  /** The scroll position, kept inside the list. */
  const at = () => Math.max(0, Math.min(top.value, total.value - view()))

  const indices = computed(() => {
    const count = opts.count()
    if (!virtual.value) return Array.from({ length: count }, (_, i) => i)
    const h = rowHeight()
    const start = Math.max(0, Math.floor(at() / h) - overscan)
    const end = Math.min(count, Math.ceil((at() + view()) / h) + overscan)
    const out = Array.from({ length: Math.max(0, end - start) }, (_, i) => start + i)
    const pin = opts.pinned?.() ?? -1
    if (pin >= 0 && pin < count && (pin < start || pin >= end)) {
      if (pin < start) out.unshift(pin)
      else out.push(pin)
    }
    return out
  })

  const read = () => {
    const el = scroller.value
    if (!el) return
    top.value = el.scrollTop
    height.value = el.clientHeight
  }
  let observer: ResizeObserver | undefined
  let on: HTMLElement | null = null
  const detach = () => {
    on?.removeEventListener('scroll', read)
    observer?.disconnect()
    observer = undefined
    on = null
  }
  watch(
    scroller,
    (el) => {
      detach()
      if (!el || typeof window === 'undefined') return
      on = el
      el.addEventListener('scroll', read, { passive: true })
      if (typeof ResizeObserver !== 'undefined') {
        observer = new ResizeObserver(read)
        observer.observe(el)
      }
      read()
    },
    { immediate: true, flush: 'post' },
  )
  onScopeDispose(detach)

  function scrollTo(index: number) {
    const el = scroller.value
    if (index < 0) return
    if (!virtual.value) {
      // Every row is rendered: scroll the scroller (only, not the page) to show it.
      const row = el?.querySelector<HTMLElement>(`[data-index="${index}"]`)
      if (!el || !row) return
      const r = row.getBoundingClientRect()
      const s = el.getBoundingClientRect()
      if (r.top < s.top) el.scrollTop -= s.top - r.top
      else if (r.bottom > s.bottom) el.scrollTop += r.bottom - s.bottom
      return
    }
    const h = rowHeight()
    const rowTop = index * h
    const now = at()
    let next = now
    if (rowTop < now) next = rowTop
    else if (rowTop + h > now + view()) next = rowTop + h - view()
    if (next === now) return
    top.value = next
    if (el) el.scrollTop = next
  }

  return { virtual, indices, total, offset: (index) => index * rowHeight(), scrollTo }
}
