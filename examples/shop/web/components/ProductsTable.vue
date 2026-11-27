<script setup vapor lang="ts">
// An island on /products: the products from the server, paged, sorted,
// searched and filtered by status there; a dialog to make or change one,
// checked with the same rules the server has; and a confirmation before a
// product is deleted. Every call goes through the generated client.
import { onMounted, ref } from 'vue'
import type { TableColumn } from '@weft/ui'
import { api, ApiError, schema, type Product } from '../.weft-types/api.ts'
import { requireUser, signInAgain, useSession } from '../session.ts'

// The table asks only once the visitor is known to be signed in.
const ready = ref(false)
const { session } = useSession()
onMounted(async () => {
  ready.value = (await requireUser()) !== null
})

// The session can end while the list is open (signed out in another tab):
// the next page, sort or search is a 401, which no Retry can cure, so the
// visitor is sent to sign in and come back. Anything else keeps the table's
// own row and its Retry.
function listFailed(e: unknown) {
  if (e instanceof ApiError && e.status === 401) signInAgain()
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
  <DataTable
    v-if="ready"
    :resource="api.products"
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
  <Alert v-else-if="session.problem" tone="danger" :title="session.problem" />
  <Skeleton v-else height="12rem" />
</template>
