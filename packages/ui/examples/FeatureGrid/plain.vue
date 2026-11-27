<script setup vapor lang="ts">
import { FeatureGrid, type FeatureItem } from '@weft/ui'

const items: FeatureItem[] = [
  {
    icon: 'smartphone',
    title: 'Touch and pointer',
    description: 'Menus and pickers open bottom sheets on touch and anchored popovers on pointer.',
    to: '/docs/guides/adaptive',
    linkLabel: 'Adaptive model',
  },
  {
    icon: 'sliders-horizontal',
    title: 'Token themes',
    description: 'Every rule reads --weft-* custom properties with safe fallbacks.',
    to: '/docs/guides/themes',
    linkLabel: 'Theme tokens',
  },
  {
    icon: 'database',
    title: 'SQLite built in',
    description: 'Schema migrations and typed stores run at startup through database/sql.',
    to: '/docs/guides/data',
    linkLabel: 'Data guide',
  },
]
</script>

<template>
  <FeatureGrid
    variant="plain"
    title="Built for the long haul"
    :items="items"
  />
</template>
