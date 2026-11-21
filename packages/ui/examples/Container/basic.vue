<script setup vapor lang="ts">
import { Container, WeftProvider } from '@weft/ui'
</script>

<template>
  <!-- On a landing surface the content is centred at the page width. -->
  <WeftProvider surface="landing">
    <Container as="section">
      <h2>Every invoice, paid on time</h2>
      <p>Send, chase and reconcile from one place.</p>
    </Container>
  </WeftProvider>
</template>
