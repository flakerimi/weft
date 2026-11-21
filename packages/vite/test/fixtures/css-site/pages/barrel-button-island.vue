<script setup vapor lang="ts">
import { Island } from '@weft/ui'
import BarrelButton from '../components/BarrelButton.vue'
</script>

<template>
  <main><Island when="load" :component="BarrelButton" /></main>
</template>
