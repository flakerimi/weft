<script setup vapor lang="ts">
// Reads the document while it sets up, which only a browser has: the server render must fail.
const title = document.title
</script>

<template>
  <p>{{ title }}</p>
</template>
