// What a Table is told and what it works out: its columns, the query it
// asks a `fetch` function (the wire format's `page`, `perPage` and `sort`),
// the stable client-side sort, row keys and the words it says.

/** One column of a Table, and one label–value pair of each card when narrow. */
export interface TableColumn<T = Record<string, unknown>> {
  /** The column's key: the row property shown (unless `accessor` says otherwise), the sort key sent to `fetch`, and the name of its `<key>-cell` and `<key>-header` slots. */
  key: string
  /** The header's text; default the key. */
  label?: string
  /** Its header sorts the rows (ascending, descending, then not). */
  sortable?: boolean
  /** Where its text sits: `start` (default), `center` or `end` (for numbers). */
  align?: 'start' | 'center' | 'end'
  /** The column's width, any CSS length. */
  width?: string
  /** The cell's value, sorted by and shown; default `row[key]`. */
  accessor?: (row: T) => unknown
  /** The text shown for the value; default `String(value)`, nothing for null. */
  format?: (value: unknown, row: T) => string
  /** Its cells name their rows (`<th scope="row">`), and its value names a row's checkbox. */
  rowHeader?: boolean
}

/** What a Table asks its `fetch` for: the wire format's query parameters. */
export interface TableQuery {
  /** The page, from 1. */
  page: number
  perPage: number
  /** The column key, ascending (`name`), or descending with a minus (`-since`); absent when unsorted. */
  sort?: string
}

/** What `fetch` answers: the wire format's list (`page` and `perPage` may come back too, and are not needed). */
export interface TableResult<T> {
  items: T[]
  total: number
  page?: number
  perPage?: number
}

export type TableFetch<T> = (query: TableQuery) => Promise<TableResult<T>> | TableResult<T>

/** A row's key: what `v-model:selected` lists. */
export type RowKey = string | number

/** The words a Table says; give some or all to translate them. */
export interface TableLabels {
  /** The checkbox that chooses every row shown. */
  selectAll: string
  /** A row's checkbox, named by its row-header value (or its first column's). */
  selectRow: (name: string) => string
  /** Said with the text shown when there are no rows. */
  empty: string
  /** Said while the rows are loading (the skeleton rows are hidden from assistive technology). */
  loading: string
  /** Said when `fetch` failed. */
  error: string
  /** The button that tries `fetch` again. */
  retry: string
  /** Before the sort buttons of the card view. */
  sortBy: string
  ascending: string
  descending: string
}

export const TABLE_LABELS: TableLabels = {
  selectAll: 'Select all',
  selectRow: (name) => (name ? `Select ${name}` : 'Select row'),
  empty: 'Nothing to show.',
  loading: 'Loading',
  error: 'The rows could not be loaded.',
  retry: 'Try again',
  sortBy: 'Sort by',
  ascending: 'sorted ascending',
  descending: 'sorted descending',
}

/** A sort in the wire format, read: the column key and its direction. */
export interface Sort {
  key: string
  desc: boolean
}

/** `name` → ascending by name, `-since` → descending by since; empty → none. */
export function parseSort(sort: string | null | undefined): Sort | null {
  if (!sort) return null
  return sort.startsWith('-') ? { key: sort.slice(1), desc: true } : { key: sort, desc: false }
}

/** The next sort a press on column `key`'s header gives: ascending, then descending, then none. */
export function nextSort(current: string | null | undefined, key: string): string | null {
  const now = parseSort(current)
  if (!now || now.key !== key) return key
  return now.desc ? null : `-${key}`
}

/** A cell's value: its column's accessor, else the row's property. */
export function valueOf<T>(column: TableColumn<T>, row: T): unknown {
  return column.accessor ? column.accessor(row) : (row as Record<string, unknown>)?.[column.key]
}

/** The text a cell shows. */
export function textOf<T>(column: TableColumn<T>, row: T): string {
  const value = valueOf(column, row)
  if (column.format) return column.format(value, row)
  return value == null ? '' : String(value)
}

const collator = new Intl.Collator(undefined, { numeric: true, sensitivity: 'base' })

/**
 * Compare two cell values for an ascending sort: numbers and dates by
 * amount, booleans false first, anything else as text (numeric-aware, case
 * ignored). Empty values (null, undefined, '') sort last.
 */
const isEmpty = (v: unknown) => v == null || v === ''

export function compareValues(a: unknown, b: unknown): number {
  const emptyA = isEmpty(a)
  const emptyB = isEmpty(b)
  if (emptyA || emptyB) return emptyA === emptyB ? 0 : emptyA ? 1 : -1
  if (a instanceof Date && b instanceof Date) return a.getTime() - b.getTime()
  if (typeof a === 'number' && typeof b === 'number') return a - b
  if (typeof a === 'bigint' && typeof b === 'bigint') return a < b ? -1 : a > b ? 1 : 0
  if (typeof a === 'boolean' && typeof b === 'boolean') return Number(a) - Number(b)
  return collator.compare(String(a), String(b))
}

/**
 * `rows` sorted by `sort` (a new array; `rows` as given when unsorted or
 * the key names no column). Stable: rows that compare equal keep their
 * order, in both directions. Empty values stay last when descending too.
 */
export function sortRows<T>(rows: readonly T[], columns: readonly TableColumn<T>[], sort: string | null | undefined): T[] {
  const s = parseSort(sort)
  const column = s && columns.find((c) => c.key === s.key)
  if (!s || !column) return [...rows]
  const keyed = rows.map((row, index) => ({ row, index, value: valueOf(column, row) }))
  const direction = s.desc ? -1 : 1
  keyed.sort((x, y) => {
    const by = compareValues(x.value, y.value) * (isEmpty(x.value) || isEmpty(y.value) ? 1 : direction)
    return by || x.index - y.index
  })
  return keyed.map((k) => k.row)
}

/** A row's key: `rowKey` as a property name or a function; default the row's `id`, else `fallback`. */
export function keyOf<T>(row: T, rowKey: string | ((row: T) => RowKey) | undefined, fallback: RowKey): RowKey {
  if (typeof rowKey === 'function') return rowKey(row)
  const value = (row as Record<string, unknown>)?.[rowKey ?? 'id']
  return typeof value === 'string' || typeof value === 'number' ? value : fallback
}

/** A query as URL search parameters, as the wire format reads them: `page=2&perPage=20&sort=-since`. */
export function tableSearch(query: TableQuery): URLSearchParams {
  const params = new URLSearchParams({ page: String(query.page), perPage: String(query.perPage) })
  if (query.sort) params.set('sort', query.sort)
  return params
}

/**
 * The keys chosen after a Shift+click from `anchor` to `target` in `order`
 * (the rows as shown): every row between them, both included, takes `on`.
 * Without the anchor in `order`, only the target changes.
 */
export function selectRange(chosen: readonly RowKey[], order: readonly RowKey[], anchor: RowKey | null, target: RowKey, on: boolean): RowKey[] {
  const from = anchor === null ? -1 : order.indexOf(anchor)
  const to = order.indexOf(target)
  const span = from < 0 || to < 0 ? [target] : order.slice(Math.min(from, to), Math.max(from, to) + 1)
  const set = new Set(chosen)
  for (const key of span) on ? set.add(key) : set.delete(key)
  // Keep the order they were chosen in, then the new ones in row order.
  return [...chosen.filter((k) => set.has(k)), ...span.filter((k) => set.has(k) && !chosen.includes(k))]
}
