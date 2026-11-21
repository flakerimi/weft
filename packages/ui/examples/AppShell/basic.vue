<script setup vapor lang="ts">
import { AppShell, Button, Section, type NavItem } from '@weft/ui'

const items: NavItem[] = [
  { label: 'Home', to: '/', icon: 'house' },
  { label: 'Inbox', to: '/inbox', icon: 'inbox', badge: 3 },
  { label: 'Calendar', to: '/calendar', icon: 'calendar' },
  { label: 'Reports', to: '/reports', icon: 'chart-line' },
  { label: 'Projects', icon: 'folder', children: [{ label: 'Weft', to: '/projects/weft' }] },
  { label: 'Settings', to: '/settings', icon: 'settings' },
]
</script>

<template>
  <AppShell :items="items" title="Acme">
    <template #actions><Button icon="user" variant="ghost" tone="neutral" aria-label="Account" /></template>
    <Section title="Inbox">
      <p>Three new messages.</p>
    </Section>
  </AppShell>
</template>
