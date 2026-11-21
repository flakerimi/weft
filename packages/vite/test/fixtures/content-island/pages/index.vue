<script setup vapor lang="ts">
import { Island } from '@weft/ui'
import Titles from '../components/Titles.vue'
</script>

<template>
  <main><Island when="load" :component="Titles" /></main>
</template>
