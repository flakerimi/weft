<script setup vapor lang="ts">
import { Cta } from '@weft/ui'
</script>

<template>
  <Cta
    headline="Ready in a minute"
    title="Create your first Weft app"
    description="One command scaffolds the Go module, the page build and the first route."
    :actions="[
      { label: 'Install Weft', to: '/docs/getting-started/install' },
      { label: 'View the docs', to: '/docs' },
    ]"
  />
</template>
