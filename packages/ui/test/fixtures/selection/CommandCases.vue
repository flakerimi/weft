<script setup vapor lang="ts">
import { computed, ref } from 'vue'
import Dialog from '../../../src/overlays/Dialog.vue'
import CommandPalette from '../../../src/selection/CommandPalette.vue'
import WeftProvider from '../../../src/adaptive/WeftProvider.vue'
import type { Input } from '../../../src/adaptive/detect.ts'
import type { CommandGroup, CommandItem } from '../../../src/selection/command.ts'

const props = defineProps<{
  input?: Input
  inDialog?: boolean
  recent?: boolean | number
  recentKey?: string
  limit?: number
  loading?: boolean
  close?: boolean
  preserveGroupOrder?: boolean
  emptySlot?: boolean
  bare?: boolean
  disabled?: boolean
  many?: number
}>()
const log = ref<string[]>([])
const term = ref('')
const open = ref(false)
const closed = ref(0)
const act = (name: string) => () => log.value.push(name)
const groups = computed<CommandGroup[]>(() => [
  {
    id: 'actions',
    label: 'Actions',
    items: [
      { id: 'new-file', label: 'New file', icon: 'plus', kbds: ['meta', 'n'], onSelect: act('new-file') },
      { id: 'new-window', label: 'New window', suffix: 'Opens a window', onSelect: act('new-window') },
      { id: 'save', label: 'Save', disabled: true, onSelect: act('save') },
      { id: 'sync', label: 'Sync', loading: true, onSelect: act('sync') },
      {
        id: 'theme',
        label: 'Change theme',
        placeholder: 'Pick a theme…',
        children: [
          { id: 'light', label: 'Light', onSelect: act('light') },
          { id: 'dark', label: 'Dark', onSelect: act('dark') },
        ],
      },
      { id: 'docs', label: 'Open docs', to: '/docs' },
    ],
  },
  {
    id: 'people',
    label: 'People',
    items: Array.from({ length: props.many ?? 3 }, (_, i): CommandItem => ({ id: `p${i}`, label: ['Ada Lovelace', 'Grace Hopper', 'Alan Turing'][i] ?? `Person ${i}`, description: i === 0 ? 'Mathematician' : undefined, keywords: i === 1 ? ['navy'] : undefined, onSelect: act(`p${i}`) })),
  },
  {
    id: 'pinned',
    label: 'Pinned',
    ignoreFilter: true,
    postFilter: (t: string, items: CommandItem[]) => (t ? items.slice(0, 1) : items),
    items: [
      { id: 'pin1', label: 'Always here', onSelect: act('pin1') },
      { id: 'pin2', label: 'Also here', onSelect: act('pin2') },
    ],
  },
])
/** Without the group that ignores the filter, so a search can find nothing. */
const shown = computed(() => (props.bare || props.emptySlot ? groups.value.slice(0, 2) : groups.value))
function onSelect(item: CommandItem) {
  log.value.push(`select:${item.id}`)
  if (props.inDialog) open.value = false
}
defineExpose({ log, term, open, closed })
</script>

<template>
  <WeftProvider :input="input">
    <template v-if="inDialog">
      <Dialog v-model:open="open" title="Command palette">
        <button class="opener">Search</button>
        <template #body>
          <CommandPalette v-model:search-term="term" :groups="groups" @select="onSelect" />
        </template>
      </Dialog>
    </template>
    <CommandPalette
      v-else-if="!emptySlot"
      v-model:search-term="term"
      :groups="shown"
      :recent="recent"
      :recent-key="recentKey"
      :limit="limit"
      :loading="loading"
      :close="close"
      :preserve-group-order="preserveGroupOrder"
      :disabled="disabled"
      @select="onSelect"
      @update:open="(on: boolean) => !on && closed++"
    />
    <CommandPalette v-else :groups="shown">
      <template #empty="{ searchTerm }"><b class="none">Nothing for {{ searchTerm }}</b></template>
    </CommandPalette>
    <button class="after">after</button>
  </WeftProvider>
</template>
