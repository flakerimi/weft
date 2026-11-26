<script setup vapor lang="ts">
// An island on /products: the products from the server, paged, sorted,
// searched and filtered by status there; a dialog to make or change one,
// checked with the same rules the server has; and a confirmation before a
// product is deleted. Every call goes through the generated client.
import { computed, onMounted, reactive, ref, useTemplateRef } from 'vue'
import type { FormApi, TableColumn } from '@weft/ui'
import { api, ApiError, schema, type Product, type ProductInput } from '../.weft-types/api.ts'
import { formProblem } from '../answers.ts'
import { requireUser, signInAgain, useSession } from '../session.ts'

// The table asks only once the visitor is known to be signed in.
const ready = ref(false)
const { session } = useSession()
onMounted(async () => {
  ready.value = (await requireUser()) !== null
})

const table = useTemplateRef<{ refresh(): void }>('table')

// The session can end while the list is open (signed out in another tab):
// the next page, sort or search is a 401, which no Retry can cure, so the
// visitor is sent to sign in and come back. Anything else keeps the table's
// own row and its Retry.
function listFailed(e: unknown) {
  if (e instanceof ApiError && e.status === 401) signInAgain()
}
const search = ref('')
const status = ref('')
const filters = computed(() => ({ status: status.value }))
const statuses = [
  { label: 'Any', value: '' },
  { label: 'Draft', value: 'draft' },
  { label: 'Live', value: 'live' },
]

const columns: TableColumn<Product>[] = [
  { key: 'name', label: 'Name', sortable: true, rowHeader: true },
  { key: 'price', label: 'Price', sortable: true, align: 'end', format: (v) => (v as number).toFixed(2) },
  { key: 'sale', label: 'Sale', align: 'end', format: (v) => (v == null ? '' : (v as number).toFixed(2)) },
  { key: 'status', label: 'Status' },
  { key: 'actions', label: 'Actions', align: 'end' },
]

// The dialog that makes a product, or changes the one in `editing`. The sale
// is a pointer in Go, nullable in the schema, so emptying it sends null,
// which clears it. The price is not: an emptied price cannot be sent, so the
// form asks for one (0 is a price), rather than keep the old one unsaid.
const open = ref(false)
const editing = ref<Product | null>(null)
const draft = reactive<ProductInput>({ name: '', price: 0, sale: null, status: 'draft' })
const rules = { ...schema.ProductInput, price: { ...schema.ProductInput.price, required: true } }
const problem = ref('')
const formKey = ref(0)
const form = useTemplateRef<FormApi>('form')

function edit(product: Product | null) {
  editing.value = product
  Object.assign(draft, product ? { name: product.name, price: product.price, sale: product.sale ?? null, status: product.status } : { name: '', price: 0, sale: null, status: 'draft' })
  problem.value = ''
  formKey.value++ // a fresh form, with no errors from the last time
  open.value = true
}

/** Only the fields that differ from the product as it was loaded. */
function changes(values: ProductInput, was: Product): Partial<ProductInput> {
  const out: Partial<ProductInput> = {}
  if (values.name !== was.name) out.name = values.name
  if (values.price !== was.price) out.price = values.price
  if ('sale' in values && values.sale !== (was.sale ?? null)) out.sale = values.sale
  if (values.status !== was.status) out.status = values.status
  return out
}

async function save(values: ProductInput) {
  problem.value = ''
  try {
    if (editing.value) {
      const changed = changes(values, editing.value)
      if (Object.keys(changed).length > 0) await api.products.update(editing.value.id, changed)
    } else {
      await api.products.create(values)
    }
    open.value = false
    table.value?.refresh()
  } catch (e) {
    const { fields, message } = formProblem(e)
    form.value?.setErrors(fields)
    problem.value = message
  }
}

// The product to delete, once its Delete button is pressed.
const removing = ref<Product | null>(null)
const confirming = ref(false)
const removeProblem = ref('')

function askRemove(product: Product) {
  removing.value = product
  removeProblem.value = ''
  confirming.value = true
}

async function remove() {
  if (!removing.value) return
  await api.products.remove(removing.value.id)
  table.value?.refresh()
}

function removeFailed(e: unknown) {
  removeProblem.value = formProblem(e).message
}
</script>

<template>
  <Stack :gap="4">
    <Stack direction="row" :gap="3" align="end" justify="between" wrap>
      <Stack direction="row" :gap="3" align="end" wrap>
        <FormField label="Search">
          <Input v-model="search" type="search" placeholder="Name" />
        </FormField>
        <FormField label="Status">
          <RadioGroup v-model="status" :items="statuses" orientation="horizontal" />
        </FormField>
      </Stack>
      <Button label="New product" icon="plus" @click="edit(null)" />
    </Stack>

    <Table
      v-if="ready"
      ref="table"
      :fetch="api.products.list"
      :columns="columns"
      :search="search"
      :filters="filters"
      :per-page="10"
      default-sort="name"
      caption="Products"
      hide-caption
      @error="listFailed"
    >
      <template #name-cell="{ row }">
        <Link :to="`/products/${row.id}`">{{ row.name }}</Link>
      </template>
      <template #status-cell="{ row }">
        <Badge :label="row.status" :tone="row.status === 'live' ? 'success' : 'neutral'" variant="soft" />
      </template>
      <template #actions-cell="{ row }">
        <Stack direction="row" :gap="2" justify="end">
          <Button label="Edit" icon="pencil" variant="ghost" size="sm" @click="edit(row)" />
          <Button label="Delete" icon="trash" variant="ghost" tone="danger" size="sm" @click="askRemove(row)" />
        </Stack>
      </template>
      <template #empty>
        <EmptyState title="No products" description="Make one with New product, or change the search." icon="package" variant="ghost" />
      </template>
    </Table>
    <Alert v-else-if="session.problem" tone="danger" :title="session.problem" />
    <Skeleton v-else height="12rem" />
  </Stack>

  <Dialog v-model:open="open" :title="editing ? `Edit ${editing.name}` : 'New product'">
    <template #body>
      <Form :key="formKey" ref="form" :state="draft" :schema="rules" @submit="save">
        <Stack :gap="4">
          <Alert v-if="problem" tone="danger" :title="problem" />
          <FormField label="Name" name="name">
            <Input v-model="draft.name" />
          </FormField>
          <FormField label="Price" name="price">
            <InputNumber v-model="draft.price" :min="0" :step="0.01" />
          </FormField>
          <FormField label="Sale" name="sale" hint="Empty when it is not on sale.">
            <InputNumber v-model="draft.sale" :min="0" :step="0.01" />
          </FormField>
          <FormField label="Status" name="status">
            <RadioGroup v-model="draft.status" :items="statuses.slice(1)" orientation="horizontal" />
          </FormField>
          <Stack direction="row" :gap="2" justify="end">
            <Button label="Cancel" variant="ghost" @click="open = false" />
            <Button type="submit" :label="editing ? 'Save' : 'Create'" />
          </Stack>
        </Stack>
      </Form>
    </template>
  </Dialog>

  <ConfirmDialog
    v-model:open="confirming"
    :title="`Delete ${removing?.name ?? 'this product'}?`"
    description="It is gone for good; this cannot be undone."
    confirm-label="Delete"
    tone="danger"
    @confirm="remove"
    @error="removeFailed"
  >
    <template #body>
      <Alert v-if="removeProblem" tone="danger" :title="removeProblem" />
    </template>
  </ConfirmDialog>
</template>
