<script setup vapor lang="ts">
// An island on the run-time page /products/[id]: reads the id from the
// address, asks api.products.show, and shows the product, or that there is
// none with that id.
import { onMounted, ref } from 'vue'
import { useParams } from '@weft/ui'
import { api, ApiError, type Product } from '../.weft-types/api.ts'
import { formProblem, productId } from '../answers.ts'
import { requireUser, useSession } from '../session.ts'

const product = ref<Product | null>(null)
const missing = ref(false)
const problem = ref('')
const loading = ref(true)

onMounted(async () => {
  try {
    if (!(await requireUser())) {
      problem.value = useSession().session.problem
      return
    }
    // Only an id as the server writes one: /products/0x2a is not product 42.
    const id = productId(useParams('/products/:id').id)
    if (id === null) missing.value = true
    else product.value = await api.products.show(id)
  } catch (e) {
    if (e instanceof ApiError && e.status === 404) missing.value = true
    else problem.value = formProblem(e).message
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
