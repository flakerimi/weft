<script setup vapor lang="ts">
// A scene: a maintenance window in a DateRangePicker, drawn open: its own
// Popover, in place (:portal="false"), holds its Calendar with the range.
import { DateRangePicker, FormField, Stack, type DateRange } from '@weft/ui'

const window: DateRange = { start: '2026-12-08', end: '2026-12-12' }
</script>

<template>
  <Stack class="scene-window">
    <FormField label="Maintenance window" name="window">
      <DateRangePicker :model-value="window" locale="en-GB" :number-of-months="1" :portal="false" default-open />
    </FormField>
  </Stack>
</template>
