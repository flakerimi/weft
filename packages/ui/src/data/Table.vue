<script setup vapor lang="ts" generic="T extends object">
// Rows of data under column headers: a native <table> with a caption,
// th scope="col" headers and, for a `rowHeader` column, th scope="row" cells.
// A sortable header holds a button that sorts ascending, descending, then not
// (stable, on the client; through `fetch`, on the server), and says so with
// aria-sort. `selectable` adds a checkbox per row, named by the row, under a
// header checkbox that shows none, some (indeterminate) or all of the rows
// shown, with Shift+click choosing a range. The header stays in view: inside
// the table when it scrolls (a height, windowed, or wider than its box),
// else against the page. While loading it is aria-busy, drawing
// skeleton rows until it has rows; with no rows it shows the `empty` slot.
// `perPage` pages the rows on the client with a Pagination; `fetch(query)`
// asks a server for each page instead, in the wire format's `page`,
// `perPage` and `sort`. Past 200 rows on one page it renders only the rows
// in view, each a fixed height. When its own box is narrow, each row is a
// card (a list item of label–value pairs) from the same columns, with the
// same selection and sorting. With `keyboard` it is an APG grid
// (useTableGrid): one tab stop, arrows between cells over every row, windowed
// or not, the focused cell scrolled into view and its row kept drawn; the
// controls in cells leave the tab order until Enter or F2 moves in.
import { computed, onMounted, onScopeDispose, ref, useTemplateRef, watch } from 'vue'
import Skeleton from '../basics/Skeleton.vue'
import Button from '../basics/Button.vue'
import Checkbox from '../forms/Checkbox.vue'
import Pagination from '../navigation/Pagination.vue'
import { useWeft } from '../adaptive/context.ts'
import { useWidth } from '../adaptive/useWidth.ts'
import { useControllable } from '../headless/useControllable.ts'
import { useRovingFocus } from '../headless/useRovingFocus.ts'
import { useVirtualList } from '../headless/useVirtualList.ts'
import type { Size } from '../types.ts'
import { nextSort, parseSort, selectRange, TABLE_LABELS, textOf, valueOf, type RowKey, type TableColumn, type TableFetch, type TableLabels } from './table.ts'
import { useTableGrid } from './tableGrid.ts'
import { useTableData } from './useTableData.ts'
import { useLocale } from '../adaptive/locale.ts'
import '../styles/base.css'
import '../styles/table.css'

