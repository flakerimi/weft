<script setup vapor lang="ts">
import { ref } from 'vue'
import { Combobox, FormField } from '@weft/ui'

const person = ref<{ id: number; label: string } | undefined>()
const people = [
  { id: 1, label: 'Ada Lovelace', description: 'Analyst' },
  { id: 2, label: 'Grace Hopper', description: 'Rear admiral' },
  { id: 3, label: 'Alan Turing', description: 'Mathematician' },
  { id: 4, label: 'Katherine Johnson', description: 'Orbital mechanics' },
  { id: 5, label: 'Edsger Dijkstra', description: 'Shortest paths' },
]
</script>

<template>
  <!-- Type to filter; on touch the list opens as a bottom sheet with its own search field. -->
  <FormField label="Assignee">
    <Combobox v-model="person" :items="people" by="id" placeholder="Search people" clear />
  </FormField>
</template>
