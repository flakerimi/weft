<script setup vapor lang="ts">
import { Island } from '@weft/ui'
import LabelSlot from '../components/LabelSlot.vue'
</script>

<template>
  <main><Island when="load" :component="LabelSlot" /></main>
</template>
