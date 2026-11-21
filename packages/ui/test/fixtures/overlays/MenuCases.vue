<script setup vapor lang="ts">
import { computed, ref } from 'vue'
import DropdownMenu from '../../../src/overlays/DropdownMenu.vue'
import WeftProvider from '../../../src/adaptive/WeftProvider.vue'
import type { Input } from '../../../src/adaptive/detect.ts'
import type { MenuItem } from '../../../src/overlays/menu.ts'

const props = defineProps<{ input?: Input; dir?: 'ltr' | 'rtl' }>()
const input = ref<Input | undefined>(props.input)
const open = ref(false)
const log = ref<string[]>([])
const grid = ref(false)
const sort = ref('name')
const items = computed<MenuItem[][]>(() => [
  [
    { type: 'label', label: 'File' },
    { label: 'New', icon: 'plus', kbds: ['meta', 'n'], description: 'An empty file', onSelect: () => log.value.push('new') },
    { label: 'Open', disabled: true, onSelect: () => log.value.push('open') },
    { label: 'Rename', onSelect: () => log.value.push('rename') },
    { label: 'Pin', onSelect: (e) => { e.preventDefault(); log.value.push('pin') } },
  ],
  [
    { type: 'checkbox', label: 'Show grid', checked: grid.value, onUpdateChecked: (v) => (grid.value = v) },
    { type: 'radio', label: 'By name', checked: sort.value === 'name', onUpdateChecked: () => (sort.value = 'name') },
    { type: 'radio', label: 'By date', checked: sort.value === 'date', onUpdateChecked: () => (sort.value = 'date') },
  ],
  [
    {
      label: 'Share',
      children: [
        { label: 'Copy link', onSelect: () => log.value.push('copy') },
        { label: 'Email', onSelect: () => log.value.push('email') },
      ],
    },
    { label: 'Delete', tone: 'danger', onSelect: () => log.value.push('delete') },
    { label: 'Sync', loading: true, color: 'warning', onSelect: () => log.value.push('sync') },
  ],
])
defineExpose({ input, open, log, grid, sort })
</script>

<template>
  <WeftProvider :input="input">
    <div :dir="dir">
      <DropdownMenu v-model:open="open" :items="items">
        <button class="trigger">Actions</button>
      </DropdownMenu>
      <button class="after">after</button>
    </div>
  </WeftProvider>
</template>
