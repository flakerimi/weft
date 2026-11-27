<script setup vapor lang="ts">
// FormFields whose label slot holds components that take ids (a Tooltip, a
// Checkbox), over a control that is no labelable element and over one that
// is, then a field after them: the server draws a field's label after its
// control (label-slot-island-ssr.test.ts), and the island must still count
// every id as the browser does when it hydrates (label-slot-island.test.ts).
import { ref } from 'vue'
import { Button, Checkbox, FormField, Input, RadioGroup, Slider, Tooltip } from '@weft/ui'

const volume = ref(30)
const plan = ref<string | number | null>('free')
</script>

<template>
  <div class="label-slot">
    <FormField name="volume">
      <template #label>Volume <Tooltip text="How loud"><Button label="?" /></Tooltip></template>
      <Slider v-model="volume" />
    </FormField>
    <FormField name="plan">
      <template #label>Plan <Checkbox label="Remember" /></template>
      <RadioGroup v-model="plan" :items="['free', 'pro']" />
    </FormField>
    <FormField name="email">
      <template #label>Email <Tooltip text="Work address"><Button label="?" /></Tooltip></template>
      <Input type="email" />
    </FormField>
    <FormField label="Name" name="name"><Input /></FormField>
  </div>
</template>
