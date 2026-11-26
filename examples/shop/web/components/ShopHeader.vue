<script setup vapor lang="ts">
// The bar at the top of every page, drawn on the server; who is signed in is
// an island inside it, since pages are built with no session.
import SessionBar from './SessionBar.vue'
</script>

<template>
  <Header title="Shop">
    <Link to="/products">Products</Link>
    <template #actions>
      <Island when="load" :component="SessionBar" />
    </template>
  </Header>
</template>
