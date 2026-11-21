<script setup vapor lang="ts">
import { Grid, StatCard } from '@weft/ui'
</script>

<template>
  <Grid :cols="{ narrow: 1, regular: 3 }" :gap="4">
    <StatCard label="Revenue" value="€48,210" :delta="12.4" description="vs last month" icon="banknote">
      <template #sparkline>
        <svg viewBox="0 0 100 40" preserveAspectRatio="none" fill="none" stroke="currentColor" stroke-width="2"><polyline points="0,32 15,28 30,30 45,20 60,22 75,12 100,6" /></svg>
      </template>
    </StatCard>
    <StatCard label="Active customers" :value="1284" :delta="-2" description="vs last month" icon="users" />
    <StatCard label="Churn" value="1.8%" :delta="-0.4" delta-unit=" pts" invert description="vs last quarter" />
  </Grid>
</template>
