<script setup vapor lang="ts">
// A scene: a maintenance window picked with a DateRangePicker, its Calendar
// drawn as the picker shows it open.
import { Calendar, DateRangePicker, FormField, type DateRange } from '@weft/ui'

const window: DateRange = { start: '2026-12-08', end: '2026-12-12' }
</script>

<template>
  <div class="scene-window">
    <FormField label="Maintenance window" name="window"><DateRangePicker :model-value="window" locale="en-GB" /></FormField>
    <div class="scene-window__panel"><Calendar :model-value="window" range locale="en-GB" /></div>
  </div>
</template>
