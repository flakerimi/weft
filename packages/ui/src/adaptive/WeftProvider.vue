<script setup vapor lang="ts">
import { computed, provide } from 'vue'
import { useWeft, weftKey } from './context.ts'
import type { Input, Surface } from './detect.ts'

const props = withDefaults(defineProps<{ input?: Input; surface?: Surface; as?: string }>(), {
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
