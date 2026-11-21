<script setup vapor lang="ts">
import { AppShell } from '@weft/ui'
</script>

<template>
  <AppShell title="Acme" :items="[{ label: 'Home', to: '/' }]"><p>Shell page</p></AppShell>
</template>
