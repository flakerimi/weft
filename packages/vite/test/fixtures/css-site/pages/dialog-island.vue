<script setup vapor lang="ts">
import { Island } from '@weft/ui'
import Opener from '../components/Opener.vue'
</script>

<template>
  <main><Island when="load" :component="Opener" /></main>
</template>
