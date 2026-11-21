<script setup vapor lang="ts">
import { Island } from '@weft/ui'
import Basics from '../components/Basics.vue'
</script>

<template>
  <main><Island when="load" :component="Basics" /></main>
</template>
