<script setup vapor lang="ts">
// Every @weft/ui component's first example (examples/<Name>/basic.vue), each
// in a card, so the controls above show all of them under every theme,
// scheme, input and surface at once.
import type { Component } from 'vue'
import { Card, Grid, Section } from '@weft/ui'

const found = import.meta.glob<{ default: Component }>('../../../packages/ui/examples/*/basic.vue', { eager: true })
const examples = Object.entries(found)
  .map(([path, mod]) => ({ name: /examples\/([^/]+)\/basic\.vue$/.exec(path)![1]!, component: mod.default }))
  .sort((a, b) => a.name.localeCompare(b.name))
</script>

<template>
  <Section title="Components" :description="`The first example of each of the ${examples.length} components.`">
    <Grid :cols="{ narrow: 1, regular: 2, wide: 3 }">
      <Card v-for="example in examples" :key="example.name" class="example" :data-component="example.name">
        <template #header>{{ example.name }}</template>
        <component :is="example.component" />
      </Card>
    </Grid>
  </Section>
</template>

<style>
/* AppShell is a whole page's frame, as tall as the viewport; in its card it gets a page of its own height. */
.example[data-component='AppShell'] .weft-app-shell { min-block-size: 0; block-size: 26rem; overflow: auto; }
.example[data-component='AppShell'] .weft-app-shell__sidebar { block-size: auto; }
</style>
