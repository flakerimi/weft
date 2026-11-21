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
}>(), {
  as: 'div',
})

const parent = useWeft()
const input = computed(() => props.input ?? parent.input.value)
const surface = computed(() => props.surface ?? parent.surface.value)
const locale = computed(() => props.locale ?? parent.locale.value)
provide(weftKey, { input, surface, locale })
</script>

<template>
  <component :is="as" class="weft-provider" :data-weft-input="input" :data-weft-surface="surface">
    <slot />
  </component>
</template>
