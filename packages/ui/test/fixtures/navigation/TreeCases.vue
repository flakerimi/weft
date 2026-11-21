<script setup vapor lang="ts">
import { ref } from 'vue'
import Tree from '../../../src/navigation/Tree.vue'
import WeftProvider from '../../../src/adaptive/WeftProvider.vue'
import type { Input } from '../../../src/adaptive/detect.ts'
import type { TreeItem } from '../../../src/navigation/tree.ts'

const props = defineProps<{ multiple?: boolean; big?: boolean; input?: Input; dir?: 'ltr' | 'rtl' }>()
const selected = ref<string | string[] | null>(props.multiple ? [] : null)
const expanded = ref<string[]>(['src'])
const toggled: string[] = []
const small: TreeItem[] = [
  {
    label: 'src',
    value: 'src',
    icon: 'folder',
    children: [
      { label: 'app.ts', value: 'app' },
      { label: 'components', value: 'components', children: [{ label: 'Button.vue', value: 'button' }, { label: 'Card.vue', value: 'card' }] },
      { label: 'locked.ts', value: 'locked', disabled: true },
    ],
  },
  { label: 'docs', value: 'docs', children: [{ label: 'guide.md', value: 'guide' }] },
  { label: 'package.json', value: 'package' },
]
const big: TreeItem[] = [
  { label: 'rows', value: 'rows', children: Array.from({ length: 500 }, (_, i) => ({ label: `Row ${i + 1}`, value: `r${i + 1}` })) },
]
const items = props.big ? big : small
if (props.big) expanded.value = ['rows']
defineExpose({ selected, expanded, toggled })
</script>

<template>
  <WeftProvider :input="input">
    <div :dir="dir">
      <Tree
        v-model="selected"
        v-model:expanded="expanded"
        :items="items"
        :multiple="multiple"
        aria-label="Files"
        @toggle="(item, open) => toggled.push(`${item.value}:${open}`)"
      />
    </div>
  </WeftProvider>
</template>
