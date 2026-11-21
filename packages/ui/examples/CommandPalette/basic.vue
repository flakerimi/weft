<script setup vapor lang="ts">
import { ref } from 'vue'
import { CommandPalette, type CommandGroup } from '@weft/ui'

const last = ref('')
const run = (what: string) => () => (last.value = what)
const groups: CommandGroup[] = [
  {
    id: 'actions',
    label: 'Actions',
    items: [
      { id: 'new', label: 'New document', icon: 'file-plus', kbds: ['command', 'n'], onSelect: run('New document') },
      {
        id: 'theme',
        label: 'Change theme',
        icon: 'palette',
        placeholder: 'Choose a theme…',
        children: [
          { id: 'light', label: 'Light', icon: 'sun', onSelect: run('Light theme') },
          { id: 'dark', label: 'Dark', icon: 'moon', onSelect: run('Dark theme') },
        ],
      },
      { id: 'settings', label: 'Settings', icon: 'settings', suffix: 'Account and workspace', onSelect: run('Settings') },
    ],
  },
  {
    id: 'people',
    label: 'People',
    items: [
      { id: 'ada', label: 'Ada Lovelace', icon: 'user', keywords: ['analyst'], onSelect: run('Ada') },
      { id: 'grace', label: 'Grace Hopper', icon: 'user', keywords: ['compiler'], onSelect: run('Grace') },
    ],
  },
]
</script>

<template>
  <CommandPalette :groups="groups" recent aria-label="Commands" />
  <p>Last run: {{ last || 'nothing yet' }}</p>
</template>
