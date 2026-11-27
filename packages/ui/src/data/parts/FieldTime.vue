<script setup vapor lang="ts">
// The time of a datetime field, a part only EntityDialog draws: an InputTime
// beside the field's DatePicker, inside the same FormField. The date takes
// the field's id and its label, so the time must not: it gets an id of its
// own and is named by the hidden label whose id it is given ("Starts,
// time"), while it shares the field's description, error, required mark and
// its blur and change, so the field checks it as one with the date. It never
// tells the field to draw its label as no <label>, which the date needs.
import { inject } from 'vue'
import InputTime from '../../dates/InputTime.vue'
import { fieldKey, provideField } from '../../forms/field.ts'
import { useId } from '../../headless/useId.ts'

const props = defineProps<{
  /** The id of the element that names the time. */
  labelId: string
  /** The time of day, `HH:mm`, or null while it is empty. */
  modelValue: string | null
}>()

const emit = defineEmits<{
  /** The time changed: `HH:mm`, or null while a part is empty. */
  'update:modelValue': [value: string | null]
}>()

const id = useId('time')
const field = inject(fieldKey, null)
if (field) {
  provideField({
    ...field,
    id,
    labelId: props.labelId,
    hasLabel: () => true,
    name: () => undefined,
    namedByLabel() {},
  })
}
</script>

<template>
  <InputTime :model-value="modelValue" granularity="minute" @update:model-value="emit('update:modelValue', ($event as string | null) ?? null)" />
</template>
