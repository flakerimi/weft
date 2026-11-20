<script setup vapor lang="ts">
import { useSSRContext, type Component } from 'vue'

export type IslandWhen = 'load' | 'idle' | 'visible' | 'media'

const p = defineProps<{
  when: IslandWhen
  media?: string
  component: Component
  props?: Record<string, unknown>
}>()

// The page is rendered on the server only; in the browser the island runtime
// hydrates <weft-island> on its own, so this component renders nothing there.
const server = import.meta.env.SSR

// The island's id is the component's SFC `__name` (its file name), the key the
// weft() plugin gives its chunk in the registry. A `name` option could differ
// from the file name, so it is not used.
const id = (p.component as { __name?: string }).__name
let json = ''
if (server) {
  if (!id) throw new Error('Island: the component has no SFC __name; an island must be a <script setup> single-file component.')
  if (p.when === 'media' && !p.media) throw new Error('Island: when="media" needs a media query in the media prop.')
  // Escape what could end the <script> element or open a comment inside it.
  json = JSON.stringify(p.props ?? {})
    .replace(/</g, '\\u003c')
    .replace(/>/g, '\\u003e')
    .replace(/&/g, '\\u0026')
  const ctx = useSSRContext<{ islands?: string[] }>()
  if (ctx) (ctx.islands ??= []).push(id)
}
</script>

<template>
  <component is="weft-island" v-if="server" :data-id="id" :data-when="when" :data-media="media">
    <component is="script" type="application/json" :innerHTML="json" />
    <component :is="component" v-bind="p.props" />
  </component>
</template>
