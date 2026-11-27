<script setup vapor lang="ts" generic="T extends object, TInput extends object = any">
// A CRUD data table over a generated Warp resource (`{ list, create, update,
// remove }`) and its field schema (`schema.<Name>Input`). It composes a
// toolbar (a debounced search Input, filter controls from `filterDefs`, and a
// "New <name>" button when `create` is present), `Table` (with a trailing
// Actions column of Edit and Delete buttons when `update` or `remove` is
// present), `EntityDialog` for create and edit, and `ConfirmDialog` for
// delete.
import { computed, ref, shallowRef, useSlots, useTemplateRef } from 'vue'
import { useWidth } from '../adaptive/useWidth.ts'
import Button from '../basics/Button.vue'
import Alert from '../feedback/Alert.vue'
import EmptyState from '../feedback/EmptyState.vue'
import FormField from '../forms/FormField.vue'
import Input from '../forms/Input.vue'
import RadioGroup from '../forms/RadioGroup.vue'
import { useControllable } from '../headless/useControllable.ts'
import Stack from '../layout/Stack.vue'
import ConfirmDialog from '../overlays/ConfirmDialog.vue'
import Select from '../selection/Select.vue'
import type { Size } from '../types.ts'
import EntityDialog from './EntityDialog.vue'
import {
  DATA_TABLE_LABELS,
  entityProblem,
  humanLabel,
  type DataResource,
  type DataTableFilter,
  type DataTableLabels,
  type EntityFieldConfig,
  type EntitySchema,
} from './entity.ts'
import Table from './Table.vue'
import { valueOf, type RowKey, type TableColumn, type TableFetch } from './table.ts'
import '../styles/base.css'

const props = withDefaults(
  defineProps<{
    /** Table columns; `actions` is appended when `update` or `remove` is given. */
    columns: TableColumn<T>[]
    /** Shorthand `{ list, create, update, remove }` (e.g. `api.products`). */
    resource?: DataResource<T, TInput>
    /** List fetcher (overrides `resource.list`). */
    fetch?: TableFetch<T>
    /** Create handler (overrides `resource.create`). */
    create?: (body: TInput) => Promise<T>
    /** Update handler (overrides `resource.update`). */
    update?: (id: any, body: Partial<TInput>) => Promise<T>
    /** Delete handler (overrides `resource.remove`). */
    remove?: (id: any) => Promise<unknown>
    /** Schema passed to `EntityDialog`. */
    schema?: EntitySchema
    /** Field overrides passed to `EntityDialog`. */
    fields?: Record<string, EntityFieldConfig>
    /** Default values for create mode. */
    defaults?: Record<string, unknown>
    /** Singular entity name (e.g. `'product'`). */
    name?: string
    /** Primary key property; default `'id'`. */
    idKey?: string
    /** Display label property; default `'name'`. */
    labelKey?: string
    /** Show the search input; default `true`. */
    searchable?: boolean
    /** Bound with `v-model:search`. */
    search?: string
    /** Initial search string. */
    defaultSearch?: string
    /** Placeholder for the search input. */
    searchPlaceholder?: string
    /** Debounce delay in ms; default `250`. */
    searchDebounce?: number
    /** Toolbar filter controls. */
    filterDefs?: DataTableFilter[]
    /** Active filter values, bound with `v-model:filters`. */
    filters?: Record<string, string>
    /** Initial filter values. */
    defaultFilters?: Record<string, string>
    /** Rows per page; default `20`. */
    perPage?: number
    /** Current page, bound with `v-model:page`. */
    page?: number
    /** Initial page; default `1`. */
    defaultPage?: number
    /** Current sort, bound with `v-model:sort`. */
    sort?: string | null
    /** Initial sort. */
    defaultSort?: string | null
    /** Enable row selection checkboxes. */
    selectable?: boolean
    /** Selected row keys, bound with `v-model:selected`. */
    selected?: RowKey[]
    /** Initial selected row keys. */
    defaultSelected?: RowKey[]
    /** Table caption. */
    caption?: string
    /** Visually hide the caption; default `true`. */
    hideCaption?: boolean
    /** EmptyState title when there are no rows. */
    emptyTitle?: string
    /** EmptyState description when there are no rows. */
    emptyDescription?: string
    /** EmptyState icon when there are no rows. */
    emptyIcon?: string
    /** Row size (`'sm' | 'md' | 'lg'`). */
    size?: Size
    /** Labels override for i18n. */
    labels?: Partial<DataTableLabels>
  }>(),
  {
    resource: undefined,
    fetch: undefined,
    create: undefined,
    update: undefined,
    remove: undefined,
    schema: undefined,
    fields: undefined,
    defaults: undefined,
    name: 'item',
    idKey: 'id',
    labelKey: 'name',
    searchable: true,
    search: undefined,
    defaultSearch: '',
    searchPlaceholder: undefined,
    searchDebounce: 250,
    filterDefs: undefined,
    filters: undefined,
    perPage: 20,
    page: undefined,
    defaultPage: 1,
    sort: undefined,
    defaultSort: null,
    selectable: false,
    selected: undefined,
    caption: undefined,
    hideCaption: true,
    emptyTitle: undefined,
    emptyDescription: undefined,
    emptyIcon: undefined,
    size: 'md',
  },
)

