<script setup vapor lang="ts">
import { Card, Stack } from '@weft/ui'

const theme = defineModel<string>('theme', { required: true })
const scheme = defineModel<string>('scheme', { required: true })
const input = defineModel<string>('input', { required: true })
const surface = defineModel<string>('surface', { required: true })
</script>

<template>
  <Card class="controls">
    <template #header>Controls</template>
    <Stack direction="row" :gap="4" wrap>
      <label>Theme
        <select v-model="theme"><option value="neutral">neutral</option><option value="lisa">lisa</option></select>
      </label>
      <label>Scheme
        <select v-model="scheme">
          <option value="system">system</option><option value="light">light</option><option value="dark">dark</option>
        </select>
      </label>
      <label>Input
        <select v-model="input">
          <option value="auto">auto</option><option value="pointer">pointer</option><option value="touch">touch</option>
        </select>
      </label>
      <label>Surface
        <select v-model="surface"><option value="app">app</option><option value="landing">landing</option></select>
      </label>
    </Stack>
  </Card>
</template>
