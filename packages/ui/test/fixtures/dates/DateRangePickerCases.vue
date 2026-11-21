<script setup vapor lang="ts">
import { ref } from 'vue'
import type { DateRange } from '../../../src/dates/dates.ts'
import FormField from '../../../src/forms/FormField.vue'
import DateRangePicker from '../../../src/dates/DateRangePicker.vue'
import WeftProvider from '../../../src/adaptive/WeftProvider.vue'
import type { Input } from '../../../src/adaptive/detect.ts'
import type { CalendarDate } from '@internationalized/date'

const props = withDefaults(defineProps<{
  input?: Input
  dir?: 'ltr' | 'rtl'
  locale?: string
  initial?: unknown
  min?: string
  max?: string
  isDateDisabled?: (date: CalendarDate) => boolean
  valueFormat?: 'iso' | 'object'
  disabled?: boolean
  readonly?: boolean
  placeholder?: string
  numberOfMonths?: number
}>(), { initial: undefined })
const input = ref<Input | undefined>(props.input)
const value = ref(((props.initial ?? null) as DateRange | null))
const open = ref(false)
const changes = ref<unknown[]>([])
const submitted = ref<string[]>([])
function onSubmit(event: Event) {
  event.preventDefault()
  const data = new FormData(event.target as HTMLFormElement)
  submitted.value = [data.get('stay[start]') as string, data.get('stay[end]') as string]
}
defineExpose({ input, value, open, changes, submitted })
</script>

<template>
  <WeftProvider :input="input" :locale="locale">
    <div :dir="dir">
      <button class="before">before</button>
      <form @submit="onSubmit">
        <FormField label="Stay" name="stay" description="Arrival and departure">
          <DateRangePicker
            v-model="value"
            v-model:open="open"
            :min="min"
            :max="max"
            :is-date-disabled="isDateDisabled"
            :value-format="valueFormat"
            :disabled="disabled"
            :readonly="readonly"
            :placeholder="placeholder"
            :number-of-months="numberOfMonths"
            @change="changes.push($event)"
          />
        </FormField>
        <button class="submit" type="submit">Save</button>
      </form>
      <button class="after">after</button>
    </div>
  </WeftProvider>
</template>
