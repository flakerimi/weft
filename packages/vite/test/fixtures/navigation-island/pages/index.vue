<script setup vapor lang="ts">
import { Island } from '@weft/ui'
import Navigation from '../components/Navigation.vue'
</script>

<template>
  <div><Island when="load" :component="Navigation" /></div>
</template>
