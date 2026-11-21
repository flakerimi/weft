<script setup vapor lang="ts">
import { ref } from 'vue'
import FormField from '../../../src/forms/FormField.vue'
import Combobox from '../../../src/selection/Combobox.vue'
import WeftProvider from '../../../src/adaptive/WeftProvider.vue'
import type { Input } from '../../../src/adaptive/detect.ts'
import type { SelectItem, SelectItems } from '../../../src/selection/items.ts'
import type { CreateItem } from '../../../src/selection/listbox.ts'

const props = defineProps<{
  input?: Input
  items?: SelectItems
  trigger?: 'input' | 'button'
  mode?: 'combobox' | 'autocomplete'
  multiple?: boolean
  createItem?: CreateItem
  clear?: boolean
  disabled?: boolean
  async?: boolean
  emptySlot?: boolean
  errorSlot?: boolean
  debounce?: number
  initial?: unknown
}>()
const input = ref<Input | undefined>(props.input)
const fruit: SelectItems = [
  { label: 'Apple', id: 1 },
  { label: 'Apricot', id: 2 },
  { label: 'Banana', id: 3 },
  { label: 'Blueberry', id: 4, disabled: true },
  { label: 'Cherry', id: 5 },
]
const value = ref<unknown>(props.initial ?? (props.multiple ? [] : undefined))
const open = ref(false)
const term = ref('')
const log = ref<string[]>([])
const submitted = ref(0)
/** The pending loads, each resolved by the test: `calls[i].resolve(items)`. */
const calls: { term: string; signal: AbortSignal; resolve: (items: SelectItem[]) => void; reject: (error: unknown) => void }[] = []
function load(t: string, ctx: { signal: AbortSignal }) {
  return new Promise<SelectItem[]>((resolve, reject) => calls.push({ term: t, signal: ctx.signal, resolve, reject }))
}
defineExpose({ input, value, open, term, log, calls, submitted })
</script>

<template>
  <WeftProvider :input="input">
    <form @submit.prevent="submitted++">
      <FormField label="Fruit" name="fruit">
        <Combobox
          v-if="!emptySlot"
          v-model="value"
          v-model:open="open"
          v-model:search-term="term"
          :items="async ? undefined : (items ?? fruit)"
          :load="async ? load : undefined"
          :debounce="async ? (debounce ?? 0) : undefined"
          :trigger="trigger"
          :mode="mode"
          :multiple="multiple"
          :create-item="createItem"
          :clear="clear"
          :disabled="disabled"
          value-key="id"
          placeholder="Search fruit"
          @create="(t: string) => log.push(`create:${t}`)"
          @clear="log.push('clear')"
          @remove-tag="(v: unknown) => log.push(`remove:${v}`)"
          @error="(e: unknown) => log.push(`error:${(e as Error).message}`)"
        >
          <template v-if="errorSlot" #error="{ error }"><b class="failed">Sorry: {{ (error as Error).message }}</b></template>
        </Combobox>
        <Combobox v-else v-model="value" :items="fruit" value-key="id">
          <template #empty="{ searchTerm }"><b class="none">No fruit like {{ searchTerm }}</b></template>
        </Combobox>
      </FormField>
    </form>
    <button class="after">after</button>
  </WeftProvider>
</template>
