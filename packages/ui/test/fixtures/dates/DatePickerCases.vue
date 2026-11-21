<script setup vapor lang="ts">
import { ref } from 'vue'
import type { DateInput } from '../../../src/dates/dates.ts'
import FormField from '../../../src/forms/FormField.vue'
import DatePicker from '../../../src/dates/DatePicker.vue'
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
}>(), { initial: undefined })
const input = ref<Input | undefined>(props.input)
const value = ref(((props.initial ?? null) as DateInput | null))
const open = ref(false)
const changes = ref<unknown[]>([])
const blurs = ref(0)
const submitted = ref<string | null>(null)
function onSubmit(event: Event) {
  event.preventDefault()
  submitted.value = new FormData(event.target as HTMLFormElement).get('start') as string | null
}
defineExpose({ input, value, open, changes, submitted, blurs })
</script>

<template>
  <WeftProvider :input="input" :locale="locale">
    <div :dir="dir">
      <button class="before">before</button>
      <form @submit="onSubmit">
        <FormField label="Start date" name="start" description="When it begins">
          <DatePicker
            v-model="value"
            v-model:open="open"
            :min="min"
            :max="max"
            :is-date-disabled="isDateDisabled"
            :value-format="valueFormat"
            :disabled="disabled"
            :readonly="readonly"
            :placeholder="placeholder"
            @change="changes.push($event)"
            @blur="blurs++"
          />
        </FormField>
        <button class="submit" type="submit">Save</button>
      </form>
      <button class="after">after</button>
    </div>
  </WeftProvider>
</template>
