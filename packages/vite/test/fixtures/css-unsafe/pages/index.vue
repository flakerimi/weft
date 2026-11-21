<script setup vapor lang="ts">
// The build test writes styles/bad.css for each case: a url() the build must refuse.
import '../styles/bad.css'
</script>

<template>
  <main><h1>Unsafe</h1></main>
</template>
