<script setup vapor lang="ts">
// An island on the run-time page /products/[id]: reads the id from the
// address, asks api.products.show, and shows the product, or that there is
// none with that id.
import { onMounted, ref } from 'vue'
import { useParams } from '@weft/ui'
import { api, ApiError, type Product } from '../.weft-types/api.ts'
import { requireUser } from '../session.ts'

const product = ref<Product | null>(null)
const missing = ref(false)
const problem = ref('')
const loading = ref(true)

onMounted(async () => {
  if (!(await requireUser())) return
  try {
    // The address gives text and the route takes a number; an id that is no
    // number goes as NaN, which the server answers with the same 404.
    product.value = await api.products.show(Number(useParams().id))
  } catch (e) {
    if (e instanceof ApiError && e.status === 404) missing.value = true
    else problem.value = e instanceof ApiError ? e.message : 'The server could not be reached; check the connection and try again.'
  } finally {
    loading.value = false
  }
})

const when = (time: string) => new Date(time).toLocaleString()
</script>

<template>
  <Stack :gap="4">
    <Skeleton v-if="loading" height="8rem" />
    <EmptyState v-else-if="missing" title="No product has that id." icon="package" />
    <Alert v-else-if="problem" tone="danger" :title="problem" />
    <Card v-else-if="product">
      <template #header>{{ product.name }}</template>
      <Stack :gap="2">
        <Stack direction="row" :gap="2" align="center">
          <span>{{ product.price.toFixed(2) }}</span>
          <Badge :label="product.status" :tone="product.status === 'live' ? 'success' : 'neutral'" variant="soft" />
        </Stack>
        <span>Made {{ when(product.created_at) }}, changed {{ when(product.updated_at) }}</span>
      </Stack>
    </Card>
    <Link to="/products">Back to the products</Link>
  </Stack>
</template>
