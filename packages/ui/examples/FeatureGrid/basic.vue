<script setup vapor lang="ts">
import { FeatureGrid, type FeatureItem } from '@weft/ui'

const items: FeatureItem[] = [
  {
    icon: 'zap',
    title: 'Vapor components',
    description: 'Compiled directly to DOM operations without a virtual DOM diff.',
  },
  {
    icon: 'package',
    title: 'Single Go binary',
    description: 'Pre-compressed HTML, assets and typed JSON routes in one executable.',
  },
  {
    icon: 'shield',
    title: 'Typed end to end',
    description: 'Go route structs and validation rules generate the TypeScript client and form schemas.',
  },
]
</script>

<template>
  <FeatureGrid
    headline="Architecture"
    title="Everything an app needs, nothing it does not"
    description="Four parts designed together so your pages and your API share one vocabulary."
    :items="items"
  />
</template>