const emit = defineEmits<{
  /** The search text changed. */
  'update:search': [q: string]
  /** The active filter values changed. */
  'update:filters': [filters: Record<string, string>]
  /** The sort changed. */
  'update:sort': [sort: string | null]
  /** The current page changed. */
  'update:page': [page: number]
  /** The chosen rows changed. */
  'update:selected': [keys: RowKey[]]
  /** Emitted after create or update succeeds. */
  saved: [record: T]
  /** Emitted after delete succeeds. */
  deleted: [record: T]
  /** Emitted when `fetch`, `create`, `update`, or `remove` rejects. */
  error: [error: unknown]
}>()

defineSlots<{
  /** Custom controls placed inside the toolbar beside the search input and filters. */
  toolbar?: () => unknown
  /** Custom buttons placed beside the `"New <name>"` button at the end of the toolbar. */
  'toolbar-actions'?: () => unknown
  /** Custom form body forwarded to the built-in `EntityDialog`'s default slot. */
  dialog?: (props: { state: TInput; record: T | null }) => unknown
  /** Custom empty state forwarded to `Table`. */
  empty?: () => unknown
  /** Forwarded to `Table` for custom column cells. */
  [cell: `${string}-cell`]: ((props: { row: T; value: unknown; index: number; column: TableColumn<T> }) => unknown) | undefined
  /** Forwarded to `Table` for custom column headers. */
  [header: `${string}-header`]: ((props: { column: TableColumn<T> }) => unknown) | undefined
}>()

const slots = useSlots()
const t = computed<DataTableLabels>(() => ({ ...DATA_TABLE_LABELS, ...props.labels }))
const root = useTemplateRef<HTMLElement>('root')
const width = useWidth(root, { prepaint: false })
const table = useTemplateRef<{ refresh(): Promise<void> }>('table')

const search = useControllable(props, 'search', emit, () => props.defaultSearch)
const filters = useControllable(props, 'filters', emit, () => ({ ...(props.defaultFilters ?? {}) }))
const sort = useControllable(props, 'sort', emit, () => props.defaultSort)
const page = useControllable(props, 'page', emit, () => props.defaultPage)
const selected = useControllable(props, 'selected', emit, () => [...(props.defaultSelected ?? [])])

const fetchFn = computed(() => props.fetch ?? props.resource?.list)
const createFn = computed(() => props.create ?? props.resource?.create)
const updateFn = computed(() => props.update ?? props.resource?.update)
const removeFn = computed(() => props.remove ?? props.resource?.remove)
const hasRowActions = computed(() => Boolean(updateFn.value || removeFn.value))

const allColumns = computed<TableColumn<T>[]>(() => {
  if (!hasRowActions.value || props.columns.some((c) => c.key === 'actions')) {
    return props.columns
  }
  return [
    ...props.columns,
    { key: 'actions', label: t.value.actionsColumn, align: 'end' },
  ]
})

