<script setup vapor lang="ts">
import { ref } from 'vue'
import FormField from '../../../src/forms/FormField.vue'
import Listbox from '../../../src/selection/Listbox.vue'
import WeftProvider from '../../../src/adaptive/WeftProvider.vue'
import type { Input } from '../../../src/adaptive/detect.ts'
import type { SelectItems } from '../../../src/selection/items.ts'
import type { CreateItem } from '../../../src/selection/listbox.ts'

const props = defineProps<{
  input?: Input
  items?: SelectItems
  multiple?: boolean
  filter?: boolean
  loading?: boolean
  disabled?: boolean
  createItem?: CreateItem
  emptySlot?: boolean
  initial?: unknown
}>()
const fruit: SelectItems = ['Apple', 'Banana', { label: 'Blueberry', disabled: true }, 'Cherry', 'Date']
const value = ref<unknown>(props.initial ?? (props.multiple ? [] : undefined))
const term = ref('')
const created = ref<string[]>([])
defineExpose({ value, term, created })
</script>

<template>
  <WeftProvider :input="input">
    <FormField label="Fruit" name="fruit" hint="Pick one">
      <Listbox
        v-if="!emptySlot"
        v-model="value"
        v-model:search-term="term"
        :items="items ?? fruit"
        :multiple="multiple"
        :filter="filter"
        :loading="loading"
        :disabled="disabled"
        :create-item="createItem"
        @create="(t: string) => created.push(t)"
      />
      <Listbox v-else v-model="value" v-model:search-term="term" :items="items ?? fruit" filter>
        <template #empty="{ searchTerm }"><b class="none">Nothing for {{ searchTerm }}</b></template>
      </Listbox>
    </FormField>
    <button class="after">after</button>
  </WeftProvider>
</template>
