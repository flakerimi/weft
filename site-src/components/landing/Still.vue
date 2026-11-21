<script setup vapor lang="ts">
// A picture of an app on the landing page, drawn once by the server from the
// real components and never hydrated: inert (nothing in it can be used or
// focused, and assistive technology skips it, so the page says in words what
// it shows), kept from the speculation rules' prefetching, and marked
// data-site-picture, so the site's own heading style stays out. Its slot has
// an id scope of its own, as an island has, so ids stay unique on the page.
// It is a preview (inset): no main landmark or h1 of its own. `input` and
// `width` draw it as a pointer or a touch screen of a given width.
import { provide } from 'vue'
import { ID_SCOPE, WeftProvider, type IdScope } from '@weft/ui'

const props = defineProps<{
  /** The scope's prefix, unique on the page: its ids are <name>-<scope>-<n>. */
  scope: string
  /** The input it is drawn for. */
  input?: 'pointer' | 'touch'
  /** The width class it is drawn at. */
  width?: 'narrow' | 'regular' | 'wide'
}>()
provide<IdScope>(ID_SCOPE, { prefix: props.scope, next: 0 })
</script>

<template>
  <div inert data-weft-no-prefetch data-site-picture>
    <WeftProvider inset surface="app" :input="input" :width="width"><slot /></WeftProvider>
  </div>
</template>
