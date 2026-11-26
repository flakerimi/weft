<script setup vapor lang="ts">
import { Island } from '@weft/ui'
import Login from '../components/Login.vue'
</script>

<template>
  <main><Island when="load" :component="Login" /></main>
</template>
