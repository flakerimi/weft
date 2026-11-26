<script setup vapor lang="ts">
import { reactive, ref } from 'vue'
import { Button, Form, FormField, Input, InputNumber, type FieldSchema } from '../../../src/index.ts'

const props = defineProps<{ noSchema?: boolean }>()
// An optional number, a required one, a nullable number and a nullable text
// (pointers in Go), one named by its own prop outside a FormField, and a null
// the app set that no field holds.
const state = reactive({
  price: 5 as number | null,
  stock: 1 as number | null,
  sale: 3 as number | null,
  memo: 'hi' as string | null,
  nick: 'ada',
  weight: null as number | null,
  note: null as string | null,
})
const schema: FieldSchema<keyof typeof state> = {
  price: { min: 0 },
  stock: { required: true },
  sale: { min: 0, nullable: true },
  memo: { nullable: true },
}
const submitted = ref<Record<string, unknown>[]>([])
const failed = ref<Record<string, string>[]>([])
defineExpose({ state, submitted, failed })
</script>

<template>
  <Form :state="state" :schema="props.noSchema ? undefined : schema" @submit="submitted.push($event)" @error="failed.push($event)">
    <FormField label="Price" name="price" class="price"><InputNumber v-model="state.price" /></FormField>
    <FormField label="Stock" name="stock" class="stock"><InputNumber v-model="state.stock" /></FormField>
    <FormField label="Sale" name="sale" class="sale"><InputNumber v-model="state.sale" /></FormField>
    <FormField label="Memo" name="memo" class="memo"><Input v-model="state.memo" /></FormField>
    <FormField label="Nickname" name="nick" class="nick"><Input v-model="state.nick" /></FormField>
    <InputNumber v-model="state.weight" name="weight" class="weight" aria-label="Weight" />
    <Button type="submit" label="Save" />
  </Form>
</template>
