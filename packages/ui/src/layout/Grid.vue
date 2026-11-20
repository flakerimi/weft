<script setup vapor lang="ts">
import { computed, useTemplateRef } from 'vue'
import { useWidth } from '../adaptive/useWidth.ts'
import type { Gap } from './types.ts'

type Cols = number | { narrow?: number; regular?: number; wide?: number }

const props = withDefaults(defineProps<{ cols?: Cols; gap?: Gap }>(), {
  cols: () => ({ narrow: 1, regular: 2, wide: 3 }),
  gap: 4,
})

const root = useTemplateRef<HTMLElement>('root')
const width = useWidth(root)
const order = ['narrow', 'regular', 'wide'] as const
const defaults = { narrow: 1, regular: 2, wide: 3 }
// A partial object falls back to the nearest smaller width given, else the defaults.
const count = computed(() => {
  if (typeof props.cols === 'number') return props.cols
  const c = props.cols
  for (let i = order.indexOf(width.value); i >= 0; i--) {
    const n = c[order[i]!]
    if (n !== undefined) return n
  }
  return defaults[width.value]
})
</script>

<template>
  <div ref="root" class="weft-grid" :data-gap="gap" :data-width="width" :style="{ '--weft-grid-cols': count }">
    <slot />
  </div>
</template>
