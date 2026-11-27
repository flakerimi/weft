<script setup vapor lang="ts">
import { Card, Hero } from '@weft/ui'
</script>

<template>
  <Hero
    orientation="horizontal"
    headline="Zero virtual DOM"
    title="Pages that ship only what they render"
    description="Static sections render on the server and ship zero JavaScript; interactive islands hydrate when they enter view."
    :actions="[{ label: 'Read the guide', to: '/docs/guides/pages-and-islands' }]"
  >
    <Card>
      <template #header>weft build</template>
      <p>dist/pages: 14 routes pre-rendered (gzip + brotli)</p>
    </Card>
  </Hero>
</template>
