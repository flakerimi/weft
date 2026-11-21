<script setup vapor lang="ts">
// Every selection component in one island: the server renders their fields
// and the always-shown lists (a Listbox long enough to draw only its first
// rows, a CommandPalette), none of the lists that open; the browser hydrates
// that markup in place (selection-island.test.ts), then they work.
import { ref } from 'vue'
import { Combobox, CommandPalette, FormField, InputTags, Listbox, Select, type CommandGroup } from '@weft/ui'

const fruit = ref('banana')
const person = ref<unknown>({ id: 2, label: 'Grace Hopper' })
const row = ref(3)
const tags = ref(['vapor'])
const ran = ref('')
const fruits = [
  { label: 'Apple', value: 'apple' },
  { label: 'Banana', value: 'banana' },
  { label: 'Cherry', value: 'cherry' },
]
const people = [
  { id: 1, label: 'Ada Lovelace' },
  { id: 2, label: 'Grace Hopper' },
]
const rows = Array.from({ length: 300 }, (_, i) => ({ label: `Row ${i + 1}`, value: i + 1 }))
const groups: CommandGroup[] = [{ id: 'go', label: 'Go to', items: [{ id: 'home', label: 'Home', onSelect: () => (ran.value = 'home') }] }]
</script>

<template>
  <div class="selection">
    <FormField label="Fruit" name="fruit"><Select v-model="fruit" :items="fruits" /></FormField>
    <FormField label="Person" name="person"><Combobox v-model="person" :items="people" by="id" /></FormField>
    <FormField label="Row" name="row"><Listbox v-model="row" :items="rows" value-key="value" /></FormField>
    <FormField label="Tags" name="tags"><InputTags v-model="tags" /></FormField>
    <CommandPalette :groups="groups" aria-label="Commands" />
    <output>{{ fruit }}|{{ row }}|{{ tags.join(',') }}|{{ ran }}</output>
  </div>
</template>
