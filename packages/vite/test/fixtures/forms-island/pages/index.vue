<script setup vapor lang="ts">
import { Island } from '@weft/ui'
import Forms from '../components/Forms.vue'
</script>

<template>
  <main><Island when="load" :component="Forms" /></main>
</template>
