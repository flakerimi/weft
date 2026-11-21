<script setup vapor lang="ts">
import { ref } from 'vue'
import { ToggleGroup, type ToggleGroupItem } from '@weft/ui'

const align = ref<string | number | null | (string | number)[]>('left')
const items: ToggleGroupItem[] = [
  { value: 'left', icon: 'align-left', ariaLabel: 'Left' },
  { value: 'center', icon: 'align-center', ariaLabel: 'Center' },
  { value: 'right', icon: 'align-right', ariaLabel: 'Right' },
]
</script>

<template>
  <ToggleGroup v-model="align" aria-label="Text alignment" :items="items" />
</template>
