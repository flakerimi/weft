<script setup vapor lang="ts">
import { Island } from '@weft/ui'
import Barrel from '../components/Barrel.vue'
</script>

<template>
  <main><Island when="idle" :component="Barrel" /></main>
</template>
