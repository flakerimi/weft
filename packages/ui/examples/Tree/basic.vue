<script setup vapor lang="ts">
import { ref } from 'vue'
import { Tree, type TreeItem } from '@weft/ui'

const selected = ref<string | string[] | null>(null)
const items: TreeItem[] = [
  {
    label: 'src',
    icon: 'folder',
    defaultExpanded: true,
    children: [
      { label: 'components', icon: 'folder', children: [{ label: 'Button.vue', icon: 'file' }, { label: 'Tabs.vue', icon: 'file' }] },
      { label: 'index.ts', icon: 'file-text' },
    ],
  },
  { label: 'package.json', icon: 'file-text' },
]
</script>

<template>
  <Tree v-model="selected" :items="items" aria-label="Files" />
</template>
