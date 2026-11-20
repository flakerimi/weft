<script setup vapor lang="ts">
import { Island } from '@weft/ui'
import Rated from '../components/Rated.vue'
</script>

<template>
  <main><Island when="load" :component="Rated" /></main>
</template>
