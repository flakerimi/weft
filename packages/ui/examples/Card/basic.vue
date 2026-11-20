<script setup vapor lang="ts">
import { Card } from '@weft/ui'
</script>

<template>
  <Card>
    <template #header>Pro plan</template>
    <template #actions><button type="button">Change</button></template>
    <p>10 seats, 100 GB storage.</p>
    <template #footer>Renews monthly</template>
  </Card>
</template>
