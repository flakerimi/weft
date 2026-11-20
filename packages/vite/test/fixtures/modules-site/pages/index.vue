<script setup vapor lang="ts">
import { Island } from '@weft/ui'
import Tinted from '../components/Tinted.vue'
</script>

<template>
  <main>
    <Island when="load" :component="Tinted" />
  </main>
</template>
