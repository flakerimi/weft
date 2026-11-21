<script setup vapor lang="ts">
import { Footer } from '@weft/ui'
import SiteMark from './SiteMark.vue'
import '../styles/site.css'
</script>

<template>
  <Footer>
    <template #leading><SiteMark /></template>
    <template #trailing>MIT licensed. This site is built with Weft and served by its binary.</template>
  </Footer>
</template>
