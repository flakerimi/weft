<script setup vapor lang="ts">
// Reads the window while it sets up, which only a browser has: the server render must fail.
const wide = window.innerWidth > 800
</script>

<template>
  <p>{{ wide ? 'wide' : 'narrow' }}</p>
</template>
