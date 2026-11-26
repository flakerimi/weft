<script setup vapor lang="ts">
// A list function typed as `weft gen` writes one, handed to a Table's
// `fetch` as it is, with no adapter: vue-tsc checks that it fits the prop,
// and table.test.ts what it is asked. ListQuery and ListProduct are copied
// by hand from the generator's golden output, cmd/weft/testdata/api.ts
// (`products.list` at line 219, and `ListProduct`): when that output
// changes, change them here too.
import { Table, type TableColumn } from '../../../src/index.ts'

export interface ListQuery {
  page?: number
  per_page?: number
  sort?: string
  q?: string
  fields?: string[]
  full?: boolean
  filter?: Record<string, string>
}

interface Product {
  id: number
  name: string
}
interface ListProduct {
  items: Product[]
  total: number
  page: number
  per_page: number
}

const asked: (ListQuery | undefined)[] = []
const api = {
  products: {
    async list(query?: ListQuery): Promise<ListProduct> {
      asked.push(query)
      return { items: [{ id: 1, name: 'Tea' }], total: 1, page: query?.page ?? 1, per_page: query?.per_page ?? 20 }
    },
  },
}
const columns: TableColumn<Product>[] = [{ key: 'name', label: 'Name', sortable: true }]
defineExpose({ asked })
</script>

<template>
  <Table :fetch="api.products.list" :columns="columns" :per-page="5" search="tea" :filters="{ status: 'live', kind: '' }" sort="-name" />
</template>
