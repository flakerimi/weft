<script setup vapor lang="ts">
import { Badge, Island } from '@weft/ui'
import Tally from '../components/Tally.vue'
</script>

<template>
  <main>
    <Badge>page</Badge>
    <Island when="load" :component="Tally" />
  </main>
</template>
