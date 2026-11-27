<script setup vapor lang="ts">
import { Island } from '@weft/ui'
import Late from '../components/Late.vue'
import Session from '../components/Session.vue'
</script>

<template>
  <main><Island when="load" :component="Session" /><Island when="load" :component="Late" /></main>
</template>
