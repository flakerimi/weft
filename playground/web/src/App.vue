<script setup vapor lang="ts">
import { computed, ref, watchEffect } from 'vue'
import { Card, Divider, Grid, Page, Section, Stack, WeftProvider, type InputMethod, type Surface } from '@weft/ui'
import Controls from './Controls.vue'
import Gallery from './Gallery.vue'
import Readout from './Readout.vue'
import { applyScheme, applyTheme, type Scheme, type ThemeName } from './themes.ts'

// Initial controls can come from the URL: ?theme=lisa&scheme=dark&input=touch&surface=landing
const q = new URLSearchParams(location.search)
const pick = <T extends string>(key: string, allowed: readonly T[], fallback: T): T => {
  const v = q.get(key) as T | null
  return v && allowed.includes(v) ? v : fallback
}
const theme = ref<ThemeName>(pick('theme', ['neutral', 'lisa'] as const, 'neutral'))
const scheme = ref<Scheme>(pick('scheme', ['system', 'light', 'dark'] as const, 'system'))
const input = ref<'auto' | InputMethod>(pick('input', ['auto', 'pointer', 'touch'] as const, 'auto'))
const surface = ref<Surface>(pick('surface', ['app', 'landing'] as const, 'app'))

watchEffect(() => applyTheme(theme.value))
watchEffect(() => applyScheme(scheme.value))

const inputOverride = computed(() => (input.value === 'auto' ? undefined : input.value))
const cards = ['Customers', 'Revenue', 'Open tickets', 'ONTs online', 'Churn', 'Plans']
</script>

<template>
  <WeftProvider :input="inputOverride" :surface="surface">
    <Page title="Weft playground" description="Every component, under every theme, scheme, input and surface.">
      <template #actions><Readout /></template>

      <Controls v-model:theme="theme" v-model:scheme="scheme" v-model:input="input" v-model:surface="surface" />

      <Gallery />

      <Divider />

      <Section title="Grid" description="Columns follow the grid's own width: 1 narrow, 2 regular, 3 wide.">
        <Grid>
          <Card v-for="name in cards" :key="name">
            <template #header>{{ name }}</template>
            <p>A card inside a grid.</p>
            <template #footer>Updated just now</template>
          </Card>
        </Grid>
      </Section>

      <Divider />

      <Section title="Stack" description="Rows and columns with the gap scale.">
        <Stack direction="row" :gap="2" wrap>
          <Card v-for="gap in [1, 2, 3, 4, 6, 8]" :key="gap" :padding="3">gap {{ gap }}</Card>
        </Stack>
        <Stack :gap="1">
          <Card :padding="2">column</Card>
          <Card :padding="2">of</Card>
          <Card :padding="2">cards</Card>
        </Stack>
      </Section>

      <Section title="Card">
        <Grid :cols="{ narrow: 1, regular: 2, wide: 2 }">
          <Card>
            <template #header>Plan</template>
            <template #actions><span class="pill">Pro</span></template>
            <p>Header, actions, body and footer.</p>
            <template #footer>Renews monthly</template>
          </Card>
          <Card :padding="6"><p>Body only, padding 6.</p></Card>
        </Grid>
      </Section>
    </Page>
  </WeftProvider>
</template>

<style>
.controls select { margin-inline-start: var(--weft-space-2); min-height: max(var(--weft-control-md, 36px), var(--weft-hit-min, 24px)); font: inherit; }
.pill { padding: 2px 8px; border-radius: 999px; background: var(--weft-color-primary-soft); color: var(--weft-color-text); }
.readout { color: var(--weft-color-text-muted); }
</style>
