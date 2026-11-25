<script setup lang="ts">
// The benchmark's landing, the same content as bench/weft/pages/index.vue,
// as a Nuxt page: Nuxt hydrates the whole page, so there is no island.
import content from '../../../content.json'

useHead({
  title: content.title,
  meta: [
    { name: 'description', content: content.description },
    { property: 'og:title', content: content.title },
    { property: 'og:description', content: content.description },
  ],
})
</script>

<template>
  <main class="page">
    <header class="hero">
      <span class="tag">{{ content.hero.tag }}</span>
      <h1 class="title">{{ content.hero.title }}</h1>
      <p class="lead">{{ content.hero.lead }}</p>
      <p class="sub">{{ content.hero.sub }}</p>
    </header>

    <!-- The WebP bench/tools/images.mjs makes, with the attributes Astro's
         <Image> writes, as on the Weft page. -->
    <img
      class="image"
      src="/hero.webp"
      :alt="content.image.alt"
      :width="content.image.width"
      :height="content.image.height"
      loading="lazy"
      decoding="async"
    />

    <section>
      <h2>Features</h2>
      <div class="grid">
        <article v-for="f in content.features" :key="f.title" class="card">
          <h3>{{ f.title }}</h3>
          <p class="muted">{{ f.text }}</p>
        </article>
      </div>
    </section>

    <section>
      <h2>Pricing</h2>
      <div class="grid">
        <article v-for="p in content.pricing" :key="p.name" class="card">
          <h3>{{ p.name }}</h3>
          <p class="price">{{ p.price }} <span class="muted">{{ p.period }}</span></p>
          <ul class="list">
            <li v-for="item in p.items" :key="item">{{ item }}</li>
          </ul>
        </article>
      </div>
    </section>

    <section>
      <h2>{{ content.counter.title }}</h2>
      <p class="muted">{{ content.counter.description }}</p>
      <Counter :start="0" />
    </section>

    <footer class="footer muted">{{ content.footer }}</footer>
  </main>
</template>
