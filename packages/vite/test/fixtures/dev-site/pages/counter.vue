<script setup vapor lang="ts">
import Counter from '../components/Counter.vue'
</script>

<template>
  <main>
    <h1>Counter</h1>
    <Island when="load" :component="Counter" :props="{ start: 2 }" />
  </main>
</template>
