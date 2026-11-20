<script setup vapor lang="ts">
import { Island } from '@weft/ui'
// A component with a `name` but no SFC `__name`: the registry could not know it.
const Unnamed = { name: 'Unnamed', setup: () => [] }
</script>

<template>
  <Island when="load" :component="Unnamed" />
</template>
