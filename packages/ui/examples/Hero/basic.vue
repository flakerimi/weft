<script setup vapor lang="ts">
import { Hero } from '@weft/ui'
</script>

<template>
  <Hero
    headline="One binary"
    title="Build Go and Vue apps together"
    description="Vapor components, a pre-rendered page build and a Go server that ships in a single executable."
    :actions="[
      { label: 'Get started', to: '/docs/getting-started/install' },
      { label: 'Browse components', to: '/components' },
    ]"
  />
</template>
