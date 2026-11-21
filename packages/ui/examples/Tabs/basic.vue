<script setup vapor lang="ts">
import { Tabs, type TabsItem } from '@weft/ui'

const items: TabsItem[] = [
  { label: 'Account', icon: 'user', content: 'Your name and email.' },
  { label: 'Password', icon: 'lock', content: 'Change your password.' },
  { label: 'Billing', icon: 'credit-card', badge: 2, content: 'Two invoices are due.' },
]
</script>

<template>
  <Tabs :items="items" aria-label="Settings" />
</template>
