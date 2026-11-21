<script setup vapor lang="ts">
import { ref } from 'vue'
import { FormField, Listbox, type SelectItems } from '@weft/ui'

const chosen = ref<string[]>(['Rust'])
// Two groups, each under a heading.
const languages: SelectItems = [
  [{ type: 'label', label: 'Systems' }, 'C', 'C++', 'Go', 'Rust', 'Zig'],
  [{ type: 'label', label: 'Scripting' }, 'JavaScript', 'Lua', 'Python', 'Ruby'],
]
</script>

<template>
  <FormField label="Languages" description="Pick as many as you use.">
    <Listbox v-model="chosen" :items="languages" multiple filter placeholder="Filter languages…" />
  </FormField>
</template>