const customColumns = computed(() => allColumns.value.filter((c) => c.key !== 'actions' || Boolean(slots['actions-cell'])))

interface ResolvedFilter {
  key: string
  label: string
  control: 'radio' | 'select'
  items: { label: string; value: string }[]
}

const resolvedFilters = computed<ResolvedFilter[]>(() => {
  if (!props.filterDefs?.length) return []
  return props.filterDefs.map((def) => {
    const items: { label: string; value: string }[] = def.items.map((item) =>
      typeof item === 'string' ? { label: humanLabel(item), value: item } : { label: item.label, value: item.value },
    )
    if (!items.some((i) => i.value === '')) {
      items.unshift({ label: t.value.anyFilter, value: '' })
    }
    const control = def.control ?? (items.length <= 4 && width.value !== 'narrow' ? 'radio' : 'select')
    return {
      key: def.key,
      label: def.label ?? humanLabel(def.key),
      control,
      items,
    }
  })
})

function getFilter(key: string): string {
  return filters.value?.[key] ?? ''
}
function setFilter(key: string, val: unknown) {
  const next = { ...(filters.value ?? {}), [key]: val == null ? '' : String(val) }
  filters.value = next
}

const hasToolbar = computed(() =>
  Boolean(
    props.searchable ||
      resolvedFilters.value.length > 0 ||
      createFn.value ||
      slots.toolbar ||
      slots['toolbar-actions'],
  ),
)

// Create / Edit dialog state
const dialogOpen = ref(false)
const editing = shallowRef<T | null>(null)

function edit(record: T | null) {
  editing.value = record
  dialogOpen.value = true
}

function onSaved(record: T) {
  emit('saved', record)
  void table.value?.refresh()
}

// Delete confirmation state
const confirming = ref(false)
const removing = shallowRef<T | null>(null)
const removeProblem = ref('')

function itemLabel(record: T | null): string {
  if (!record) return `this ${props.name}`
  const rec = record as Record<string, unknown>
  const headerCol = props.columns.find((c) => c.rowHeader)
  const raw = rec[props.labelKey] ?? (headerCol ? valueOf(headerCol, record) : undefined) ?? rec[props.idKey]
  return raw !== undefined && raw !== null && String(raw) !== '' ? String(raw) : `this ${props.name}`
}

function askRemove(record: T) {
  removing.value = record
  removeProblem.value = ''
  confirming.value = true
}

async function doRemove() {
  if (!removing.value || !removeFn.value) return
  removeProblem.value = ''
  const target = removing.value
  const id = (target as Record<string, unknown>)[props.idKey]
  await removeFn.value(id)
  emit('deleted', target)
  await table.value?.refresh()
}

function onRemoveError(e: unknown) {
  emit('error', e)
  removeProblem.value = entityProblem(e, new Set(), t.value).message
}

function formatCell(column: TableColumn<T>, row: T): string {
  const val = valueOf(column, row)
  if (column.format) return column.format(val, row)
  if (val === null || val === undefined) return ''
  if (typeof val === 'boolean') return val ? t.value.booleanYes : t.value.booleanNo
  if (typeof val === 'number') return val.toLocaleString()
  return String(val)
}

defineExpose({
  /** Reload the current page from `fetch` / `resource.list`. */
  refresh: () => table.value?.refresh() ?? Promise.resolve(),
  /** Open the built-in `EntityDialog` for `record` (or `null` for create mode). */
  edit,
  /** Open the built-in delete `ConfirmDialog` for `record`. */
  askRemove,
})
</script>

