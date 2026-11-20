<script setup vapor lang="ts">
import { Card, Island, Page, Section, Stack, useHead } from '@weft/ui'
import CopyCommand from '../components/CopyCommand.vue'

useHead({
  title: 'Weft — one UI system for Go and Vue apps',
  description:
    'Vapor-first Vue components that adapt to touch and pointer, landing and app, shipped inside a single Go binary.',
})

const install = 'curl -fsSL https://weft.base.al/install | sh'

const features = [
  { title: 'Yours, all the way down', text: "Every component, API and style is Weft's. No wrapped library to wait on." },
  { title: 'Adaptive by default', text: 'Written once; sized and shaped for touch or pointer, landing or app.' },
  { title: 'Go-first', text: 'One binary serves your API and embeds your Vue app, pre-compressed.' },
  { title: 'Built for agents', text: 'Big building blocks and one way to do each thing.' },
]
</script>

<template>
  <div class="site" data-weft-surface="landing">
    <Page>
      <template #header>
        <Stack :gap="4" align="start">
          <span class="site-tag">pre-alpha</span>
          <h1 class="site-title">weft</h1>
          <p class="site-lead">One UI system for Go and Vue apps.</p>
          <p class="site-sub">
            Vapor-first components that know whether they're under a finger or a mouse, on a landing page or in an
            app — shipped inside a single Go binary.
          </p>
        </Stack>
      </template>

      <Section title="Install" description="One file, checked against its SHA-256, put in ~/.local/bin.">
        <Stack :gap="3">
          <Island when="idle" :component="CopyCommand" :props="{ command: install }" />
          <p class="site-note">
            Already installed? <code>weft update</code> moves you to the newest release;
            <code>weft update --check</code> only looks. Set <code>WEFT_BIN_DIR</code> to install elsewhere,
            <code>WEFT_VERSION</code> to pin one.
          </p>
        </Stack>
      </Section>

      <Section title="Why Weft">
        <div class="site-features">
          <Card v-for="f in features" :key="f.title" as="article">
            <template #header>{{ f.title }}</template>
            <p class="site-card-text">{{ f.text }}</p>
          </Card>
        </div>
      </Section>

      <footer class="site-footer">
        This page is built with Weft: rendered at build time, one island for the copy button, served by the Weft
        binary. MIT licensed.
      </footer>
    </Page>
  </div>
</template>

<style>
.site {
  --weft-page-max: 960px;
  --site-accent: var(--weft-color-primary);
  min-height: 100vh;
  font-size: var(--weft-text-base);
}
@media (prefers-color-scheme: dark) {
  .site { --site-accent: var(--weft-color-primary-soft); }
}
.site .weft-page { padding-block: clamp(40px, 9vw, 96px) var(--weft-space-8); }
.site code { font-family: var(--weft-font-mono); font-size: 0.9em; }
.site-tag {
  font: 0.75em/1.6 var(--weft-font-mono);
  padding: 2px 10px;
  border: 1px solid var(--site-accent);
  border-radius: 999px;
  color: var(--site-accent);
}
.site-title {
  font-size: clamp(56px, 12vw, 96px);
  font-weight: var(--weft-weight-title);
  letter-spacing: -0.045em;
  line-height: 0.9;
}
.site-lead { font-size: 1.5em; font-weight: var(--weft-weight-heading); line-height: 1.3; }
.site-sub, .site-note, .site-footer { color: var(--weft-color-text-muted); max-width: 40em; }
.site-note, .site-footer { font-size: 0.85em; }
.site-features {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(min(100%, 200px), 1fr));
  gap: var(--weft-space-4);
}
.site-features .weft-card__title { color: var(--site-accent); }
.site-card-text { color: var(--weft-color-text-muted); font-size: 0.9em; }
.site-footer { max-width: none; border-top: 1px solid var(--weft-color-border); padding-top: var(--weft-space-6); }
</style>
