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

// The island's id is the component's SFC name (its file name); the island
// runtime keys its chunk registry by the same name.
const named = p.component as { __name?: string; name?: string }
const id = named.__name ?? named.name
let json = ''
if (server) {
  if (!id) throw new Error('Island: the component has no name; give it a file of its own.')
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
