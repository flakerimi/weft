<script setup vapor lang="ts">
import { onMounted, onScopeDispose, ref } from 'vue'
import { Button, CommandPalette, Dialog, Kbd, type CommandGroup, type CommandItem } from '@weft/ui'

// A Cmd+K (Ctrl+K) launcher: the shortcut toggles a Dialog holding the
// palette. The Dialog's focus trap puts focus in the search field, Escape
// closes it, and running a command closes it too, focus going back to the
// button that opened it.
const open = ref(false)
const last = ref('')
const groups: CommandGroup[] = [
  {
    id: 'go',
    label: 'Go to',
    items: [
      { id: 'home', label: 'Home', icon: 'house' },
      { id: 'projects', label: 'Projects', icon: 'folder' },
      { id: 'billing', label: 'Billing', icon: 'credit-card' },
    ],
  },
  {
    id: 'actions',
    label: 'Actions',
    items: [
      { id: 'invite', label: 'Invite a teammate', icon: 'user-plus', kbds: ['command', 'i'] },
      { id: 'sign-out', label: 'Sign out', icon: 'log-out' },
    ],
  },
]

function onSelect(item: CommandItem) {
  last.value = item.label
  open.value = false
}

function onKey(event: KeyboardEvent) {
  if (event.key.toLowerCase() === 'k' && (event.metaKey || event.ctrlKey)) {
    event.preventDefault()
    open.value = !open.value
  }
}
onMounted(() => document.addEventListener('keydown', onKey))
onScopeDispose(() => {
  if (typeof document !== 'undefined') document.removeEventListener('keydown', onKey)
})
</script>

<template>
  <Dialog v-model:open="open" title="Search">
    <Button variant="outline" tone="neutral" icon="search">
      Search <Kbd value="command" size="sm" /><Kbd value="K" size="sm" />
    </Button>
    <template #body>
      <CommandPalette :groups="groups" recent recent-key="example-launcher" @select="onSelect" />
    </template>
  </Dialog>
  <p>Last run: {{ last || 'nothing yet' }}</p>
</template>
