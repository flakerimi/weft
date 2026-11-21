<script setup vapor lang="ts">
import { Island } from '@weft/ui'
import Shell from '../components/Shell.vue'
</script>

<template>
  <div><Island when="load" :component="Shell" /></div>
</template>
