<script setup vapor lang="ts">
// An island that wakes after the session is known (session-island.test.ts
// holds its loader back): its first render must still be the server's, with
// no perm button, no row actions and the Guard loading, or hydration would
// not match.
import { Button, DataTable, Guard, type DataResource, type EntitySchema, type TableColumn } from '@weft/ui'

interface Product {
  id: number
  name: string
}
const rows: Product[] = [{ id: 1, name: 'Lamp' }]
const columns: TableColumn<Product>[] = [{ key: 'name', label: 'Name', rowHeader: true }]
const schema: EntitySchema = { name: { type: 'string', required: true } }
const resource: DataResource<Product, Omit<Product, 'id'>> = {
  list: async () => ({ items: rows, total: rows.length }),
  create: async (body) => ({ id: 2, ...body }),
  update: async (id, body) => ({ ...rows[0]!, ...body }),
  remove: async () => undefined,
}
</script>

<template>
  <div class="late">
    <Button label="Export" perm="products:read" />
    <Guard><p class="signed-in">Signed in</p></Guard>
    <DataTable :columns="columns" :resource="resource" :schema="schema" name="product" caption="Products" :searchable="false" :per-page="2" perm="products" />
  </div>
</template>
