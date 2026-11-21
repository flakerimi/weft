<script setup vapor lang="ts">
import { NavigationMenu, type NavItem } from '@weft/ui'

const items: NavItem[] = [
  { label: 'Guide', to: '/guide', icon: 'book-open' },
  {
    label: 'Components',
    icon: 'box',
    children: [
      { label: 'Button', to: '/components/button', description: 'Actions and links drawn as buttons.' },
      { label: 'Tabs', to: '/components/tabs', description: 'Views picked from a row of tabs.' },
    ],
  },
  { label: 'Help', to: '/help', icon: 'life-buoy' },
]
</script>

<template>
  <NavigationMenu :items="items" label="Docs" />
</template>
