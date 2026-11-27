<script setup vapor lang="ts">
import { Cta } from '@weft/ui'
</script>

<template>
  <Cta
    orientation="horizontal"
    variant="soft"
    title="Move your back office to one binary"
    description="Keep your SQLite database, login and pages together without a separate Node server in production."
    :actions="[{ label: 'See the example shop', to: '/docs' }]"
  />
</template>
