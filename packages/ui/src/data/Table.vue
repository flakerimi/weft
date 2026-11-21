<script setup vapor lang="ts" generic="T extends object">
// Rows of data under column headers: a native <table> with a caption,
// th scope="col" headers and, for a `rowHeader` column, th scope="row" cells.
// A sortable header holds a button that sorts ascending, descending, then not
// (stable, on the client; through `fetch`, on the server), and says so with
// aria-sort. `selectable` adds a checkbox per row, named by the row, under a
// header checkbox that shows none, some (indeterminate) or all of the rows
// shown, with Shift+click choosing a range. The header stays in view while
// the rows scroll inside the table. While loading it is aria-busy, drawing
// skeleton rows until it has rows; with no rows it shows the `empty` slot.
// `perPage` pages the rows on the client with a Pagination; `fetch(query)`
// asks a server for each page instead, in the wire format's `page`,
// `perPage` and `sort`. Past 200 rows on one page it renders only the rows
// in view, each a fixed height. When its own box is narrow, each row is a
// card (a list item of label–value pairs) from the same columns, with the
// same selection and sorting. With `keyboard` it is an APG grid: one tab
// stop, arrows between cells, Enter into a cell's control, Escape back out.
import { computed, onMounted, ref, shallowRef, useTemplateRef, watch } from 'vue'
import Skeleton from '../basics/Skeleton.vue'
import Button from '../basics/Button.vue'
import Checkbox from '../forms/Checkbox.vue'
import Pagination from '../navigation/Pagination.vue'
import { useWeft } from '../adaptive/context.ts'
import { useWidth } from '../adaptive/useWidth.ts'
import { focus, tabbables } from '../headless/dom.ts'
import { useControllable } from '../headless/useControllable.ts'
import { useRovingFocus } from '../headless/useRovingFocus.ts'
import { useVirtualList } from '../headless/useVirtualList.ts'
import type { Size } from '../types.ts'
import {
  keyOf,
  nextSort,
  parseSort,
  selectRange,
  sortRows,
  TABLE_LABELS,
  textOf,
  valueOf,
  type RowKey,
  type TableColumn,
  type TableFetch,
  type TableLabels,
  type TableQuery,
} from './table.ts'
import '../styles/base.css'
import '../styles/table.css'

const props = withDefaults(
  defineProps<{
    /** The columns: `{ key, label, sortable, align, width, accessor, format, rowHeader }`. */
    columns: TableColumn<T>[]
    /** The rows, sorted and paged on the client. */
    rows?: T[]
    /** Asks for a page of rows instead (`{ page, perPage, sort }` → `{ items, total }`): sorting and paging happen on the server. */
    fetch?: TableFetch<T>
    /** A row's key, for selection and rendering: a property name or a function; default the row's `id`, else its place. */
    rowKey?: string | ((row: T) => RowKey)
    /** The sort, in the wire format (`name`, `-since`; null for none), bound with `v-model:sort`. */
    sort?: string | null
    /** The sort when `v-model:sort` is not bound. */
    defaultSort?: string | null
    /** A checkbox on each row, and one in the header for every row shown. */
    selectable?: boolean
    /** The keys of the chosen rows, bound with `v-model:selected`. */
    selected?: RowKey[]
    /** The rows chosen when `v-model:selected` is not bound. */
    defaultSelected?: RowKey[]
    /** Rows per page: the table pages with a Pagination. With `fetch`, default 20. */
    perPage?: number
    /** The page, from 1, bound with `v-model:page`. */
    page?: number
    /** The page shown first when `v-model:page` is not bound. */
    defaultPage?: number
    /** Busy: skeleton rows until there are rows, and aria-busy. */
    loading?: boolean
    /** The text shown when there are no rows; the `empty` slot replaces it. */
    empty?: string
    /** The table's caption, which names it (and the list of cards). */
    caption?: string
    /** Draw the caption for assistive technology only. */
    hideCaption?: boolean
    /** The header stays in view while the rows scroll inside the table. */
    sticky?: boolean
    /** The most the table may be tall before its rows scroll, any CSS length; default 30rem once windowed past 200 rows. */
    height?: string
    /** px of every row once windowed past 200 rows; default from the size (44 at least on touch). */
    rowHeight?: number
    /** APG grid: one tab stop, arrow keys between cells. Off: a plain table, every control its own tab stop. */
    keyboard?: boolean
    /** `sm`, `md` or `lg`: the rows' height. */
    size?: Size
    /** The words it says; give some or all to translate them. */
    labels?: Partial<TableLabels>
  }>(),
  {
    rows: undefined,
    fetch: undefined,
    sort: undefined,
    defaultSort: null,
    selected: undefined,
    page: undefined,
    defaultPage: 1,
    sticky: true,
    size: 'md',
  },
)

