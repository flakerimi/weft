<script setup vapor lang="ts">
import { NavigationMenu, type NavItem } from '@weft/ui'

const items: NavItem[][] = [
  [
    { label: 'Home', to: '/', icon: 'house' },
    { label: 'Inbox', to: '/inbox', icon: 'inbox', badge: 4 },
    { label: 'Projects', icon: 'folder', defaultOpen: true, children: [{ label: 'Weft', to: '/projects/weft' }, { label: 'Lisa', to: '/projects/lisa' }] },
  ],
  [
    { type: 'label', label: 'Account' },
    { label: 'Settings', to: '/settings', icon: 'settings' },
  ],
]
</script>

<template>
  <NavigationMenu :items="items" orientation="vertical" />
</template>
