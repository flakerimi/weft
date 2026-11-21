<script setup vapor lang="ts">
import { Button, Card } from '@weft/ui'
</script>

<template>
  <Card>
    <template #header>Pro plan</template>
    <template #actions><Button label="Change" size="sm" variant="outline" tone="neutral" /></template>
    <p>10 seats, 100 GB storage.</p>
    <template #footer>Renews monthly</template>
  </Card>
</template>
