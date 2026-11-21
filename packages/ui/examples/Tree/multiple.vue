<script setup vapor lang="ts">
import { ref } from 'vue'
import { Tree, type TreeItem } from '@weft/ui'

const chosen = ref<string | string[] | null>([])
const items: TreeItem[] = [
  { label: 'Fruit', value: 'fruit', children: [{ label: 'Apple', value: 'apple' }, { label: 'Pear', value: 'pear' }] },
  { label: 'Vegetables', value: 'veg', children: [{ label: 'Leek', value: 'leek' }, { label: 'Kale', value: 'kale', disabled: true }] },
]
</script>

<template>
  <Tree v-model="chosen" :items="items" multiple :default-expanded="['fruit', 'veg']" aria-label="Groceries" />
</template>
