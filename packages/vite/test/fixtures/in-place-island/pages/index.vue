<script setup vapor lang="ts">
import { Island } from '@weft/ui'
import InPlace from '../components/InPlace.vue'
</script>

<template>
  <main><Island when="load" :component="InPlace" /></main>
</template>