const emit = defineEmits<{
  /** The sort changed: a header was pressed. */
  'update:sort': [sort: string | null]
  /** The chosen rows changed. */
  'update:selected': [keys: RowKey[]]
  /** Another page was chosen, or the sort went back to page 1. */
  'update:page': [page: number]
  /** `fetch` failed. */
  error: [error: unknown]
}>()

defineSlots<{
  /** Shown when there are no rows, in place of `empty`. */
  empty?: () => unknown
  /** A column's cells: `#name-cell="{ row, value, index, column }"`. */
  [cell: `${string}-cell`]: ((props: { row: T; value: unknown; index: number; column: TableColumn<T> }) => unknown) | undefined
  /** A column's header: `#name-header="{ column }"`. */
  [header: `${string}-header`]: ((props: { column: TableColumn<T> }) => unknown) | undefined
}>()

const weft = useWeft()
const t = computed<TableLabels>(() => ({ ...TABLE_LABELS, ...props.labels }))
const root = useTemplateRef<HTMLElement>('root')
const scroller = useTemplateRef<HTMLElement>('scroller')
const table = useTemplateRef<HTMLElement>('table')
const width = useWidth(root)
const cards = computed(() => width.value === 'narrow')

const sort = useControllable(props, 'sort', emit, () => props.defaultSort)
const selected = useControllable(props, 'selected', emit, () => props.defaultSelected ?? [])
const page = useControllable(props, 'page', emit, () => props.defaultPage)
watch(sort, () => (page.value = 1))

// Through fetch: the server's answer to the latest query.
const remote = shallowRef<{ items: T[]; total: number } | null>(null)
const fetching = ref(false)
const failed = shallowRef<unknown>(null)
const mounted = ref(false)
onMounted(() => (mounted.value = true))
const perPage = computed(() => props.perPage ?? (props.fetch ? 20 : undefined))
const query = computed<TableQuery>(() => {
  const q: TableQuery = { page: page.value, perPage: perPage.value! }
  if (sort.value) q.sort = sort.value
  return q
})
let asked = 0
async function load() {
  const fetch = props.fetch
  if (!fetch) return
  const mine = ++asked
  fetching.value = true
  failed.value = null
  try {
    const answer = await fetch(query.value)
    if (mine !== asked) return
    remote.value = { items: answer.items, total: answer.total }
  } catch (error) {
    if (mine !== asked) return
    failed.value = error
    emit('error', error)
  } finally {
    if (mine === asked) fetching.value = false
  }
}
watch([mounted, query, () => props.fetch], ([on]) => on && load())

interface Entry {
  row: T
  key: RowKey
}
const total = computed(() => (props.fetch ? (remote.value?.total ?? 0) : (props.rows?.length ?? 0)))
const pageCount = computed(() => (perPage.value ? Math.ceil(total.value / Math.max(1, perPage.value)) : 1))
const currentPage = computed(() => (props.fetch ? page.value : Math.min(Math.max(1, page.value), Math.max(1, pageCount.value))))
/** The rows shown: sorted and sliced to the page on the client, or the server's page. */
const shown = computed<Entry[]>(() => {
  if (props.fetch) {
    const p = currentPage.value
    return (remote.value?.items ?? []).map((row, i) => ({ row, key: keyOf(row, props.rowKey, `${p}:${i}`) }))
  }
  const entries = (props.rows ?? []).map((row, i) => ({ row, key: keyOf(row, props.rowKey, i) }))
  const byEntry: TableColumn<Entry>[] = props.columns.map((c) => ({ key: c.key, accessor: (e) => valueOf(c, e.row) }))
  const sorted = sortRows(entries, byEntry, sort.value)
  if (!perPage.value) return sorted
  const start = (currentPage.value - 1) * perPage.value
  return sorted.slice(start, start + perPage.value)
})

