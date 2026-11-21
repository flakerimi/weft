<script setup vapor lang="ts">
// Every feedback and data component in one island: the server renders the
// table (its first page, and the skeleton of the one that fetches), the
// toggles, the stat card, the timeline, the alert, the banner, the empty
// and error states and the scroll area; the browser hydrates that markup in
// place (data-island.test.ts), then they work.
import { ref } from 'vue'
import {
  Alert,
  Banner,
  EmptyState,
  ErrorState,
  ScrollArea,
  StatCard,
  Table,
  Timeline,
  Toggle,
  ToggleGroup,
  type RowKey,
  type TableColumn,
  type TableQuery,
} from '@weft/ui'

interface Row {
  id: number
  name: string
  age: number
}
const rows: Row[] = [
  { id: 1, name: 'Grace', age: 85 },
  { id: 2, name: 'Ada', age: 36 },
  { id: 3, name: 'Alan', age: 41 },
]
const columns: TableColumn<Row>[] = [
  { key: 'name', label: 'Name', sortable: true, rowHeader: true },
  { key: 'age', label: 'Age', sortable: true, align: 'end' },
]
const sort = ref<string | null>(null)
const selected = ref<RowKey[]>([])
const asked = ref(0)
async function load(query: TableQuery) {
  asked.value++
  const sorted = [...rows].sort((a, b) => (query.sort === 'name' ? a.name.localeCompare(b.name) : a.id - b.id))
  return { items: sorted.slice((query.page - 1) * query.perPage, query.page * query.perPage), total: rows.length }
}
const bold = ref(false)
const align = ref<string | number | null | (string | number)[]>('left')
const open = ref(true)
const retries = ref(0)
</script>

<template>
  <div>
    <Banner title="News" close />
    <Alert v-model:open="open" title="Heads up" description="Check this." close />
    <Table v-model:sort="sort" v-model:selected="selected" :rows="rows" :columns="columns" caption="People" selectable :per-page="2" />
    <Table class="remote" :fetch="load" :columns="columns" :per-page="2" caption="Remote" />
    <Toggle v-model="bold" label="Bold" />
    <ToggleGroup v-model="align" aria-label="Alignment" :items="[{ label: 'Left', value: 'left' }, { label: 'Right', value: 'right' }]" />
    <StatCard label="Revenue" :value="1200" :delta="4" />
    <Timeline :items="[{ title: 'Ordered', value: 'a' }, { title: 'Shipped', value: 'b' }]" model-value="b" />
    <EmptyState title="Nothing here" />
    <ErrorState error="Down" retry @retry="retries++" />
    <ScrollArea label="Notes" max-height="4rem"><p>One</p><p>Two</p></ScrollArea>
    <output>{{ sort }}|{{ selected }}|{{ asked }}|{{ bold }}|{{ align }}|{{ open }}|{{ retries }}</output>
  </div>
</template>
