<script setup vapor lang="ts">
import { Breadcrumb, type BreadcrumbItem } from '@weft/ui'

const items: BreadcrumbItem[] = [
  { label: 'Home', to: '/', icon: 'house' },
  { label: 'Docs', to: '/docs' },
  { label: 'Components', to: '/docs/components' },
  { label: 'Breadcrumb' },
]
</script>

<template>
  <Breadcrumb :items="items" />
</template>
