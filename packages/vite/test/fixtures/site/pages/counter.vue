<script setup vapor lang="ts">
import { Island } from '@weft/ui'
import Counter from '../components/Counter.vue'
</script>

<template>
  <main>
    <h1>Counter</h1>
    <Island when="visible" :component="Counter" :props="{ start: 2 }" />
  </main>
</template>
