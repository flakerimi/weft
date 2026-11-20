<script setup vapor lang="ts">
import { Card, Island, Page, Section, Stack, useHead } from '@weft/ui'
import content from '../../content.json'
import Counter from '../components/Counter.vue'

useHead({ title: content.title, description: content.description })
</script>

<template>
  <div class="bench" data-weft-surface="landing">
    <Page>
      <template #header>
        <Stack :gap="4" align="start">
          <span class="bench-tag">{{ content.hero.tag }}</span>
          <h1 class="bench-title">{{ content.hero.title }}</h1>
          <p class="bench-lead">{{ content.hero.lead }}</p>
          <p class="bench-sub">{{ content.hero.sub }}</p>
        </Stack>
      </template>

      <!-- Weft has no <Image> yet: bench/tools/images.mjs made this WebP from
           bench/image/hero.jpg the way Astro's <Image width={1600}> does, and
           the attributes are the ones Astro's <Image> writes. -->
      <img
        class="bench-image"
        src="/hero.webp"
        :alt="content.image.alt"
        :width="content.image.width"
        :height="content.image.height"
        loading="lazy"
        decoding="async"
      />

      <Section title="Features">
        <div class="bench-grid">
          <Card v-for="f in content.features" :key="f.title" as="article">
            <template #header>{{ f.title }}</template>
            <p class="bench-muted">{{ f.text }}</p>
          </Card>
        </div>
      </Section>

      <Section title="Pricing">
        <div class="bench-grid">
          <Card v-for="p in content.pricing" :key="p.name" as="article">
            <template #header>{{ p.name }}</template>
            <p class="bench-price">{{ p.price }} <span class="bench-muted">{{ p.period }}</span></p>
            <ul class="bench-list">
              <li v-for="item in p.items" :key="item">{{ item }}</li>
            </ul>
          </Card>
        </div>
      </Section>

      <Section :title="content.counter.title" :description="content.counter.description">
        <Island when="idle" :component="Counter" :props="{ start: 0 }" />
      </Section>

      <footer class="bench-footer bench-muted">{{ content.footer }}</footer>
    </Page>
  </div>
</template>

<style>
.bench { --weft-page-max: 960px; min-height: 100vh; }
.bench-tag {
  font: 0.75em/1.6 var(--weft-font-mono);
  padding: 2px 10px;
  border: 1px solid currentColor;
  border-radius: 999px;
}
.bench-title { font-size: clamp(40px, 8vw, 72px); font-weight: var(--weft-weight-title); letter-spacing: -0.03em; line-height: 1; }
.bench-lead { font-size: 1.35em; line-height: 1.35; }
.bench-muted { color: var(--weft-color-text-muted); }
.bench-image { display: block; width: 100%; height: auto; border-radius: var(--weft-radius-control); }
.bench-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(min(100%, 220px), 1fr));
  gap: var(--weft-space-4);
}
.bench-price { font-size: 2em; font-weight: var(--weft-weight-title); }
.bench-price .bench-muted { font-size: 0.4em; font-weight: normal; }
.bench-list { padding-left: 1.2em; line-height: 1.8; }
.bench-footer { border-top: 1px solid var(--weft-color-border); padding-top: var(--weft-space-6); font-size: 0.85em; }
</style>