const busy = computed(() => !!props.loading || (!!props.fetch && (fetching.value || (!remote.value && !failed.value))))
const skeletons = computed(() => (busy.value && shown.value.length === 0 ? Array.from({ length: perPage.value ?? 5 }, (_, i) => i) : []))
const isEmpty = computed(() => !busy.value && !failed.value && shown.value.length === 0)
const span = computed(() => props.columns.length + (props.selectable ? 1 : 0))
const current = computed(() => parseSort(sort.value))
const ariaSort = (column: TableColumn<T>) =>
  current.value?.key === column.key ? (current.value.desc ? 'descending' : 'ascending') : undefined
const label = (column: TableColumn<T>) => column.label ?? column.key

function toggleSort(column: TableColumn<T>) {
  if (!column.sortable) return
  sort.value = nextSort(sort.value, column.key)
}

// Selection: over the rows shown; a Shift+click chooses from the last row clicked.
const chosen = computed(() => new Set(selected.value))
const shownKeys = computed(() => shown.value.map((e) => e.key))
const allState = computed<boolean | 'indeterminate'>(() => {
  const n = shownKeys.value.filter((k) => chosen.value.has(k)).length
  return n === 0 ? false : n === shownKeys.value.length ? true : 'indeterminate'
})
let anchor: RowKey | null = null
let shift = false
function toggleAll() {
  const keys = shownKeys.value
  if (allState.value === true) selected.value = selected.value.filter((k) => !keys.includes(k))
  else selected.value = [...selected.value, ...keys.filter((k) => !chosen.value.has(k))]
}
function toggleRow(key: RowKey, on: boolean | 'indeterminate') {
  const want = on === true
  if (shift && anchor !== null) selected.value = selectRange(selected.value, shownKeys.value, anchor, key, want)
  else selected.value = want ? (chosen.value.has(key) ? selected.value : [...selected.value, key]) : selected.value.filter((k) => k !== key)
  anchor = key
  shift = false
}
/** Heard before the checkbox's change: whether Shift was held. */
const noteShift = (event: MouseEvent) => (shift = event.shiftKey)
const nameCol = computed(() => props.columns.find((c) => c.rowHeader) ?? props.columns[0])
const rowName = (row: T) => (nameCol.value ? textOf(nameCol.value, row) : '')

// Windowed past 200 rows: fixed-height rows between two spacer rows.
const ROW = { sm: 36, md: 44, lg: 52 } as const
const rowPx = computed(() => props.rowHeight ?? (weft.input.value === 'touch' ? Math.max(44, ROW[props.size]) : ROW[props.size]))
const list = useVirtualList(scroller, {
  count: () => shown.value.length,
  itemHeight: () => rowPx.value,
  enabled: () => !cards.value && shown.value.length > 200,
})
const windowed = computed(() => list.virtual.value)
const visible = computed(() => list.indices.value)
const before = computed(() => (windowed.value && visible.value.length ? list.offset(visible.value[0]!) : 0))
const after = computed(() => (windowed.value && visible.value.length ? list.total.value - list.offset(visible.value[visible.value.length - 1]! + 1) : 0))
const maxHeight = computed(() => props.height ?? (windowed.value ? '30rem' : undefined))

// The grid: one tab stop over the cells (a cell's own control, a sort button or a checkbox, stands for it).
const grid = computed(() => !!props.keyboard && !cards.value)
const gridEl = computed(() => (grid.value ? table.value : null))
useRovingFocus(gridEl, { orientation: 'grid', columns: () => span.value, inner: 'ignore' })
const item = computed(() => (grid.value ? '' : undefined))
const CELL = 'td[data-weft-item], th[data-weft-item]'
function onGridKeydown(event: KeyboardEvent) {
  if (!grid.value || event.defaultPrevented) return
  const target = event.target as HTMLElement
  if ((event.key === 'Enter' || event.key === 'F2') && target.matches(CELL)) {
    const inner = tabbables(target)[0]
    if (inner) {
      event.preventDefault()
      focus(inner)
    }
  } else if (event.key === 'Escape') {
    const cell = target.closest<HTMLElement>(CELL)
    if (cell && cell !== target) {
      event.preventDefault()
      focus(cell)
    }
  }
}

