<script setup vapor lang="ts">
import { computed, provide } from 'vue'
import { useWeft, weftKey } from './context.ts'
import type { Input, Surface } from './detect.ts'
import '../styles/base.css'

const props = withDefaults(defineProps<{
  /** Force `pointer` or `touch` for everything inside; unset passes on the detected or outer input. */
  input?: Input
  /** `landing` or `app` density for everything inside; unset passes on the outer surface. */
  surface?: Surface
  /** The element to render; it has `display: contents`, so it does not affect layout. */
  as?: string
}>(), {
  as: 'div',
})

const parent = useWeft()
const input = computed(() => props.input ?? parent.input.value)
const surface = computed(() => props.surface ?? parent.surface.value)
provide(weftKey, { input, surface })
</script>

<template>
  <component :is="as" class="weft-provider" :data-weft-input="input" :data-weft-surface="surface">
    <slot />
  </component>
</template>
