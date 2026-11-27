<script setup vapor lang="ts">
import { DataTable, type EntitySchema, type TableColumn } from '@weft/ui'

interface Item {
  id: number
  name: string
  status: string
}

type ItemInput = Omit<Item, 'id'>

const schema: EntitySchema = {
  name: { kind: 'string', required: true },
  status: { kind: 'string', required: true, enum: ['draft', 'live'] },
}

const columns: TableColumn<Item>[] = [
  { key: 'name', label: 'Name', sortable: true, rowHeader: true },
  { key: 'status', label: 'Status' },
]

let items: Item[] = [
  { id: 1, name: 'Kettle', status: 'live' },
  { id: 2, name: 'Lamp', status: 'draft' },
]

const resource = {
  list: async () => ({ items, total: items.length }),
  create: async (b: ItemInput) => ((items = [...items, { id: items.length + 1, ...b }]), items.at(-1)!),
  update: async (id: number, b: Partial<ItemInput>) => ((items = items.map((r) => (r.id === id ? { ...r, ...b } : r))), items[0]!),
  remove: async (id: number) => (items = items.filter((r) => r.id !== id)),
}
</script>

<template>
  <DataTable :resource="resource" :schema="schema" :columns="columns" name="product" caption="Products" :per-page="5" />
</template>
