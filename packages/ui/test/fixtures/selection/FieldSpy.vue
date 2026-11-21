<script setup vapor lang="ts">
// A stand-in FormField that counts what its control reports: blurs and changes.
import { ref } from 'vue'
import { provideField } from '../../../src/forms/field.ts'
import Combobox from '../../../src/selection/Combobox.vue'
import Select from '../../../src/selection/Select.vue'
import WeftProvider from '../../../src/adaptive/WeftProvider.vue'
import type { Input } from '../../../src/adaptive/detect.ts'

const props = defineProps<{ control: 'select' | 'combobox-button'; input?: Input }>()
const blurs = ref(0)
const inputs = ref(0)
provideField({
  id: 'spy',
  labelId: 'spy-label',
  name: () => 'spy',
  describedBy: () => undefined,
  invalid: () => false,
  required: () => false,
  disabled: () => false,
  size: () => undefined,
  blur: () => blurs.value++,
  input: () => inputs.value++,
  namedByLabel: () => {},
})
const items = ['One', 'Two']
defineExpose({ blurs, inputs })
</script>

<template>
  <WeftProvider :input="props.input">
    <Select v-if="control === 'select'" :items="items" value-key="" />
    <Combobox v-else :items="items" trigger="button" />
    <button class="after">after</button>
  </WeftProvider>
</template>
