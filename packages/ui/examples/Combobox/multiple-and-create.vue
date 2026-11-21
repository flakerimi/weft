<script setup vapor lang="ts">
import { ref } from 'vue'
import { Combobox, FormField } from '@weft/ui'

const labels = ref(['bug', 'docs', 'feature', 'help wanted', 'question'])
const chosen = ref<string[]>(['bug'])

// What was typed becomes a label, chosen at once.
function create(label: string) {
  labels.value = [...labels.value, label]
  chosen.value = [...chosen.value, label]
}
</script>

<template>
  <FormField label="Labels" hint="Backspace removes the last">
    <Combobox v-model="chosen" :items="labels" multiple create-item placeholder="Add labels" @create="create" />
  </FormField>
</template>
