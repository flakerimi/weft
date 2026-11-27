<script setup vapor lang="ts" generic="T extends object, TInput extends object = any">
// A CRUD data table over a generated Warp resource (`{ list, create, update,
// remove }`) and its field schema (`schema.<Name>Input`). It composes a
// toolbar (a debounced search Input, filter controls from `filterDefs`, and a
// "New <name>" button when `create` is present), `Table` (with a trailing
// Actions column of Edit and Delete buttons when `update` or `remove` is
// present), `EntityDialog` for create and edit, and `ConfirmDialog` for
// delete.
import { computed, nextTick, ref, shallowRef, useSlots, useTemplateRef } from 'vue'
import { useWeft } from '../adaptive/context.ts'
import { useLocale } from '../adaptive/locale.ts'
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
    /** Table columns. */
    columns: TableColumn<T>[]
    /** Generated resource `{ list, create, update, remove }`. */
    resource?: DataResource<T, TInput>
    /** List fetcher. */
    fetch?: TableFetch<T>
    /** Create handler. */
    create?: (body: TInput) => Promise<T>
    /** Update handler. */
    update?: (id: any, body: Partial<TInput>) => Promise<T>
    /** Delete handler. */
    remove?: (id: any) => Promise<unknown>
    /** Field schema. */
    schema?: EntitySchema
    /** Field configs. */
    fields?: EntityFieldConfig[]
    /** Default values. */
    defaults?: Partial<TInput> | (() => Partial<TInput>)
    /** Singular name. */
    name?: string
    /** ID property. */
    idKey?: string
    /** Label property. */
    labelKey?: string
    /** Show search input. */
    searchable?: boolean
    /** Search string. */
    search?: string
    /** Initial search. */
    defaultSearch?: string
    /** Search placeholder. */
    searchPlaceholder?: string
    /** Debounce in ms. */
    searchDebounce?: number
    /** Toolbar filters. */
    filterDefs?: DataTableFilter[]
    /** Active filters. */
    filters?: Record<string, string>
    /** Initial filters. */
    defaultFilters?: Record<string, string>
    /** Rows per page. */
    perPage?: number
    /** Current page. */
    page?: number
    /** Initial page. */
    defaultPage?: number
    /** Current sort. */
    sort?: string | null
    /** Initial sort. */
    defaultSort?: string | null
    /** Row checkboxes. */
    selectable?: boolean
    /** Selected keys. */
    selected?: RowKey[]
    /** Initial selected. */
    defaultSelected?: RowKey[]
    /** Table caption. */
    caption?: string
    /** Hide caption. */
    hideCaption?: boolean
    /** Empty title. */
    emptyTitle?: string
    /** Empty description. */
    emptyDescription?: string
    /** Empty icon. */
    emptyIcon?: string
    /** Row size. */
    size?: Size
    /** Labels override. */
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
  dialog?: (props: { state: TInput; record: T | null; schema: EntitySchema }) => unknown
  /** Custom empty state forwarded to `Table`. */
  empty?: () => unknown
  /** Forwarded to `Table` for custom column cells. */
  [cell: `${string}-cell`]: ((props: { row: T; value: unknown; index: number; column: TableColumn<T> }) => unknown) | undefined
  /** Forwarded to `Table` for custom column headers. */
  [header: `${string}-header`]: ((props: { column: TableColumn<T> }) => unknown) | undefined
}>()

const slots = useSlots()
const weft = useWeft()
const locale = useLocale(() => undefined)
const t = computed<DataTableLabels>(() => ({ ...DATA_TABLE_LABELS, ...props.labels }))
const root = useTemplateRef<HTMLElement>('root')
const width = useWidth(root, { prepaint: false })
const table = useTemplateRef<{ refresh(): Promise<void> }>('table')
const actionSize = computed<Size>(() => (weft.input.value === 'touch' ? 'md' : 'sm'))

const search = useControllable<{ search?: string }, 'search'>(props, 'search', emit, () => props.defaultSearch)
const filters = useControllable<{ filters?: Record<string, string> }, 'filters'>(props, 'filters', emit, () => ({ ...(props.defaultFilters ?? {}) }))
const sort = useControllable<{ sort?: string | null }, 'sort'>(props, 'sort', emit, () => props.defaultSort)
const page = useControllable<{ page?: number }, 'page'>(props, 'page', emit, () => props.defaultPage)
const selected = useControllable<{ selected?: RowKey[] }, 'selected'>(props, 'selected', emit, () => [...(props.defaultSelected ?? [])])

