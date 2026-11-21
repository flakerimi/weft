<script setup vapor lang="ts">
import { ref } from 'vue'
import { ToggleGroup, type ToggleGroupItem } from '@weft/ui'

const styles = ref<string | number | null | (string | number)[]>(['bold'])
const items: ToggleGroupItem[] = [
  { value: 'bold', icon: 'bold', label: 'Bold' },
  { value: 'italic', icon: 'italic', label: 'Italic' },
  { value: 'underline', icon: 'underline', label: 'Underline' },
]
</script>

<template>
  <ToggleGroup v-model="styles" multiple attached variant="outline" aria-label="Text style" :items="items" />
</template>
