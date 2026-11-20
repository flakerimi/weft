<script setup vapor lang="ts">
import { Card, Grid } from '@weft/ui'
</script>

<template>
  <Grid :cols="{ narrow: 2, wide: 4 }" :gap="3">
    <Card v-for="n in 8" :key="n" :padding="3">Tile {{ n }}</Card>
  </Grid>
</template>