const props = withDefaults(
  defineProps<{
    /** The columns: `{ key, label, sortable, align, width, accessor, format, rowHeader }`. */
    columns: TableColumn<T>[]
    /** The rows, sorted and paged on the client. */
    rows?: T[]
    /** Asks for a page of rows instead (`{ page, perPage, sort, q, filters }` → `{ items, total }`): sorting and paging happen on the server. */
    fetch?: TableFetch<T>
    /** The search text, sent to `fetch` as `q` (`?q=`); empty sends none. */
    search?: string
    /** Filters by field name, sent to `fetch` as `filters` (`?filter[plan]=pro`). */
    filters?: Record<string, string>
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
    search: undefined,
    filters: undefined,
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
// Text sorts in the WeftProvider's locale (en-US without one), the same on the server and in the browser.
const locale = useLocale(() => undefined)
const t = computed<TableLabels>(() => ({ ...TABLE_LABELS, ...props.labels }))
const root = useTemplateRef<HTMLElement>('root')
const scroller = useTemplateRef<HTMLElement>('scroller')
const table = useTemplateRef<HTMLTableElement>('table')
const width = useWidth(root)
const cards = computed(() => width.value === 'narrow')

const sort = useControllable(props, 'sort', emit, () => props.defaultSort)
const selected = useControllable(props, 'selected', emit, () => props.defaultSelected ?? [])
const page = useControllable(props, 'page', emit, () => props.defaultPage)

const mounted = ref(false)
onMounted(() => (mounted.value = true))
const data = useTableData<T>({
  rows: () => props.rows,
  fetch: () => props.fetch,
  columns: () => props.columns,
  rowKey: () => props.rowKey,
  sort,
  page,
  perPage: () => props.perPage,
  search: () => props.search,
  filters: () => props.filters,
  loading: () => !!props.loading,
  active: () => mounted.value,
  needKeys: () => !!props.selectable,
  locale: () => locale.value,
  onError: (error) => emit('error', error),
})
const { shown, total, perPage, pageCount, currentPage, busy, failed, load } = data

const skeletons = computed(() => (busy.value && shown.value.length === 0 ? Array.from({ length: perPage.value ?? 5 }, (_, i) => i) : []))
const isEmpty = computed(() => !busy.value && !failed.value && shown.value.length === 0)
const span = computed(() => props.columns.length + (props.selectable ? 1 : 0))
const current = computed(() => parseSort(sort.value))
const ariaSort = (column: TableColumn<T>) =>
  current.value?.key === column.key ? (current.value.desc ? 'descending' : 'ascending') : undefined
const label = (column: TableColumn<T>) => column.label ?? column.key

/** A press on a sortable header: the next sort, from page 1. A parent setting both keeps its page. */
function toggleSort(column: TableColumn<T>) {
  if (!column.sortable) return
  sort.value = nextSort(sort.value, column.key)
  page.value = 1
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

// The grid: one tab stop over the cells (a sort button or a checkbox stands for its cell).
const grid = computed(() => !!props.keyboard && !cards.value)
const gridEl = computed(() => (grid.value ? table.value : null))
// Windowed past 200 rows: fixed-height rows, the focused one pinned, spacer rows for the rest.
const ROW = { sm: 36, md: 44, lg: 52 } as const
const rowPx = computed(() => props.rowHeight ?? (weft.input.value === 'touch' ? Math.max(44, ROW[props.size]) : ROW[props.size]))
let pinned = (): number => -1
const list = useVirtualList(scroller, {
  count: () => shown.value.length,
  itemHeight: () => rowPx.value,
  enabled: () => !cards.value && shown.value.length > 200,
  pinned: () => (grid.value ? pinned() : -1),
  // Measured when scrolling: the caption and header above the rows, and the header where it sticks.
  leading: () => {
    const body = table.value?.tBodies[0]
    const el = scroller.value
    const box = el?.getBoundingClientRect()
    // Not laid out (no rows yet, hidden): nothing to count.
    if (!body || !el || !box?.height) return 0
    return body.getBoundingClientRect().top - box.top - el.clientTop + el.scrollTop
  },
  stuck: () => (props.sticky ? (table.value?.tHead?.getBoundingClientRect().height ?? 0) : 0),
})
const windowed = computed(() => list.virtual.value)
// Heard first (capture), so the roving focus below only keeps the tab stop.
const gridKeys = useTableGrid(gridEl, {
  rows: () => shown.value.length,
  columns: () => span.value,
  windowed: () => windowed.value,
  scrollTo: (index) => list.scrollTo(index),
})
pinned = () => gridKeys.focusRow.value
useRovingFocus(gridEl, { orientation: 'grid', columns: () => span.value, loop: false, inner: 'ignore' })
const item = computed(() => (grid.value ? '' : undefined))
const col = (index: number) => (grid.value ? index : undefined)

/** The rows to draw, in order, with a spacer row for each run of rows not drawn. */
type Part = { spacer: number; key: string } | { spacer?: undefined; key: RowKey; i: number }
const parts = computed<Part[]>(() => {
  const indices = list.indices.value
  if (!windowed.value) return indices.map((i) => ({ key: shown.value[i]!.key, i }))
  const out: Part[] = []
  let next = 0
  for (const i of indices) {
    const gap = list.offset(i) - list.offset(next)
    if (gap > 0) out.push({ spacer: gap, key: `gap:${i}` })
    out.push({ key: shown.value[i]!.key, i })
    next = i + 1
  }
  const tail = list.total.value - list.offset(next)
  if (tail > 0) out.push({ spacer: tail, key: 'gap:end' })
  return out
})

// The box: it scrolls when it has a height, is windowed, or the table is wider than it; else it
// clips, so the sticky header sticks against the page. Until measured (and on the server) it scrolls.
const maxHeight = computed(() => props.height ?? (windowed.value ? '30rem' : undefined))
const wide = ref(true)
const scrolls = computed(() => !!maxHeight.value || wide.value)
let sizes: ResizeObserver | undefined
const measure = () => {
  const box = scroller.value
  const inner = table.value
  if (box && inner) wide.value = inner.scrollWidth > box.clientWidth + 1
}
watch(
  [scroller, table],
  ([box, inner]) => {
    sizes?.disconnect()
    sizes = undefined
    if (!box || !inner) return
    measure()
    if (typeof ResizeObserver === 'undefined') return
    sizes = new ResizeObserver(measure)
    sizes.observe(box)
    sizes.observe(inner)
  },
  { flush: 'post' },
)
onScopeDispose(() => sizes?.disconnect())

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
    <div v-if="!cards" ref="scroller" class="weft-table__scroller" :data-scroll="scrolls ? 'true' : undefined" :style="maxHeight ? { maxBlockSize: maxHeight } : undefined">
      <table
        ref="table"
        class="weft-table__table"
        :role="grid ? 'grid' : undefined"
        :aria-busy="busy ? 'true' : undefined"
        :aria-rowcount="windowed ? shown.length + 1 : undefined"
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
                :data-col="col(0)"
                @update:model-value="toggleAll"
              />
            </th>
            <th
              v-for="(column, c) in columns"
              :key="column.key"
              scope="col"
              :aria-sort="ariaSort(column)"
              :data-align="column.align"
              :data-weft-item="column.sortable ? undefined : item"
              :data-col="column.sortable ? undefined : col(c + (selectable ? 1 : 0))"
              :style="column.width ? { inlineSize: column.width } : undefined"
            >
              <button v-if="column.sortable" type="button" class="weft-table__sort" :data-weft-item="item" :data-col="col(c + (selectable ? 1 : 0))" @click="toggleSort(column)">
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
          <template v-for="part in parts" :key="part.key">
            <tr v-if="part.spacer !== undefined" class="weft-table__spacer" aria-hidden="true"><td :colspan="span" :style="{ blockSize: `${part.spacer}px` }"></td></tr>
            <tr
              v-else
              :data-index="part.i"
              :aria-rowindex="windowed ? part.i + 2 : undefined"
              :aria-selected="grid && selectable ? (chosen.has(part.key) ? 'true' : 'false') : undefined"
              :data-selected="chosen.has(part.key) ? 'true' : undefined"
            >
              <td v-if="selectable" class="weft-table__select" @click="noteShift">
                <Checkbox
                  :model-value="chosen.has(part.key)"
                  :aria-label="t.selectRow(rowName(shown[part.i]!.row))"
                  :data-weft-item="item"
                  :data-col="col(0)"
                  @update:model-value="toggleRow(part.key, $event)"
                />
              </td>
              <component
                :is="column.rowHeader ? 'th' : 'td'"
                v-for="(column, c) in columns"
                :key="column.key"
                :scope="column.rowHeader ? 'row' : undefined"
                :data-align="column.align"
                :data-weft-item="item"
                :data-col="col(c + (selectable ? 1 : 0))"
              >
                <slot :name="`${column.key}-cell`" :row="shown[part.i]!.row" :value="valueOf(column, shown[part.i]!.row)" :index="part.i" :column="column">{{ textOf(column, shown[part.i]!.row) }}</slot>
              </component>
            </tr>
          </template>
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
      <div v-if="isEmpty" class="weft-table__empty"><slot name="empty">{{ empty ?? t.empty }}</slot></div>
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
