<script setup vapor lang="ts">
// The /components filter box, an island: as you type it hides the cards of
// the gallery whose name and purpose do not hold every word, and a group
// with none left. The list itself is the page's HTML, so without
// JavaScript it is all there; the box is inert until this island starts.
import { onMounted, ref, watch } from 'vue'
import { Input } from '@weft/ui'

const query = ref('')
const mounted = ref(false)
const shown = ref<number | null>(null)
onMounted(() => (mounted.value = true))

watch(query, (q) => {
  const words = q.toLowerCase().split(/\s+/).filter(Boolean)
  let count = 0
  for (const group of document.querySelectorAll<HTMLElement>('[data-gallery-group]')) {
    let any = false
    for (const card of group.querySelectorAll<HTMLElement>('[data-gallery-card]')) {
      const text = card.dataset.galleryCard ?? ''
      const match = words.every((w) => text.includes(w))
      card.hidden = !match
      if (match) any = true
      if (match) count++
    }
    group.hidden = !any
  }
  shown.value = words.length ? count : null
})
</script>

<template>
  <div class="gallery-filter" :inert="!mounted" :data-ready="mounted ? 'true' : undefined">
    <Input v-model="query" type="search" icon="search" placeholder="Filter components" aria-label="Filter components" />
    <p class="gallery-filter__count" aria-live="polite">{{ shown === null ? '' : shown === 0 ? 'No component matches.' : `${shown} shown.` }}</p>
  </div>
</template>
