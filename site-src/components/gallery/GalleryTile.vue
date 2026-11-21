<script setup vapor lang="ts">
// A card of the gallery: the component's first example drawn by the server
// at its own size on a white stage (inert, and with no script: an island in
// it is drawn in place), then its name, a link over the whole card to its
// page, and its purpose: one of the cards of the /components index.
import type { GalleryComponent } from '../../gallery/catalog.ts'
import { inlineHTML } from '../../gallery/model.ts'
import StillPreview from './StillPreview.vue'

defineProps<{
  component: GalleryComponent
  /** The id scope of its example, unique on the page. */
  scope: string
}>()
</script>

<template>
  <li class="gallery-card">
    <div class="gallery-thumb" inert data-weft-no-prefetch data-site-picture :data-component="component.name" :data-example="component.examples[0]!.file">
      <StillPreview :scope="scope"><component :is="component.examples[0]!.component" /></StillPreview>
    </div>
    <h3><a :href="component.path">{{ component.name }}</a></h3>
    <p v-html="inlineHTML(component.purpose)" />
  </li>
</template>
