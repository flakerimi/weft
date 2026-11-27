/** What Pagination draws between its controls: page numbers, and gaps. */
export type PageEntry = number | 'ellipsis'

/**
 * The pages to show for `page` of `count`, with `siblings` on each side.
 *
 * - Without `edges`: a window of `2 × siblings + 1` pages around the current
 *   one, shifted to stay inside 1…count.
 * - With `edges`: the first and last pages always, the window between them,
 *   and an ellipsis for each gap of two pages or more. The length stays at
 *   `2 × siblings + 5` (or `count`, when smaller), so the buttons do not jump
 *   about as the page changes.
 */
export function pageRange(page: number, count: number, siblings: number, edges: boolean): PageEntry[] {
  if (count < 1) return []
  const s = Math.max(0, Math.floor(siblings))
  const at = Math.min(Math.max(1, page), count)
  const span = (from: number, to: number) => Array.from({ length: to - from + 1 }, (_, i) => from + i)
  if (!edges) {
    const size = Math.min(count, 2 * s + 1)
    const start = Math.min(Math.max(1, at - s), count - size + 1)
    return span(start, start + size - 1)
  }
  const slots = 2 * s + 5
  if (count <= slots) return span(1, count)
  const left = Math.max(at - s, 1)
  const right = Math.min(at + s, count)
  const leftGap = left > 3
  const rightGap = right < count - 2
  const edge = 2 * s + 3
  if (!leftGap) return [...span(1, edge), 'ellipsis', count]
  if (!rightGap) return [1, 'ellipsis', ...span(count - edge + 1, count)]
  return [1, 'ellipsis', ...span(left, right), 'ellipsis', count]
}

/** One place in what Pagination draws before mounting: an entry, and the form it belongs to only, if one. */
export interface PageSlot {
  entry: PageEntry
  /** Drawn only in the narrow form (`narrow`) or only in the wide one (`wide`); in both when unset. */
  only?: 'narrow' | 'wide'
}

/**
 * The narrow form's pages (at most one sibling) and the wide form's
 * (`siblings`) in one list, each entry marked with the form it belongs to
 * only, so the server can draw both and CSS show one before mounting:
 * leaving out the `wide` entries gives `pageRange(page, count,
 * min(siblings, 1), edges)`, and leaving out the `narrow` ones gives
 * `pageRange(page, count, siblings, edges)`. An ellipsis goes just before the
 * page that ends its gap.
 */
export function pageRangeBoth(page: number, count: number, siblings: number, edges: boolean): PageSlot[] {
  const wide = pageRange(page, count, siblings, edges)
  const narrow = pageRange(page, count, Math.min(siblings, 1), edges)
  const numbers = (list: PageEntry[]) => new Set(list.filter((e): e is number => e !== 'ellipsis'))
  // The pages an ellipsis comes just before (an ellipsis is never last).
  const gaps = (list: PageEntry[]) => new Set(list.flatMap((e, i) => (e === 'ellipsis' && typeof list[i + 1] === 'number' ? [list[i + 1] as number] : [])))
  const inNarrow = numbers(narrow)
  const inWide = numbers(wide)
  const gapNarrow = gaps(narrow)
  const gapWide = gaps(wide)
  const only = (n: boolean, w: boolean): PageSlot['only'] => (n && w ? undefined : n ? 'narrow' : 'wide')
  const out: PageSlot[] = []
  for (const n of [...new Set([...inNarrow, ...inWide])].sort((a, b) => a - b)) {
    if (gapNarrow.has(n) || gapWide.has(n)) out.push({ entry: 'ellipsis', only: only(gapNarrow.has(n), gapWide.has(n)) })
    out.push({ entry: n, only: only(inNarrow.has(n), inWide.has(n)) })
  }
  return out
}

/** The words Pagination says, for assistive technology; give some or all to translate them. */
export interface PaginationLabels {
  /** The landmark's name. */
  nav: string
  first: string
  previous: string
  next: string
  last: string
  /** A page button's name. */
  page: (page: number) => string
}

export const PAGINATION_LABELS: PaginationLabels = {
  nav: 'Pagination',
  first: 'First page',
  previous: 'Previous page',
  next: 'Next page',
  last: 'Last page',
  page: (n) => `Page ${n}`,
}
