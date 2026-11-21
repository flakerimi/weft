/**
 * useTableData(options)
 *
 * A Table's rows: from `rows()`, sorted (stably) and sliced to the page on
 * the client; or from `fetch()`, asked for the page in the wire format
 * (`{ page, perPage, sort }`, sort absent when none) whenever one of those
 * values changes while `active()` (the Table passes "mounted", so the
 * server and the first render draw a busy table and agree). Only the
 * answer to the latest question is kept; a rejection is kept in `failed`
 * and passed to `onError`. `load()` asks again.
 *
 * Each row comes with its key: `rowKey()` (a property name or a function),
 * else its `id`, else its place (in `rows`, or `<page>:<index>` for a
 * fetched row, which is not stable across pages and sorts: with
 * `needKeys()` true, as when rows can be chosen, a fetched row without a
 * key of its own warns once in development).
 *
 * `perPage` resolves to the option, or 20 with `fetch`; `currentPage` keeps
 * the page inside the pages there are (on the client; with fetch the
 * server's total is not known before the answer). `busy` is `loading()`,
 * or a fetch under way, or no answer yet.
 */
import { computed, ref, shallowRef, watch, type ComputedRef, type Ref, type ShallowRef } from 'vue'
import { keyOf, sortRows, valueOf, type RowKey, type TableColumn, type TableFetch } from './table.ts'

export interface TableDataOptions<T> {
  rows: () => T[] | undefined
  fetch: () => TableFetch<T> | undefined
  columns: () => TableColumn<T>[]
  rowKey: () => string | ((row: T) => RowKey) | undefined
  sort: Readonly<Ref<string | null>>
  page: Readonly<Ref<number>>
  perPage: () => number | undefined
  loading: () => boolean
  active: () => boolean
  /** Rows can be chosen, so they need keys of their own. */
  needKeys?: () => boolean
  onError?: (error: unknown) => void
}

export interface TableEntry<T> {
  row: T
  key: RowKey
}

export interface TableData<T> {
  /** The rows shown, with their keys. */
  shown: ComputedRef<TableEntry<T>[]>
  total: ComputedRef<number>
  perPage: ComputedRef<number | undefined>
  pageCount: ComputedRef<number>
  currentPage: ComputedRef<number>
  busy: ComputedRef<boolean>
  failed: ShallowRef<unknown>
  load: () => Promise<void>
}

let warnedKeys = false

export function useTableData<T>(opts: TableDataOptions<T>): TableData<T> {
  const remote = shallowRef<{ items: T[]; total: number } | null>(null)
  const fetching = ref(false)
  const failed = shallowRef<unknown>(null)
  const perPage = computed(() => opts.perPage() ?? (opts.fetch() ? 20 : undefined))

  let asked = 0
  async function load() {
    const fetch = opts.fetch()
    if (!fetch) return
    const mine = ++asked
    fetching.value = true
    failed.value = null
    const query = { page: opts.page.value, perPage: perPage.value!, ...(opts.sort.value ? { sort: opts.sort.value } : {}) }
    try {
      const answer = await fetch(query)
      if (mine !== asked) return
      remote.value = { items: answer.items, total: answer.total }
    } catch (error) {
      if (mine !== asked) return
      failed.value = error
      opts.onError?.(error)
    } finally {
      if (mine === asked) fetching.value = false
    }
  }
  // Primitive sources: asking again only when a value of the query changes.
  watch(
    [opts.active, () => opts.page.value, () => perPage.value, () => opts.sort.value, opts.fetch],
    ([on]) => {
      if (on) void load()
    },
    { immediate: true },
  )

  const total = computed(() => (opts.fetch() ? (remote.value?.total ?? 0) : (opts.rows()?.length ?? 0)))
  const pageCount = computed(() => (perPage.value ? Math.ceil(total.value / Math.max(1, perPage.value)) : 1))
  const currentPage = computed(() =>
    opts.fetch() ? opts.page.value : Math.min(Math.max(1, opts.page.value), Math.max(1, pageCount.value)),
  )

  const shown = computed<TableEntry<T>[]>(() => {
    const rowKey = opts.rowKey()
    if (opts.fetch()) {
      const p = currentPage.value
      return (remote.value?.items ?? []).map((row, i) => {
        const fallback = `${p}:${i}`
        const key = keyOf(row, rowKey, fallback)
        if (key === fallback && opts.needKeys?.()) warnKeyless()
        return { row, key }
      })
    }
    const entries = (opts.rows() ?? []).map((row, i) => ({ row, key: keyOf(row, rowKey, i) }))
    const byEntry: TableColumn<TableEntry<T>>[] = opts.columns().map((c) => ({ key: c.key, accessor: (e) => valueOf(c, e.row) }))
    const sorted = sortRows(entries, byEntry, opts.sort.value)
    if (!perPage.value) return sorted
    const start = (currentPage.value - 1) * perPage.value
    return sorted.slice(start, start + perPage.value)
  })

  const busy = computed(() => opts.loading() || (!!opts.fetch() && (fetching.value || (!remote.value && !failed.value))))

  return { shown, total, perPage, pageCount, currentPage, busy, failed, load }
}

function warnKeyless() {
  if (!import.meta.env.DEV || warnedKeys) return
  warnedKeys = true
  console.warn(
    'Table: rows from fetch need a key of their own to be chosen; give each an `id`, or set `rowKey`, so v-model:selected keeps its rows across pages and sorts.',
  )
}
