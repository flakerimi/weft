<script setup vapor lang="ts">
import { computed, ref } from 'vue'
import { Button, DropdownMenu, type MenuItem } from '@weft/ui'

const grid = ref(true)
const sort = ref('name')
const items = computed<MenuItem[][]>(() => [
  [
    { type: 'label', label: 'File' },
    { label: 'Rename', icon: 'pencil' },
    { label: 'Duplicate', icon: 'copy' },
    { label: 'Share', icon: 'share', children: [{ label: 'Copy link' }, { label: 'Email' }] },
  ],
  [
    { type: 'checkbox', label: 'Show grid', checked: grid.value, onUpdateChecked: (on) => (grid.value = on) },
    { type: 'radio', label: 'By name', checked: sort.value === 'name', onUpdateChecked: () => (sort.value = 'name') },
    { type: 'radio', label: 'By date', checked: sort.value === 'date', onUpdateChecked: () => (sort.value = 'date') },
  ],
  [{ label: 'Delete', icon: 'trash', tone: 'danger' }],
])
</script>

<template>
  <!-- On touch it opens as a bottom sheet with large items. -->
  <DropdownMenu :items="items">
    <Button label="Actions" variant="outline" tone="neutral" trailing-icon="chevron-down" />
  </DropdownMenu>
</template>
