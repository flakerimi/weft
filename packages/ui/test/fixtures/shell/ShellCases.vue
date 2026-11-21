<script setup vapor lang="ts">
import { ref } from 'vue'
import AppShell from '../../../src/shell/AppShell.vue'
import WeftProvider from '../../../src/adaptive/WeftProvider.vue'
import type { Input } from '../../../src/adaptive/detect.ts'
import type { NavItem } from '../../../src/navigation/nav.ts'

const props = withDefaults(defineProps<{ input?: Input; few?: boolean; persist?: string | false }>(), { persist: 'test:collapsed' })
const input = ref<Input | undefined>(props.input)
const chosen: string[] = []
const manyItems: NavItem[] = [
  { label: 'Home', to: '/', icon: 'house' },
  { label: 'Inbox', to: '/inbox', icon: 'inbox', badge: 2 },
  { label: 'Projects', icon: 'folder', children: [{ label: 'Weft', to: '/projects/weft' }] },
  { label: 'Calendar', to: '/calendar', icon: 'calendar' },
  { label: 'Reports', to: '/reports', icon: 'chart-line' },
  { label: 'Settings', to: '/settings', icon: 'settings' },
]
const fewItems: NavItem[] = [
  { label: 'Home', to: '/', icon: 'house' },
  { label: 'Inbox', to: '/inbox', icon: 'inbox' },
  { label: 'Settings', to: '/settings', icon: 'settings' },
]
defineExpose({ input, chosen })
</script>

<template>
  <WeftProvider :input="input">
    <AppShell :items="few ? fewItems : manyItems" title="Acme" :persist="persist" @select="(item) => chosen.push(item.label!)">
      <template #actions><button class="account">Account</button></template>
      <template #footer="{ collapsed }"><span class="user">{{ collapsed ? 'A' : 'Ada' }}</span></template>
      <p class="page">Page content</p>
    </AppShell>
  </WeftProvider>
</template>
