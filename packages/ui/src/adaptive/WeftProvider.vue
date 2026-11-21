<script setup vapor lang="ts">
import { computed, provide } from 'vue'
import { useWeft, weftKey } from './context.ts'
import type { Input, Surface } from './detect.ts'
import '../styles/base.css'
import '../styles/weft-provider.css'

const props = withDefaults(defineProps<{
  /** Force `pointer` or `touch` for everything inside; unset passes on the detected or outer input. */
  input?: Input
  /** `landing` or `app` density for everything inside; unset passes on the outer surface. */
  surface?: Surface
  /** The BCP 47 locale (`de-DE`, `ar-EG`) for dates and sizes inside: names, digits, the first day of the week; unset passes on the outer one. */
  locale?: string
  /** The element to render; it has `display: contents`, so it does not affect layout. */
  as?: string
  /**
   * Everything inside is a preview of a page, drawn inside another (a
   * gallery's stage or thumbnail): a Page or an AppShell in it draws no
   * `<main>` landmark, a Page's title is an h2, and an AppShell keeps no
   * collapsed state. An outer inset provider
   * makes every one inside it inset too.
   */
  inset?: boolean
}>(), {
  as: 'div',
})

const parent = useWeft()
const input = computed(() => props.input ?? parent.input.value)
const surface = computed(() => props.surface ?? parent.surface.value)
const locale = computed(() => props.locale ?? parent.locale.value)
const inset = computed(() => props.inset || parent.inset.value)
provide(weftKey, { input, surface, locale, inset })
// Only a forced input goes on the element: the detected one is CSS's own
// `(pointer: coarse)` at the root, right before and after hydrating.
</script>

<template>
  <component :is="as" class="weft-provider" :data-weft-input="props.input" :data-weft-surface="surface">
    <slot />
  </component>
</template>
