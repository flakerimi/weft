<script setup vapor lang="ts">
// A FormField's label over a group control, drawn by the server: the label's
// `for` must name an element on the page.
import { CheckboxGroup, FormField, RadioGroup, Slider } from '@weft/ui'
</script>

<template>
  <FormField label="Plan" name="plan"><RadioGroup :items="['Starter', 'Team']" model-value="Team" /></FormField>
  <FormField label="Alerts" name="alerts"><CheckboxGroup :items="['Email', 'SMS']" :model-value="['Email']" /></FormField>
  <FormField label="Volume" name="volume"><Slider :model-value="40" /></FormField>
</template>