<template>
  <div ref="root" class="weft-data-table">
    <Stack :gap="4">
      <Stack v-if="hasToolbar" direction="row" :gap="3" align="end" justify="between" wrap>
        <Stack direction="row" :gap="3" align="end" wrap>
          <FormField v-if="searchable" :label="t.search">
            <Input
              :model-value="search"
              type="search"
              :placeholder="searchPlaceholder ?? t.searchPlaceholder"
              @update:model-value="search = $event == null ? '' : String($event)"
            />
          </FormField>
          <FormField v-for="f in resolvedFilters" :key="f.key" :label="f.label">
            <RadioGroup
              v-if="f.control === 'radio'"
              :model-value="getFilter(f.key)"
              :items="f.items"
              orientation="horizontal"
              @update:model-value="setFilter(f.key, $event)"
            />
            <Select
              v-else
              :model-value="getFilter(f.key)"
              :items="f.items"
              @update:model-value="setFilter(f.key, $event)"
            />
          </FormField>
          <slot name="toolbar" />
        </Stack>
        <Stack direction="row" :gap="2" align="center">
          <slot name="toolbar-actions" />
          <Button v-if="createFn" :label="t.newAction(name)" icon="plus" @click="edit(null)" />
        </Stack>
      </Stack>

      <Table
        ref="table"
        v-model:sort="sort"
        v-model:page="page"
        v-model:selected="selected"
        :fetch="fetchFn"
        :columns="allColumns"
        :row-key="idKey"
        :search="search"
        :search-debounce="searchDebounce"
        :filters="filters"
        :per-page="perPage"
        :selectable="selectable"
        :caption="caption"
        :hide-caption="hideCaption"
        :size="size"
        :labels="t"
        @error="emit('error', $event)"
      >
        <template v-for="col in allColumns" :key="`h-${col.key}`" #[`${col.key}-header`]="{ column }">
          <slot v-if="$slots[`${col.key}-header`]" :name="`${col.key}-header`" :column="column" />
          <template v-else>{{ column.label ?? column.key }}</template>
        </template>
        <template v-for="col in customColumns" :key="`c-${col.key}`" #[`${col.key}-cell`]="{ row, value, index, column }">
          <slot v-if="$slots[`${col.key}-cell`]" :name="`${col.key}-cell`" :row="row" :value="value" :index="index" :column="column" />
          <template v-else>{{ formatCell(column, row) }}</template>
        </template>
        <template v-if="hasRowActions && !$slots['actions-cell']" #actions-cell="{ row }">
          <Stack direction="row" :gap="2" justify="end">
            <Button v-if="updateFn" :label="t.editAction" icon="pencil" variant="ghost" size="sm" @click="edit(row)" />
            <Button v-if="removeFn" :label="t.deleteAction" icon="trash" variant="ghost" tone="danger" size="sm" @click="askRemove(row)" />
          </Stack>
        </template>
        <template #empty>
          <slot name="empty">
            <EmptyState
              v-if="emptyTitle || emptyDescription || emptyIcon"
              :title="emptyTitle"
              :description="emptyDescription"
              :icon="emptyIcon"
              variant="ghost"
            />
            <template v-else>{{ t.empty }}</template>
          </slot>
        </template>
      </Table>
    </Stack>

    <EntityDialog
      v-if="schema && (createFn || updateFn)"
      v-model:open="dialogOpen"
      :record="editing"
      :schema="schema"
      :create="createFn"
      :update="updateFn"
      :id-key="idKey"
      :label-key="labelKey"
      :name="name"
      :defaults="defaults"
      :fields="fields"
      :labels="t"
      @saved="onSaved($event as T)"
      @error="emit('error', $event)"
    >
      <template v-if="$slots.dialog" #default="{ state, record: rec }">
        <slot name="dialog" :state="state as TInput" :record="(rec ?? null) as T | null" />
      </template>
    </EntityDialog>

    <ConfirmDialog
      v-if="removeFn"
      v-model:open="confirming"
      :title="t.deleteTitle(itemLabel(removing))"
      :description="t.deleteDescription"
      :confirm-label="t.deleteConfirm"
      :cancel-label="t.cancel"
      tone="danger"
      @confirm="doRemove"
      @error="onRemoveError"
    >
      <template #body>
        <Alert v-if="removeProblem" tone="danger" :title="removeProblem" />
      </template>
    </ConfirmDialog>
  </div>
</template>
