<script setup vapor lang="ts">
import { ref } from 'vue'
import Calendar from '../../../src/dates/Calendar.vue'
import WeftProvider from '../../../src/adaptive/WeftProvider.vue'
import type { Input } from '../../../src/adaptive/detect.ts'
import type { CalendarDate } from '@internationalized/date'
import type { CalendarModel } from '../../../src/dates/dates.ts'

const props = withDefaults(defineProps<{
  input?: Input
  dir?: 'ltr' | 'rtl'
  locale?: string
  providerLocale?: string
  initial?: unknown
  range?: boolean
  multiple?: boolean
  min?: string
  max?: string
  isDateDisabled?: (date: CalendarDate) => boolean
  isDateUnavailable?: (date: CalendarDate) => boolean
  weekStartsOn?: 0 | 1 | 2 | 3 | 4 | 5 | 6
  numberOfMonths?: number
  fixedWeeks?: boolean
  weekNumbers?: boolean
  valueFormat?: 'iso' | 'object'
  disabled?: boolean
  readonly?: boolean
  focused?: string
  type?: 'date' | 'month' | 'year'
}>(), { initial: undefined, fixedWeeks: true })
const input = ref<Input | undefined>(props.input)
const value = ref(props.initial as CalendarModel | undefined)
const focusedDate = ref<string | undefined>(props.focused)
const changes = ref(0)
const months = ref(props.numberOfMonths)
defineExpose({ input, value, focusedDate, changes, months })
</script>

<template>
  <WeftProvider :input="input" :locale="providerLocale">
    <div :dir="dir">
      <button class="before">before</button>
      <Calendar
        v-model="value"
        v-model:focused-date="focusedDate as never"
        :locale="locale"
        :range="range"
        :multiple="multiple"
        :min="min"
        :max="max"
        :is-date-disabled="isDateDisabled"
        :is-date-unavailable="isDateUnavailable"
        :week-starts-on="weekStartsOn"
        :number-of-months="months"
        :fixed-weeks="fixedWeeks"
        :week-numbers="weekNumbers"
        :value-format="valueFormat"
        :disabled="disabled"
        :readonly="readonly"
        :type="type"
        @update:model-value="changes++"
      />
      <button class="after">after</button>
    </div>
  </WeftProvider>
</template>
