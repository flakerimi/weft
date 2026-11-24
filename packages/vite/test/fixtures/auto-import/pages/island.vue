<script setup vapor lang="ts">
import Counter from '../components/Counter.vue'
</script>

<template>
  <main>
    <Island when="load" :component="Counter" :props="{ start: 2 }" />
  </main>
</template>
