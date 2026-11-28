<script setup vapor lang="ts">
// An island on /products: the products from the server, paged, sorted,
// searched and filtered by status there; a dialog to make or change one,
// checked with the same rules the server has; and a confirmation before a
// product is deleted. Every call goes through the generated client. The
// Guard shows the table only to an account with products:read, once the
// session is known, so the table asks only then; perm="products" shows New,
// Edit and Delete only to an account with products:create, :edit and :delete.
import { sessionEnded, type TableColumn } from '@weft/ui'
import { api, ApiError, schema, type Product } from '../.weft-types/api.ts'

// The session can end while the list is open (signed out in another tab):
// the next page, sort or search is a 401, which no Retry can cure, so the
// session is asked again and the visitor sent to sign in and come back.
// Anything else keeps the table's own row and its Retry.
function listFailed(e: unknown) {
  if (e instanceof ApiError && e.status === 401) void sessionEnded()
}

const columns: TableColumn<Product>[] = [
  { key: 'name', label: 'Name', sortable: true, rowHeader: true },
  { key: 'price', label: 'Price', sortable: true, align: 'end', format: (v) => (v as number).toFixed(2) },
  { key: 'sale', label: 'Sale', align: 'end', format: (v) => (v == null ? '' : (v as number).toFixed(2)) },
  { key: 'status', label: 'Status' },
]

const rules = { ...schema.ProductInput, price: { ...schema.ProductInput.price, required: true } }
</script>

<template>
  <Guard perm="products:read">
    <DataTable
      :resource="api.products"
      perm="products"
      :schema="rules"
      :columns="columns"
      name="product"
      caption="Products"
      hide-caption
      :per-page="10"
      default-sort="name"
      search-placeholder="Name"
      :filter-defs="[{ key: 'status', label: 'Status', items: ['draft', 'live'] }]"
      :defaults="{ price: 0, sale: null, status: 'draft' }"
      :fields="[
        { name: 'name' },
        { name: 'price', step: 0.01 },
        { name: 'sale', step: 0.01, hint: 'Empty when it is not on sale.' },
        { name: 'status' },
      ]"
      empty-title="No products"
      empty-description="Make one with New product, or change the search."
      empty-icon="package"
      @error="listFailed"
    >
      <template #name-cell="{ row }">
        <Link :to="`/products/${row.id}`">{{ row.name }}</Link>
      </template>
      <template #status-cell="{ row }">
        <Badge :label="row.status" :tone="row.status === 'live' ? 'success' : 'neutral'" variant="soft" />
      </template>
    </DataTable>
    <template #loading><Skeleton height="12rem" /></template>
  </Guard>
</template>
