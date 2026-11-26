<script setup vapor lang="ts">
import { Island } from '@weft/ui'
import Choices from '../components/Choices.vue'
</script>

<template>
  <main><Island when="load" :component="Choices" /></main>
</template>
