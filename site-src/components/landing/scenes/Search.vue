<script setup vapor lang="ts">
// A scene: a CommandPalette searched for "reb", its matches marked, each
// command's shortcut a Kbd.
import { CommandPalette, type CommandGroup } from '@weft/ui'

const groups: CommandGroup[] = [
  {
    id: 'devices',
    label: 'Devices',
    items: [
      { id: 'reboot-berisha', label: 'Reboot Berisha L.', icon: 'rotate-cw', kbds: ['command', 'R'] },
      { id: 'reboot-port', label: 'Reboot all on 1/3/1', icon: 'rotate-cw' },
    ],
  },
  {
    id: 'tickets',
    label: 'Tickets',
    items: [{ id: 't418', label: '#418 Slow after reboot', icon: 'ticket', suffix: '2 h ago' }],
  },
]
</script>

<template>
  <CommandPalette :groups="groups" search-term="reb" aria-label="Search Northline" class="scene-search" />
</template>
