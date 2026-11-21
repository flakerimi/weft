<script setup vapor lang="ts">
import { Island } from '@weft/ui'
import Titled from '../components/Titled.vue'
</script>

<template>
  <main><Island when="load" :component="Titled" /></main>
</template>
