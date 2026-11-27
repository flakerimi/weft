<script setup vapor lang="ts">
import { ref } from 'vue'
import type { TimeInput, TimeRange } from '../../../src/dates/time.ts'
import FormField from '../../../src/forms/FormField.vue'
import InputTime from '../../../src/dates/InputTime.vue'
import WeftProvider from '../../../src/adaptive/WeftProvider.vue'
import type { Input } from '../../../src/adaptive/detect.ts'

const props = withDefaults(defineProps<{
  input?: Input
  dir?: 'ltr' | 'rtl'
  locale?: string
  initial?: unknown
  hourCycle?: 12 | 24
  granularity?: 'hour' | 'minute' | 'second'
  min?: string
  max?: string
  step?: { hour?: number; minute?: number; second?: number }
  valueFormat?: 'iso' | 'object'
  disabled?: boolean
  readonly?: boolean
  range?: boolean
  standalone?: boolean
}>(), { initial: undefined })
const input = ref<Input | undefined>(props.input)
const value = ref(((props.initial ?? null) as TimeInput | TimeRange | null))
const submitted = ref<string | null>(null)
function onSubmit(event: Event) {
  event.preventDefault()
  const data = new FormData(event.target as HTMLFormElement)
  submitted.value = props.range ? `${data.get('at[start]')}–${data.get('at[end]')}` : (data.get('at') as string | null)
}
defineExpose({ input, value, submitted })
</script>

<template>
  <WeftProvider :input="input" :locale="locale">
    <div :dir="dir">
      <InputTime v-if="standalone" v-model="value as never" :range="range" />
      <form v-else @submit="onSubmit">
        <FormField label="Start time" name="at" description="Local time">
          <InputTime
            v-model="value as never"
            :hour-cycle="hourCycle"
            :granularity="granularity"
            :min="min"
            :max="max"
            :step="step"
            :value-format="valueFormat"
            :disabled="disabled"
            :readonly="readonly"
            :range="range"
          />
        </FormField>
        <button class="submit" type="submit">Save</button>
      </form>
    </div>
  </WeftProvider>
</template>
