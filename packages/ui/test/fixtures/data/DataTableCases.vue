<script setup vapor lang="ts">
import { ref } from 'vue'
import WeftProvider from '../../../src/adaptive/WeftProvider.vue'
import DataTable from '../../../src/data/DataTable.vue'
import type {
  DataResource,
  DataTableFilter,
  DataTableLabels,
  EntityFieldConfig,
  EntitySchema,
} from '../../../src/data/entity.ts'
import type { RowKey, TableColumn, TableFetch } from '../../../src/data/table.ts'

export interface Product {
  id: number
  name: string
  price: number
  sale: number | null
  status: string
}

export type ProductInput = Omit<Product, 'id'>

const props = withDefaults(
  defineProps<{
    columns?: TableColumn<Product>[]
    resource?: DataResource<Product, ProductInput>
    fetch?: TableFetch<Product>
    create?: (body: ProductInput) => Promise<Product>
    update?: (id: number, body: Partial<ProductInput>) => Promise<Product>
    remove?: (id: number) => Promise<unknown>
    schema?: EntitySchema
    fields?: Record<string, EntityFieldConfig>
    defaults?: Partial<ProductInput>
    name?: string
    searchable?: boolean
    search?: string
    searchPlaceholder?: string
    searchDebounce?: number
    filterDefs?: DataTableFilter[]
    filters?: Record<string, string>
    perPage?: number
    defaultSort?: string | null
    selectable?: boolean
    caption?: string
    hideCaption?: boolean
    emptyTitle?: string
    emptyDescription?: string
    emptyIcon?: string
    labels?: Partial<DataTableLabels>
    customStatusCell?: boolean
  }>(),
  {
    columns: undefined,
    resource: undefined,
    fetch: undefined,
    create: undefined,
    update: undefined,
    remove: undefined,
    schema: undefined,
    fields: undefined,
    defaults: undefined,
    name: 'product',
    searchable: true,
    search: undefined,
    searchPlaceholder: undefined,
    searchDebounce: 250,
    filterDefs: undefined,
    filters: undefined,
    perPage: 10,
    defaultSort: 'name',
    selectable: false,
    caption: 'Products',
    hideCaption: true,
    emptyTitle: undefined,
    emptyDescription: undefined,
    emptyIcon: undefined,
    labels: undefined,
    customStatusCell: false,
  },
)

const COLUMNS: TableColumn<Product>[] = [
  { key: 'name', label: 'Name', sortable: true, rowHeader: true },
  { key: 'price', label: 'Price', sortable: true, align: 'end' },
  { key: 'status', label: 'Status' },
]

const searchText = ref(props.search ?? '')
const filterValues = ref<Record<string, string>>(props.filters ?? {})
const sort = ref<string | null>(props.defaultSort ?? null)
const page = ref(1)
const selected = ref<RowKey[]>([])
const savedRecords = ref<Product[]>([])
const deletedRecords = ref<Product[]>([])
const errors = ref<unknown[]>([])
const dt = ref<{ refresh(): Promise<void>; edit(r: Product | null): void; askRemove(r: Product): void } | null>(null)

defineExpose({
  searchText,
  filterValues,
  sort,
  page,
  selected,
  savedRecords,
  deletedRecords,
  errors,
  dt,
})
</script>

<template>
  <WeftProvider>
    <DataTable
      ref="dt"
      v-model:search="searchText"
      v-model:filters="filterValues"
      v-model:sort="sort"
      v-model:page="page"
      v-model:selected="selected"
      :columns="columns ?? COLUMNS"
      :resource="resource"
      :fetch="fetch"
      :create="create"
      :update="update"
      :remove="remove"
      :schema="schema"
      :fields="fields"
      :defaults="defaults"
      :name="name"
      :searchable="searchable"
      :search-placeholder="searchPlaceholder"
      :search-debounce="searchDebounce"
      :filter-defs="filterDefs"
      :per-page="perPage"
      :default-sort="defaultSort"
      :selectable="selectable"
      :caption="caption"
      :hide-caption="hideCaption"
      :empty-title="emptyTitle"
      :empty-description="emptyDescription"
      :empty-icon="emptyIcon"
      :labels="labels"
      @saved="savedRecords.push($event)"
      @deleted="deletedRecords.push($event)"
      @error="errors.push($event)"
    >
      <template v-if="customStatusCell" #status-cell="{ row }">
        <span class="custom-status">{{ row.status.toUpperCase() }}</span>
      </template>
    </DataTable>
  </WeftProvider>
</template>
