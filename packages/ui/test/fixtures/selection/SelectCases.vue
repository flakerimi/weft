<script setup vapor lang="ts">
import { ref } from 'vue'
import FormField from '../../../src/forms/FormField.vue'
import Select from '../../../src/selection/Select.vue'
import WeftProvider from '../../../src/adaptive/WeftProvider.vue'
import type { Input } from '../../../src/adaptive/detect.ts'
import type { SelectItems } from '../../../src/selection/items.ts'

const props = defineProps<{
  input?: Input
  dir?: 'ltr' | 'rtl'
  items?: SelectItems
  multiple?: boolean
  disabled?: boolean
  loading?: boolean
  initial?: unknown
}>()
const input = ref<Input | undefined>(props.input)
const fruit: SelectItems = [
  { label: 'Apple', value: 'apple' },
  { label: 'Banana', value: 'banana' },
  { label: 'Blueberry', value: 'blueberry', disabled: true },
  { label: 'Cherry', value: 'cherry' },
  { label: 'Date', value: 'date' },
]
const value = ref<unknown>(props.initial ?? (props.multiple ? [] : undefined))
const open = ref(false)
const submitted = ref<string[]>([])
function onSubmit(event: Event) {
  event.preventDefault()
  submitted.value = new FormData(event.target as HTMLFormElement).getAll('fruit').map(String)
}
defineExpose({ input, value, open, submitted })
</script>

<template>
  <WeftProvider :input="input">
    <div :dir="dir">
      <button class="before">before</button>
      <form @submit="onSubmit">
        <FormField label="Fruit" name="fruit" description="Your favourite">
          <Select
            v-model="value"
            v-model:open="open"
            :items="items ?? fruit"
            :multiple="multiple"
            :disabled="disabled"
            :loading="loading"
            placeholder="Pick a fruit"
          />
        </FormField>
        <button class="submit" type="submit">Save</button>
      </form>
      <button class="after">after</button>
    </div>
  </WeftProvider>
</template>