defineExpose({
  /** Ask `fetch` again for the page shown. */
  refresh: () => load(),
})
</script>

<template>
  <div
    ref="root"
    class="weft-table"
    :data-size="size"
    :data-layout="cards ? 'cards' : 'table'"
    :data-sticky="sticky ? 'true' : undefined"
    :data-virtual="windowed ? 'true' : undefined"
    :data-loading="busy ? 'true' : undefined"
    :style="{ '--weft-table-row': `${rowPx}px` }"
  >
    <span class="weft-visually-hidden" role="status">{{ busy ? t.loading : '' }}</span>
    <div v-if="!cards" ref="scroller" class="weft-table__scroller" :style="maxHeight ? { maxBlockSize: maxHeight } : undefined">
      <table
        ref="table"
        class="weft-table__table"
        :role="grid ? 'grid' : undefined"
        :aria-busy="busy ? 'true' : undefined"
        :aria-rowcount="windowed ? shown.length + 1 : undefined"
        @keydown="onGridKeydown"
      >
        <caption v-if="caption" :class="hideCaption ? 'weft-visually-hidden' : 'weft-table__caption'">{{ caption }}</caption>
        <thead>
          <tr :aria-rowindex="windowed ? 1 : undefined">
            <th v-if="selectable" scope="col" class="weft-table__select">
              <Checkbox
                :model-value="allState"
                :aria-label="t.selectAll"
                :disabled="shown.length === 0 || undefined"
                :data-weft-item="item"
                @update:model-value="toggleAll"
              />
            </th>
            <th
              v-for="column in columns"
              :key="column.key"
              scope="col"
              :aria-sort="ariaSort(column)"
              :data-align="column.align"
              :data-weft-item="column.sortable ? undefined : item"
              :style="column.width ? { inlineSize: column.width } : undefined"
            >
              <button v-if="column.sortable" type="button" class="weft-table__sort" :data-weft-item="item" @click="toggleSort(column)">
                <slot :name="`${column.key}-header`" :column="column">{{ label(column) }}</slot>
                <svg class="weft-table__arrow" :data-dir="ariaSort(column)" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
                  <path v-if="ariaSort(column) === 'descending'" d="m6 9 6 6 6-6" />
                  <path v-else-if="ariaSort(column) === 'ascending'" d="m18 15-6-6-6 6" />
                  <path v-else d="m7 15 5 5 5-5M7 9l5-5 5 5" />
                </svg>
              </button>
              <slot v-else :name="`${column.key}-header`" :column="column">{{ label(column) }}</slot>
            </th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="n in skeletons" :key="`s${n}`" class="weft-table__skeleton" aria-hidden="true">
            <td v-if="selectable" class="weft-table__select"></td>
            <td v-for="column in columns" :key="column.key"><Skeleton /></td>
          </tr>
          <tr v-if="failed">
            <td :colspan="span">
              <div class="weft-table__error" role="alert">
                <span>{{ t.error }}</span>
                <Button :label="t.retry" variant="outline" tone="neutral" size="sm" @click="load()" />
              </div>
            </td>
          </tr>
          <tr v-if="isEmpty">
            <td class="weft-table__empty" :colspan="span"><slot name="empty">{{ empty ?? t.empty }}</slot></td>
          </tr>
          <tr v-if="before > 0" class="weft-table__spacer" aria-hidden="true"><td :colspan="span" :style="{ blockSize: `${before}px` }"></td></tr>
          <tr
            v-for="i in visible"
            :key="shown[i]!.key"
            :data-index="i"
            :aria-rowindex="windowed ? i + 2 : undefined"
            :aria-selected="grid && selectable ? (chosen.has(shown[i]!.key) ? 'true' : 'false') : undefined"
            :data-selected="chosen.has(shown[i]!.key) ? 'true' : undefined"
          >
            <td v-if="selectable" class="weft-table__select" @click="noteShift">
              <Checkbox
                :model-value="chosen.has(shown[i]!.key)"
                :aria-label="t.selectRow(rowName(shown[i]!.row))"
                :data-weft-item="item"
                @update:model-value="toggleRow(shown[i]!.key, $event)"
              />
            </td>
            <component
              :is="column.rowHeader ? 'th' : 'td'"
              v-for="column in columns"
              :key="column.key"
              :scope="column.rowHeader ? 'row' : undefined"
              :data-align="column.align"
              :data-weft-item="item"
            >
              <slot :name="`${column.key}-cell`" :row="shown[i]!.row" :value="valueOf(column, shown[i]!.row)" :index="i" :column="column">{{ textOf(column, shown[i]!.row) }}</slot>
            </component>
          </tr>
          <tr v-if="after > 0" class="weft-table__spacer" aria-hidden="true"><td :colspan="span" :style="{ blockSize: `${after}px` }"></td></tr>
        </tbody>
      </table>
    </div>
    <div v-else class="weft-table__stack" :aria-busy="busy ? 'true' : undefined">
      <div v-if="selectable || columns.some((c) => c.sortable)" class="weft-table__toolbar">
        <Checkbox v-if="selectable" :model-value="allState" :label="t.selectAll" :disabled="shown.length === 0 || undefined" @update:model-value="toggleAll" />
        <div v-if="columns.some((c) => c.sortable)" class="weft-table__sorts" role="group" :aria-label="t.sortBy">
          <span class="weft-table__sorts-label" aria-hidden="true">{{ t.sortBy }}</span>
          <template v-for="column in columns" :key="column.key">
            <button v-if="column.sortable" type="button" class="weft-table__sort" :data-active="ariaSort(column) ? 'true' : undefined" @click="toggleSort(column)">
              {{ label(column) }}<span v-if="ariaSort(column)" class="weft-visually-hidden">, {{ ariaSort(column) === 'descending' ? t.descending : t.ascending }}</span>
              <svg v-if="ariaSort(column)" class="weft-table__arrow" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
                <path v-if="ariaSort(column) === 'descending'" d="m6 9 6 6 6-6" />
                <path v-else d="m18 15-6-6-6 6" />
              </svg>
            </button>
          </template>
        </div>
      </div>
      <div v-if="failed" class="weft-table__error" role="alert">
        <span>{{ t.error }}</span>
        <Button :label="t.retry" variant="outline" tone="neutral" size="sm" @click="load()" />
      </div>
      <p v-if="isEmpty" class="weft-table__empty"><slot name="empty">{{ empty ?? t.empty }}</slot></p>
      <ul class="weft-table__cards" :aria-label="caption">
        <li v-for="n in skeletons" :key="`s${n}`" class="weft-table__card weft-table__skeleton" aria-hidden="true">
          <Skeleton v-for="column in columns" :key="column.key" />
        </li>
        <li v-for="(entry, i) in shown" :key="entry.key" class="weft-table__card" :data-selected="chosen.has(entry.key) ? 'true' : undefined">
          <div v-if="selectable" class="weft-table__card-select" @click="noteShift">
            <Checkbox :model-value="chosen.has(entry.key)" :aria-label="t.selectRow(rowName(entry.row))" @update:model-value="toggleRow(entry.key, $event)" />
          </div>
          <dl class="weft-table__pairs">
            <div v-for="column in columns" :key="column.key" class="weft-table__pair" :data-align="column.align">
              <dt><slot :name="`${column.key}-header`" :column="column">{{ label(column) }}</slot></dt>
              <dd><slot :name="`${column.key}-cell`" :row="entry.row" :value="valueOf(column, entry.row)" :index="i" :column="column">{{ textOf(column, entry.row) }}</slot></dd>
            </div>
          </dl>
        </li>
      </ul>
    </div>
    <Pagination
      v-if="perPage && pageCount > 1"
      class="weft-table__pagination"
      :page="currentPage"
      :total="total"
      :items-per-page="perPage"
      :size="size"
      @update:page="page = $event"
    />
  </div>
</template>
