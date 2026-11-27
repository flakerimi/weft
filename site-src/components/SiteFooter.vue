<script setup vapor lang="ts">
import { Footer } from '@weft/ui'
import SiteMark from './SiteMark.vue'
import '../styles/site.css'
</script>

<template>
  <Footer>
    <template #leading><SiteMark /></template>
    <template #trailing>
      MIT licensed. © 2026 Basecode LLC. Built with Weft, served by its binary.
      <a href="https://github.com/flakerimi/weft">GitHub</a> · <a href="https://basecode.llc">basecode.llc</a>
    </template>
  </Footer>
</template>
