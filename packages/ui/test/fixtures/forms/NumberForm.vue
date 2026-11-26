<script setup vapor lang="ts">
import { reactive, ref } from 'vue'
import { Button, Form, FormField, InputNumber, type FieldSchema } from '../../../src/index.ts'

// An optional number, a required one, one named by its own prop outside a
// FormField, and a null the app set that no number field holds.
const state = reactive({
  price: 5 as number | null,
  stock: 1 as number | null,
  weight: null as number | null,
  note: null as string | null,
})
const schema: FieldSchema<keyof typeof state> = { stock: { required: true } }
const submitted = ref<Record<string, unknown>[]>([])
const failed = ref<Record<string, string>[]>([])
defineExpose({ state, submitted, failed })
</script>

<template>
  <Form :state="state" :schema="schema" @submit="submitted.push($event)" @error="failed.push($event)">
    <FormField label="Price" name="price" class="price"><InputNumber v-model="state.price" /></FormField>
    <FormField label="Stock" name="stock" class="stock"><InputNumber v-model="state.stock" /></FormField>
    <InputNumber v-model="state.weight" name="weight" class="weight" aria-label="Weight" />
    <Button type="submit" label="Save" />
  </Form>
</template>
