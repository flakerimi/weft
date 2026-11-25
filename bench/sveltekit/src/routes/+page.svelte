<script>
  // The benchmark's landing, the same content as bench/weft/pages/index.vue,
  // as a SvelteKit page: SvelteKit hydrates the whole page, so there is no
  // island.
  import content from '../../../content.json'
  import Counter from '../lib/Counter.svelte'
</script>

<svelte:head>
  <title>{content.title}</title>
  <meta name="description" content={content.description} />
  <meta property="og:title" content={content.title} />
  <meta property="og:description" content={content.description} />
</svelte:head>

<main class="page">
  <header class="hero">
    <span class="tag">{content.hero.tag}</span>
    <h1 class="title">{content.hero.title}</h1>
    <p class="lead">{content.hero.lead}</p>
    <p class="sub">{content.hero.sub}</p>
  </header>

  <!-- The WebP bench/tools/images.mjs makes, with the attributes Astro's
       <Image> writes, as on the Weft page. -->
  <img
    class="image"
    src="/hero.webp"
    alt={content.image.alt}
    width={content.image.width}
    height={content.image.height}
    loading="lazy"
    decoding="async"
  />

  <section>
    <h2>Features</h2>
    <div class="grid">
      {#each content.features as f (f.title)}
        <article class="card">
          <h3>{f.title}</h3>
          <p class="muted">{f.text}</p>
        </article>
      {/each}
    </div>
  </section>

  <section>
    <h2>Pricing</h2>
    <div class="grid">
      {#each content.pricing as p (p.name)}
        <article class="card">
          <h3>{p.name}</h3>
          <p class="price">{p.price} <span class="muted">{p.period}</span></p>
          <ul class="list">
            {#each p.items as item (item)}
              <li>{item}</li>
            {/each}
          </ul>
        </article>
      {/each}
    </div>
  </section>

  <section>
    <h2>{content.counter.title}</h2>
    <p class="muted">{content.counter.description}</p>
    <Counter start={0} />
  </section>

  <footer class="footer muted">{content.footer}</footer>
</main>