const hasDialog = computed(() => Boolean(props.schema || slots.dialog))
const fetchFn = computed(() => props.fetch ?? props.resource?.list)
const createFn = computed(() => (hasDialog.value ? (props.create ?? props.resource?.create) : undefined))
const updateFn = computed(() => (hasDialog.value ? (props.update ?? props.resource?.update) : undefined))
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
    if (def.allLabel !== false && !items.some((i) => i.value === '')) {
      items.unshift({ label: def.allLabel || t.value.anyFilter, value: '' })
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
const dialogSchema = computed<EntitySchema>(() => props.schema ?? {})

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
const removingIndex = ref(-1)
const removeProblem = ref('')

function itemLabel(record: T | null): string {
  if (!record) return props.name
  const rec = record as Record<string, unknown>
  const headerCol = props.columns.find((c) => c.rowHeader)
  const raw = rec[props.labelKey] ?? rec.title ?? (headerCol ? valueOf(headerCol, record) : undefined) ?? rec[props.idKey]
  return raw !== undefined && raw !== null && String(raw) !== '' ? String(raw) : props.name
}

function askRemove(record: T, index = -1) {
  removing.value = record
  removingIndex.value = index
  removeProblem.value = ''
  confirming.value = true
}

function focusAfterDelete(deletedIndex: number) {
  const el = root.value
  if (!el) return
  const deleteButtons = [...el.querySelectorAll<HTMLElement>('[data-weft-row-action="delete"]')]
  if (deleteButtons.length > 0) {
    const idx = deletedIndex >= 0 ? Math.min(deletedIndex, deleteButtons.length - 1) : 0
    deleteButtons[idx]?.focus()
    return
  }
  const editButtons = [...el.querySelectorAll<HTMLElement>('[data-weft-row-action="edit"]')]
  if (editButtons.length > 0) {
    const idx = deletedIndex >= 0 ? Math.min(deletedIndex, editButtons.length - 1) : 0
    editButtons[idx]?.focus()
    return
  }
  const fallback =
    [...el.querySelectorAll<HTMLElement>('button')].find((b) => b.textContent?.trim() === t.value.newAction(props.name)) ??
    el.querySelector<HTMLElement>('input[type="search"]') ??
    el
  fallback.focus()
}

async function doRemove() {
  if (!removing.value || !removeFn.value) return
  removeProblem.value = ''
  const target = removing.value
  const deletedIndex = removingIndex.value
  const id = (target as Record<string, unknown>)[props.idKey]
  await removeFn.value(id)
  emit('deleted', target)
  await table.value?.refresh()
  void nextTick(() => {
    setTimeout(() => {
      const active = typeof document !== 'undefined' ? document.activeElement : null
      if (!active || active === document.body || !active.isConnected) {
        focusAfterDelete(deletedIndex)
      }
    }, 0)
  })
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
  if (typeof val === 'number') return val.toLocaleString(locale.value)
  return String(val)
}

defineExpose({
  /** Reload the current page from `fetch` / `resource.list`. */
  refresh: () => table.value?.refresh() ?? Promise.resolve(),
  /** Open the built-in `EntityDialog` for `record` (or `null` for create mode). */
  edit,
  /** Open the built-in delete `ConfirmDialog` for `record`. */
  askRemove: (record: T) => askRemove(record, -1),
})
</script>

<template>
  <div ref="root" class="weft-data-table" tabindex="-1">
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
          <FormField v-for="f in resolvedFilters" :key="`${f.key}-${f.control}`" :label="f.label">
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
        <template v-if="hasRowActions && !$slots['actions-cell']" #actions-cell="{ row, index }">
          <Stack direction="row" :gap="2" justify="end">
            <Button
              v-if="updateFn"
              data-weft-row-action="edit"
              :label="t.editAction"
              :aria-label="t.editRowAction(itemLabel(row))"
              icon="pencil"
              variant="ghost"
              :size="actionSize"
              @click="edit(row)"
            />
            <Button
              v-if="removeFn"
              data-weft-row-action="delete"
              :label="t.deleteAction"
              :aria-label="t.deleteRowAction(itemLabel(row))"
              icon="trash"
              variant="ghost"
              tone="danger"
              :size="actionSize"
              @click="askRemove(row, index)"
            />
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
      v-if="hasDialog && (createFn || updateFn)"
      v-model:open="dialogOpen"
      :record="editing"
      :schema="dialogSchema"
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
      <template v-if="$slots.dialog" #default="{ state, record: rec, schema: s }">
        <slot name="dialog" :state="state as TInput" :record="(rec ?? null) as T | null" :schema="s" />
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
        <Alert v-if="removeProblem" role="alert" tone="danger" :title="removeProblem" />
      </template>
    </ConfirmDialog>
  </div>
</template>
