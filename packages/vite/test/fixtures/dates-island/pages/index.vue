<script setup vapor lang="ts">
import { Island } from '@weft/ui'
import Dates from '../components/Dates.vue'
</script>

<template>
  <main><Island when="load" :component="Dates" /></main>
</template>
