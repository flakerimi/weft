<script setup vapor lang="ts">
// A vertical NavigationMenu on a page with no island: it imports Sheet,
// Tooltip and Badge for forms and items this page never draws.
import { NavigationMenu } from '@weft/ui'

const items = [[{ type: 'label' as const, label: 'Docs' }, { label: 'Install', to: '/install', active: true }, { label: 'Themes', to: '/themes' }]]
</script>

<template>
  <main><NavigationMenu :items="items" orientation="vertical" label="Docs" /></main>
</template>
