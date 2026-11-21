<script setup vapor lang="ts">
import { Island } from '@weft/ui'
import Selection from '../components/Selection.vue'
</script>

<template>
  <main><Island when="load" :component="Selection" /></main>
</template>
