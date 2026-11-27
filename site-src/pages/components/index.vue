<script setup vapor lang="ts">
// /components: every component in the catalogue, by group, each a card with
// its first example drawn live by the server at its own size on a white
// stage (inert, and with no script: an island in an example is drawn in
// place), its name and its purpose. A
// filter box (an island) narrows the cards as you type;
// without JavaScript every card shows. The page also writes the catalogue
// as it is, at /catalog.json, for agents.
import { Island, Page, useHead } from '@weft/ui'
import { useFile } from 'virtual:weft/files'
import '../../styles/fonts.css'
import '../../styles/shell.css'
import '../../styles/gallery.css'
import '../../styles/gallery-index.css'
import { catalogText, components, groups, parityText, sizesText } from '../../gallery/catalog.ts'
import GalleryFilter from '../../components/gallery/GalleryFilter.vue'
import GalleryNav from '../../components/gallery/GalleryNav.vue'
import GalleryTile from '../../components/gallery/GalleryTile.vue'
import SiteFooter from '../../components/SiteFooter.vue'
import SiteHeader from '../../components/SiteHeader.vue'

const description = `${components.length} Vapor components for Vue that adapt to touch and pointer, landing and app, and the width they get.`
useHead({ title: 'Components — Weft', description })
useFile('/catalog.json', catalogText)
useFile('/components/parity.md', parityText)
useFile('/components/sizes.md', sizesText)
</script>

<template>
  <div class="site">
    <SiteHeader current="components" path="/components" menu="components-nav" />
    <div class="docs">
      <GalleryNav />
      <Page id="main" class="docs-page gallery-page gallery-index">
        <template #header>
          <h1 class="docs-title">Components</h1>
          <p class="docs-lead">{{ description }}</p>
          <p class="gallery-links">
            How they compare with Nuxt UI: <a href="/components/parity.md">parity</a>. What each costs a page:
            <a href="/components/sizes.md">sizes</a>. The whole catalogue, for agents: <a href="/catalog.json">catalog.json</a>.
          </p>
          <Island when="idle" :component="GalleryFilter" />
        </template>
        <section v-for="g in groups" :key="g.group" class="gallery-group" :aria-labelledby="`group-${g.group}`" data-gallery-group>
          <h2 :id="`group-${g.group}`">{{ g.label }}</h2>
          <ul class="gallery-cards">
            <GalleryTile v-for="c in g.components" :key="c.name" :component="c" :scope="`t${components.indexOf(c)}`" data-gallery-card />
          </ul>
        </section>
      </Page>
    </div>
    <SiteFooter />
  </div>
</template>
