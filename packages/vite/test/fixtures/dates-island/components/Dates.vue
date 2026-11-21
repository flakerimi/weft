<script setup vapor lang="ts">
// Every date and file component in one island, in en-GB: the server renders
// the Calendar's grid, the pickers' fields, the time's parts and the file
// zone (no calendar that opens); the browser hydrates that markup in place
// (dates-island.test.ts), then they work. The Calendar with no value draws
// the same first state wherever and whenever it renders, then shows today's
// month once mounted.
import { ref } from 'vue'
import { Calendar, DatePicker, DateRangePicker, FileUpload, FormField, InputTime, WeftProvider, type DateRange } from '@weft/ui'

const day = ref('2024-02-14')
const due = ref<string | null>('2024-03-01')
const stay = ref<DateRange | null>({ start: '2024-04-02', end: '2024-04-05' })
const at = ref<string | null>('09:30')
const files = ref<File[]>([])
</script>

<template>
  <WeftProvider locale="en-GB" input="pointer">
    <div class="dates">
      <Calendar v-model="day" aria-label="Day" />
      <Calendar class="loose" aria-label="Any day" />
      <FormField label="Due" name="due"><DatePicker v-model="due" /></FormField>
      <FormField label="Stay" name="stay"><DateRangePicker v-model="stay" /></FormField>
      <FormField label="At" name="at"><InputTime v-model="at" /></FormField>
      <FormField label="Files" name="files"><FileUpload v-model="files" multiple /></FormField>
      <output>{{ day }}|{{ due }}|{{ stay?.start }}–{{ stay?.end }}|{{ at }}|{{ files.length }}</output>
    </div>
  </WeftProvider>
</template>
