<script setup vapor lang="ts">
import { Island } from '@weft/ui'
import Overlays from '../components/Overlays.vue'
</script>

<template>
  <main><Island when="load" :component="Overlays" /></main>
</template>
