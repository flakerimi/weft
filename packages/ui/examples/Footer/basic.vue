<script setup vapor lang="ts">
import { Footer, Link } from '@weft/ui'
</script>

<template>
  <Footer>
    <template #leading><span>© 2026 Weft</span></template>
    <Link to="/privacy" standalone>Privacy</Link>
    <Link to="/terms" standalone>Terms</Link>
    <template #trailing><Link to="https://example.com" standalone>Status</Link></template>
  </Footer>
</template>
