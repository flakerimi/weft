<script setup vapor lang="ts">
import { ref } from 'vue'
import Table from '../../../src/data/Table.vue'
import WeftProvider from '../../../src/adaptive/WeftProvider.vue'
import type { Input } from '../../../src/adaptive/detect.ts'
import type { RowKey, TableColumn, TableFetch } from '../../../src/data/table.ts'

export interface Person {
  id: number
  name: string
  age: number | null
  city: string
}

const props = withDefaults(
  defineProps<{
    rows?: Person[]
    fetch?: TableFetch<Person>
    columns?: TableColumn<Person>[]
    selectable?: boolean
    perPage?: number
    loading?: boolean
    keyboard?: boolean
    caption?: string
    input?: Input
    dir?: 'ltr' | 'rtl'
    slots?: boolean
    controls?: boolean
    rowHeight?: number
    height?: string
  }>(),
  { rows: undefined, fetch: undefined, columns: undefined, caption: 'People' },
)

const COLUMNS: TableColumn<Person>[] = [
  { key: 'name', label: 'Name', sortable: true, rowHeader: true },
  { key: 'age', label: 'Age', sortable: true, align: 'end' },
  { key: 'city', label: 'City' },
]
const sort = ref<string | null>(null)
const selected = ref<RowKey[]>([])
const page = ref(1)
const errors = ref<unknown[]>([])
const table = ref<{ refresh(): Promise<void> } | null>(null)
defineExpose({ sort, selected, page, errors, table })
</script>

<template>
  <WeftProvider :input="input">
    <div :dir="dir">
      <Table
        ref="table"
        v-model:sort="sort"
        v-model:selected="selected"
        v-model:page="page"
        :rows="rows"
        :fetch="fetch"
        :columns="columns ?? COLUMNS"
        :selectable="selectable"
        :per-page="perPage"
        :loading="loading"
        :keyboard="keyboard"
        :caption="caption"
        :row-height="rowHeight"
        :height="height"
        @error="errors.push($event)"
      >
        <template v-if="slots" #city-cell="{ value, row }"><em class="city">{{ value }} ({{ row.id }})</em></template>
        <template v-if="controls" #age-cell="{ value }"><button type="button" class="edit">Edit {{ value }}</button></template>
        <template v-if="slots" #empty><p class="no-people">No people yet</p></template>
      </Table>
    </div>
  </WeftProvider>
</template>
