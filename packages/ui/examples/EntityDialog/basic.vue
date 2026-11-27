<script setup vapor lang="ts">
import { ref } from 'vue'
import { Button, EntityDialog, type EntitySchema } from '@weft/ui'

interface Product {
  id: number
  name: string
  price: number
  sale: number | null
  status: 'draft' | 'live'
}

const open = ref(false)
const schema: EntitySchema = {
  name: { type: 'string', required: true },
  price: { type: 'number', required: true, min: 0 },
  sale: { type: 'number', min: 0, nullable: true },
  status: { type: 'string', required: true, enum: ['draft', 'live'] },
}

async function create(body: Omit<Product, 'id'>): Promise<Product> {
  return { id: 1, ...body }
}
</script>

<template>
  <Button label="New product" icon="plus" @click="open = true" />
  <EntityDialog
    v-model:open="open"
    name="product"
    :schema="schema"
    :defaults="{ price: 0, status: 'draft' }"
    :fields="[
      { name: 'price', step: 0.01 },
      { name: 'sale', step: 0.01, hint: 'Empty when it is not on sale.' },
    ]"
    :create="create"
  />
</template>
