<script setup vapor lang="ts">
import { ref } from 'vue'
import WeftProvider from '../../../src/adaptive/WeftProvider.vue'
import EntityDialog from '../../../src/data/EntityDialog.vue'
import Input from '../../../src/forms/Input.vue'
import type { EntityDialogLabels, EntityFieldConfig, EntitySchema } from '../../../src/data/entity.ts'

export interface Item {
  id: number
  name: string
  price: number
  sale: number | null
  status: string
  category?: string
  email?: string
  website?: string
  notes?: string
  active?: boolean
}

export type ItemInput = Omit<Item, 'id'>

const props = withDefaults(
  defineProps<{
    open?: boolean
    record?: Item | null
    schema: EntitySchema
    create?: (body: ItemInput) => Promise<Item>
    update?: (id: number, body: Partial<ItemInput>) => Promise<Item>
    idKey?: string
    labelKey?: string
    name?: string
    title?: string
    description?: string
    defaults?: Partial<ItemInput>
    fields?: Record<string, EntityFieldConfig>
    labels?: Partial<EntityDialogLabels>
    customBody?: boolean
    customNameSlot?: boolean
  }>(),
  {
    open: false,
    record: null,
    create: undefined,
    update: undefined,
    idKey: undefined,
    labelKey: undefined,
    name: undefined,
    title: undefined,
    description: undefined,
    defaults: undefined,
    fields: undefined,
    labels: undefined,
    customBody: false,
    customNameSlot: false,
  },
)

const isOpen = ref(props.open)
const currentRecord = ref<Item | null>(props.record ?? null)
const savedRecords = ref<Item[]>([])
const errors = ref<unknown[]>([])

defineExpose({ isOpen, currentRecord, savedRecords, errors })
</script>

<template>
  <WeftProvider>
    <EntityDialog
      v-model:open="isOpen"
      :record="currentRecord"
      :schema="schema"
      :create="create"
      :update="update"
      :id-key="idKey"
      :label-key="labelKey"
      :name="name"
      :title="title"
      :description="description"
      :defaults="defaults"
      :fields="fields"
      :labels="labels"
      @saved="savedRecords.push($event)"
      @error="errors.push($event)"
    >
      <template v-if="customBody" #default="{ state }">
        <div class="custom-body">
          <Input :model-value="String(state.name ?? '')" @update:model-value="state.name = String($event ?? '')" />
        </div>
      </template>
      <template v-if="customNameSlot" #name-field="{ state }">
        <div class="custom-name-field">
          <Input :model-value="String(state.name ?? '')" @update:model-value="state.name = String($event ?? '')" />
        </div>
      </template>
    </EntityDialog>
  </WeftProvider>
</template>
