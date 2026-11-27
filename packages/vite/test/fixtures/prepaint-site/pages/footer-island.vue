<script setup vapor lang="ts">
import { Island } from '@weft/ui'
import FooterIsland from '../components/FooterIsland.vue'
</script>

<template>
  <div><Island when="load" :component="FooterIsland" /></div>
</template>
