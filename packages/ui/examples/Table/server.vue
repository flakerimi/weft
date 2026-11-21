<script setup vapor lang="ts">
// A Table driven by a server speaking the wire format: each page and sort
// is a request (here answered in memory, as a weft.Resource would).
import { EmptyState, Table, tableSearch, type TableColumn, type TableQuery, type TableResult } from '@weft/ui'

interface Customer {
  id: number
  name: string
  plan: string
  since: string
}

const all: Customer[] = Array.from({ length: 87 }, (_, i) => ({
  id: i + 1,
  name: `Customer ${String(i + 1).padStart(2, '0')}`,
  plan: ['basic', 'pro', 'business'][i % 3]!,
  since: `2026-${String((i % 12) + 1).padStart(2, '0')}-01`,
}))

/** In an app: `fetch('/api/customers?' + tableSearch(query)).then((r) => r.json())`. */
async function load(query: TableQuery): Promise<TableResult<Customer>> {
  const params = tableSearch(query)
  const sort = params.get('sort')
  const key = (sort?.replace(/^-/, '') ?? 'id') as keyof Customer
  const sorted = [...all].sort((a, b) => String(a[key]).localeCompare(String(b[key]), undefined, { numeric: true }) * (sort?.startsWith('-') ? -1 : 1))
  const start = (query.page - 1) * query.perPage
  return { items: sorted.slice(start, start + query.perPage), total: all.length, page: query.page, perPage: query.perPage }
}

const columns: TableColumn<Customer>[] = [
  { key: 'name', label: 'Name', sortable: true, rowHeader: true },
  { key: 'plan', label: 'Plan', sortable: true },
  { key: 'since', label: 'Customer since', sortable: true, align: 'end' },
]
</script>

<template>
  <Table :fetch="load" :columns="columns" :per-page="10" caption="Customers" height="24rem">
    <template #empty>
      <EmptyState title="No customers yet" icon="users" variant="ghost" />
    </template>
  </Table>
</template>
