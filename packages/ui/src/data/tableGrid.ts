/**
 * useTableGrid(table, options)
 *
 * The keyboard of a Table with `keyboard` (WAI-ARIA APG grid), on top of
 * useRovingFocus (which keeps the one tab stop). Cells are the table's
 * `[data-weft-item]` elements, each with its column in `data-col` and its
 * row in the `data-index` of its `<tr>` (none in the header: row -1). It
 * moves by data row and column, not over the rows rendered, so it works
 * when only some rows are:
 *
 * - Left and Right (mirrored right to left) move along the row and stop at
 *   its ends; Up and Down move a row; Home and End go to the row's ends;
 *   Ctrl+Home to the first cell of the header, Ctrl+End to the last cell of
 *   the last row.
 * - The target row is the `focusRow` (a windowed Table pins it, so it stays
 *   rendered when the pointer scrolls it away), scrolled into view (with
 *   `scrollTo(index)` when windowed, else `scrollIntoView` nearest), then
 *   focused once rendered.
 * - The controls inside a cell (a button a cell slot draws) leave the tab
 *   order (`tabindex="-1"`, the old value kept), the grid being one tab
 *   stop; rows drawn later are handled as they come. Enter or F2 on a cell
 *   gives its controls back and focuses the first; Escape in one, or focus
 *   leaving the cell, takes them out again. Keys pressed in a control are
 *   its own.
 *
 * Navigation keys are heard in the capture phase and their default
 * prevented, so useRovingFocus (which skips handled keys) does not move as
 * well. `table` null (no keyboard, or the card view) undoes all of it.
 */
import { nextTick, onScopeDispose, ref, watch, type Ref } from 'vue'
import { focus, isRtl, tabbables } from '../headless/dom.ts'

export interface TableGridOptions {
  /** How many data rows there are. */
  rows: () => number
  /** How many cells a row has. */
  columns: () => number
  /** Whether only the rows in view are rendered. */
  windowed: () => boolean
  /** Scroll the row at `index` into view (useVirtualList's scrollTo). */
  scrollTo: (index: number) => void
}

export interface TableGrid {
  /** The data row that holds focus (or last held it), -1 for the header or none. */
  focusRow: Readonly<Ref<number>>
}

const ITEM = '[data-weft-item]'
const CELL = 'td[data-weft-item], th[data-weft-item]'
const PARKED = 'data-weft-parked'

export function useTableGrid(table: Readonly<Ref<HTMLElement | null>>, opts: TableGridOptions): TableGrid {
  const focusRow = ref(-1)
  /** The cell whose controls are given back (Enter), or null. */
  let entered: HTMLElement | null = null
  let root: HTMLElement | null = null
  let observer: MutationObserver | undefined

  const rowOf = (el: Element) => {
    const index = el.closest('tr')?.getAttribute('data-index')
    return index == null ? -1 : Number(index)
  }
  const find = (row: number, col: number) =>
    root?.querySelector<HTMLElement>(row < 0 ? `thead [data-col="${col}"]` : `tr[data-index="${row}"] [data-col="${col}"]`) ?? null

  function park(cells: Iterable<HTMLElement>) {
    for (const cell of cells) {
      if (cell === entered) continue
      for (const el of tabbables(cell)) {
        el.setAttribute(PARKED, el.getAttribute('tabindex') ?? '')
        el.setAttribute('tabindex', '-1')
      }
    }
  }
  function unpark(within: Element) {
    for (const el of within.querySelectorAll<HTMLElement>(`[${PARKED}]`)) {
      const was = el.getAttribute(PARKED)!
      if (was === '') el.removeAttribute('tabindex')
      else el.setAttribute('tabindex', was)
      el.removeAttribute(PARKED)
    }
  }
  const parkAll = () => root && park(root.querySelectorAll<HTMLElement>(CELL))

  async function moveTo(row: number, col: number) {
    focusRow.value = row
    const windowed = opts.windowed()
    if (windowed) opts.scrollTo(Math.max(0, row))
    let el = find(row, col)
    if (!el && windowed) {
      await nextTick()
      el = find(row, col)
    }
    if (!el) return
    focus(el)
    if (!windowed) el.scrollIntoView?.({ block: 'nearest', inline: 'nearest' })
  }

  function onNavigate(event: KeyboardEvent) {
    if (event.defaultPrevented || event.altKey || event.metaKey) return
    const item = (event.target as Element).closest<HTMLElement>(ITEM)
    if (!item || item !== event.target || !root?.contains(item)) return
    const last = opts.columns() - 1
    const count = opts.rows()
    let row = rowOf(item)
    let col = Number(item.dataset.col ?? 0)
    const rtl = isRtl(root)
    const ctrl = event.ctrlKey
    switch (event.key) {
      case rtl ? 'ArrowLeft' : 'ArrowRight':
        col = Math.min(col + 1, last)
        break
      case rtl ? 'ArrowRight' : 'ArrowLeft':
        col = Math.max(col - 1, 0)
        break
      case 'ArrowDown':
        row = Math.min(row + 1, count - 1)
        break
      case 'ArrowUp':
        row = Math.max(row - 1, -1)
        break
      case 'Home':
        if (ctrl) row = -1
        col = 0
        break
      case 'End':
        if (ctrl) row = count - 1
        col = last
        break
      default:
        return
    }
    event.preventDefault()
    void moveTo(row, col)
  }

  function onKeydown(event: KeyboardEvent) {
    if (event.defaultPrevented) return
    const target = event.target as HTMLElement
    if ((event.key === 'Enter' || event.key === 'F2') && target.matches(CELL)) {
      unpark(target)
      const inner = tabbables(target)[0]
      if (!inner) return
      event.preventDefault()
      entered = target
      focus(inner)
    } else if (event.key === 'Escape') {
      const cell = target.closest<HTMLElement>(CELL)
      if (!cell || cell === target) return
      event.preventDefault()
      entered = null
      focus(cell)
      park([cell])
    }
  }

  function onFocusin(event: FocusEvent) {
    const item = (event.target as Element).closest<HTMLElement>(ITEM)
    if (item && root?.contains(item)) focusRow.value = rowOf(item)
  }

  function onFocusout(event: FocusEvent) {
    const cell = entered
    if (!cell || cell.contains(event.relatedTarget as Node | null)) return
    entered = null
    park([cell])
  }

  function detach() {
    observer?.disconnect()
    observer = undefined
    if (!root) return
    root.removeEventListener('keydown', onNavigate, true)
    root.removeEventListener('keydown', onKeydown)
    root.removeEventListener('focusin', onFocusin)
    root.removeEventListener('focusout', onFocusout)
    unpark(root)
    entered = null
    root = null
  }

  watch(
    table,
    (el) => {
      detach()
      if (!el) return
      root = el
      el.addEventListener('keydown', onNavigate, true)
      el.addEventListener('keydown', onKeydown)
      el.addEventListener('focusin', onFocusin)
      el.addEventListener('focusout', onFocusout)
      parkAll()
      if (typeof MutationObserver !== 'undefined') {
        observer = new MutationObserver(parkAll)
        observer.observe(el, { childList: true, subtree: true })
      }
    },
    { immediate: true, flush: 'post' },
  )
  onScopeDispose(detach)

  return { focusRow }
}
